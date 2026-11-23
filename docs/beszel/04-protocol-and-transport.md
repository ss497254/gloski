# 04 — Protocol and transport

Beszel uses WebSocket + CBOR with an SSH fallback; Gloski uses one bidirectional
gRPC stream. **Do not port the transport.** The value here is the connection
lifecycle rules, deadline arithmetic, identity handshake and realtime fan-out —
all of which were learned from production incidents and are visible in
`supplemental/CHANGELOG.md`.

## Wire envelope

`internal/common/common-ws.go` (93 LOC) defines a generic request/response
envelope with integer CBOR keys:

```go
const (
    GetData WebSocketAction = iota  // 0
    CheckFingerprint                // 1
    GetContainerLogs                // 2
    GetContainerInfo                // 3
    GetSmartData                    // 4
    GetSystemdInfo                  // 5
    GetZfsData                      // 6
    SyncNetworkMonitors             // 7
    GetPackageUpdates               // 8
    GetSystemdLogs                  // 9
)

type HubRequest[T any] struct {
    Action WebSocketAction `cbor:"0,keyasint"`
    Data   T               `cbor:"1,keyasint,omitempty,omitzero"`
    Id     *uint32         `cbor:"2,keyasint,omitempty"`
}

type AgentResponse struct {
    Id          *uint32     `cbor:"0,keyasint,omitempty"`
    SystemData  *CombinedData `cbor:"1,..."` // legacy <= 0.17
    Fingerprint *FingerprintResponse `cbor:"2,..."` // legacy
    Error       string      `cbor:"3,keyasint,omitempty,omitzero"`
    String      *string     `cbor:"4,..."` // legacy
    SmartData   map[string]smart.SmartData `cbor:"5,..."` // legacy
    ServiceInfo systemd.ServiceDetails `cbor:"6,..."` // legacy
    Data        cbor.RawMessage `cbor:"7,..."` // generic payload, 0.18+
    SmartComplete bool        `cbor:"8,keyasint,omitempty,omitzero"`
}
```

Notable design points (C):

- A **generic `bytes` payload field (7)** lets new endpoints be added without
  changing the envelope; typed legacy slots 1–6 are kept for old hubs.
- Every response has a request `Id` and an `Error` string, so one socket
  multiplexes many concurrent requests.
- `FingerprintRequest` carries a `Signature []byte` plus `NeedSysInfo`; the
  response carries fingerprint plus optional hostname/port/name for
  auto-registration.
- `DataRequestOptions{CacheTimeMs uint16, IncludeDetails bool}` lets the caller
  ask for a specific cache tier — the hub can request 1-second realtime data or
  a normal 60-second poll on the same connection.

Gloski's protobuf oneof already provides the typed envelope. Borrow only the
**generic payload escape hatch** and the **cache-time request option** if the
stats message starts churning.

## Connection lifecycle

1. Agent opens `ws[s]://hub/api/beszel/agent-connect` with headers `X-Token`
   (≤64 chars) and `X-Beszel` (semver). (`agent/client.go:210-251`)
2. Hub resolves the token: 1-hour expiry map of universal tokens →
   `universal_tokens` table → `fingerprints` table. No record → 401.
   (`internal/hub/agent_connect.go:67-84`)
3. **Mutual auth challenge:** the hub holds an Ed25519 keypair at
   `<data_dir>/id_ed25519` (0600, cached signer, `internal/hub/hub.go:160-216`).
   It signs the **token string** and sends `CheckFingerprint`. The agent verifies
   the signature against **all configured public keys** (`KEY`/`KEY_FILE`,
   multiple allowed), which authenticates the hub and supports key rotation
   without agent downtime. (`agent/client.go:393-405`)
4. Agent fingerprint: `sha256(host.HostID() or fallback)[:24]` hex, persisted in
   `<data_dir>/fingerprint`, rejecting a known-bad product UUID.
5. Registration decision tree (`agent_connect.go:173-264`):
   - one fingerprint record with empty fingerprint → set it on first connect;
   - one record with mismatched fingerprint → **reject**;
   - multiple records or a universal token → match by fingerprint, else
     **auto-create a device record** from agent-supplied hostname/port/name
     and remote IP.
6. On connect, the hub resets SMART/ZFS cooldowns, flags network monitors for
   sync and starts the updater.

**Take for Gloski (C):** the "reject on mismatch instead of silently creating a
duplicate" rule and the key-ring verification pattern. Gloski still keys devices
by name despite storing `machine_id`; the fingerprint tree is a good model for
closing that gap.

## Deadline and timeout constants

These are the most transferable numbers in the codebase.

| Constant | Value | Reason |
|---|---|---|
| Hub read deadline | **100 s** | Must outlast a full failed poll: 60 s interval + 30 s fetch + 10 s margin. A shorter deadline expires mid-collection, before the failed fetch can ping. |
| Agent-side deadline | **120 s** | Agent drops its own connection if a collection overruns. |
| Data request budget | **30 s** | Slow `zpool list` needs more than the 5 s default request timeout. |
| Default pending-request TTL | 5 s | When a caller supplies no deadline. |
| Max agent silence | **3 min** | After this, stop extending the deadline; a genuinely dead socket is allowed to close. |
| Down-state delay | **5 s** | Wait for reconnect before marking a system down. Uses `weak.Pointer` so removed systems are not retained. |
| Poll interval | 60 s | |
| Startup stagger | `interval/N`, capped at 2 s | Avoid hub overload at boot. |
| WS first-fetch jitter | 51–95% of interval | Spread load. |
| SSH connect / handshake / operation | 4 s / 10 s / 20 s | Keepalive 30 s, one retry. Handshake deadline guards banner-withheld attacks. |
| Agent reconnect ticker | 10 s | Kept armed while disconnected. |
| Connect throttle | 5 s | Prevent duplicate in-flight dials. |
| Health ticker | 90 s | |

## Slow collectors and stuck I/O — the key pattern

Documented in `internal/hub/ws/ws.go`, `internal/hub/systems/system.go` and
`ws_data_timeout_test.go`:

- On a fetch error, a live WebSocket is **not** closed. The hub calls
  `WsConn.Ping()`, which extends the read deadline **without requiring a reply**,
  but only while `TimeSinceLastMessage() < 3min`. Past that, pings stop and the
  100 s deadline closes the likely-dead socket.
- A `context.DeadlineExceeded` from a slow WS fetch is **never** treated as a
  broken connection: the error is returned without closing the socket. All other
  errors close it and fall back to SSH.
- An SSH operation that hangs is run in a goroutine with
  `runWithTimeout(timeout, op, onTimeout)`; on timeout `onTimeout` closes the SSH
  client, releasing all blocked reads/writes, and returns a retryable error so
  the next tick re-dials.
- `closeOnCancellation` uses `context.AfterFunc` and a `done` channel so the
  cancellation close cannot outlive the operation.
- `CloseClient`/`Connect` use a generation guard: compare-and-clear only if
  `t.client == client`, so a late close never discards a replacement connection.

**Apply to Gloski (C):** gRPC keepalive currently detects a dead connection, but
there is no equivalent "slow collector must not look like a dead agent" rule.
When implementing roadmap step 1.3, separate three states: stream alive and
collecting, stream alive but collection slow/failed, stream dead. The diagnose
chain's "connected but stale" wording should follow the same distinction. Also
adopt the "close to unblock" guard in `internal/provisioning` SSH operations.

## WS→SSH fallback — do not port

`System.request()` tries WS, then decides via `shouldFallbackToSSH(err)` whether
to fall back. The agent mirrors this with a three-state machine
(`Disconnected`/`WebSocketConnected`/`SSHConnected`). Gloski's owner decision is
explicit: **no SSH fallback for actions**. Do not bring this over. The reusable
parts are the state-machine discipline, coalesced change channels and race
guards listed above.

## Browser realtime fan-out

Beszel rides PocketBase's SSE broker, with ordinary collection subscriptions for
changes and a custom `rt_metrics` topic for 1-second metrics. Portable mechanics
(B):

- Lazily start a worker on the first subscriber; stop it when the count reaches
  zero; use a generation-based stop channel so an old worker cannot stop a new one.
- `claimRealtimeFetches()` snapshots subscribers and marks them in-flight, so
  **slow agents are skipped on later ticks** rather than accumulating backlog.
- `finishRealtimeFetch` releases only the exact subscription pointer claimed,
  making delete/recreate races safe.
- Before each broadcast, refresh the system's user membership from the DB and
  re-check every client, because custom topics bypass collection rules.
- Frontend uses a `cancelled` flag plus deferred unsubscribe so a subscription
  confirming after unmount is immediately torn down.

Gloski's `/api/fleet/ws` and `.../stats/ws` already cover the transport; take the
skip-slow, generation and cancelled-unsubscribe ideas only.

## Version gates

`beszel.go` (27 LOC) defines named minimum versions rather than a single global
protocol version: `MinVersionCbor` 0.12.0, `MinVersionAgentResponse` 0.13.0,
`MinVersionZfsData` 0.18.9, `MinVersionNetworkMonitors` 0.20.0,
`MinVersionMonitorIPVersion` 0.22.0. The frontend hides chart ranges whose
`minVersion` exceeds the connected agent's version.

Gloski has one `ProtocolVersion` (4, raised for native check probes). Keep it
while features are few; split it the same way when a new capability needs an
older-agent fallback rather than a full reinstall.

## Credentials, TLS and proxy details (C)

- Token from `TOKEN` or `TOKEN_FILE` (blank lines and comments skipped; exactly
  one token — multiple rejected because there is one outbound hub).
- Public keys from `KEY` or `KEY_FILE`; parsed into a **slice** for hub key rotation.
- `CA_CERT_FILE` appends a PEM to the system cert pool.
- `HUB_HEADERS` parses `Name: value` pairs with
  `httpguts.ValidHeaderFieldName/Value` and **never echoes secret values in
  errors**; reserved header names are ignored with a warning.
- The proxy dialer is bound to the agent lifetime context, so a replacement
  connection does not leak the old socket.
- `EXIT_ON_DNS_ERROR` exits after 2 s for container restart policies.

Take the secret-safe error messages and lifetime-bound dialing as habits; the
token model itself is not applicable to Gloski.
