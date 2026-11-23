# 01 — Design and architecture

This document compares the two products at the design level and extracts the
architectural patterns worth taking. It is intentionally about *shape* and
*decisions*, not code; the code is in the topic documents.

## The two products

| | Gloski | Beszel |
|---|---|---|
| Purpose | Control tower: run jobs, open terminals, manage files, inspect processes, repair agents | Monitoring platform: collect metrics, keep history, graph them, alert |
| Centre of gravity | Remote **action** | Observability |
| Master/hub | `gloski`: net/http, gorilla/websocket, gRPC listeners, SQLite, embedded agent builds | `beszel hub`: PocketBase app with cron, collections, rules, realtime broker, admin UI |
| Agent | `gloski-agent` (package `worker`): stateless, executes jobs, owns PTYs and file transfers | `beszel-agent`: stateless, collects metrics, hosts an SSH fallback server |
| Transport | One bidirectional gRPC `Agent.Connect` stream; unix socket locally, h2c on Tailnet | WebSocket + CBOR, SSH fallback; one request/response per action |
| Browser | React SPA → master HTTP/WS only | React SPA → PocketBase REST + SSE + custom `rt_metrics` topic |
| Identity | Name-keyed device with `machine_id` guard | Token + persisted fingerprint, auto-registration from universal tokens |
| Lifecycle | Master installs/restarts over SSH; one general repair | Agent self-updates via GitHub releases, init-system restart, many platform paths |
| Notifications | None, by decision | Alerts, quiet hours, shoutrrr, email |
| Users | Single optional API key/JWT | Multi-user, roles, OAuth/OIDC, sharing |

**The overlap is the hub-and-agent shape**, which is the obvious architecture
for any fleet tool. The feature sets diverge sharply: Beszel has no jobs,
terminal, file manager, process control, schedules or run-on-many. Gloski has no
native metric collectors, history, alerts or multi-user.

## What each design optimises for

Beszel optimises for **breadth of hardware coverage and low operating cost**.
That explains the enormous collector tree, the six container image variants,
the 30+ release targets, and the many init systems: it must work on NAS boxes,
OpenWrt routers, FreeBSD, pfSense, Raspberry Pis and cloud VMs, usually installed
by the user. It also explains PocketBase: it gives auth, an admin UI, realtime,
backups and a query API for free.

Gloski optimises for **reliability of action**. Every device is deployed from the
dashboard over SSH; there is one repair (reinstall); no device-local supervisor
is trusted. That is why the master owns the lifecycle and why the agent is
deliberately disposable.

Importantly, **the two optimisation targets conflict**. Adopting Beszel's
self-updating, multi-init, package-manager-based agent would undo the master-owned
lifecycle. Adopting PocketBase would replace the API, DB and auth layer. The goal
is therefore selective borrowing, not convergence.

## Side-by-side component maps

Beszel (simplified):

```
agent/
  agent.go            god-struct; gatherStats with cache intervals
  client.go           WebSocket client, auth challenge, CBOR
  connection_manager.go  WS <-> SSH state machine
  {cpu,disk,gpu,smart,zfs,btrfs,sensors,battery,docker,...}.go  collectors
  network_monitor_*.go   active probes, bounded history
  deltatracker/       generic rate calculator
internal/
  hub/                PocketBase app, API, agent_connect, realtime, systems
  records/            tiered rollups, retention, averaging
  alerts/             evaluation, history, quiet hours, delivery
  ghupdate/           self-update with checksum and rollback
  entities/           compact CBOR/JSON data model
  site/               React SPA, embedded in the binary
```

Gloski (from `docs/architecture.md`):

```
server/
  internal/controltower/   Tower, routes, listeners, diagnose, lifecycle
  internal/controltower/agents/   master half of the gRPC stream
  internal/fleet/          registry, heartbeats, deployments, event hub
  internal/provisioning/   SSH install/start/stop, scripts, probe
  internal/tailnet/        tailscaled LocalAPI peer discovery
  internal/worker/         device agent: jobs, files, terminal, stats
  internal/system/         collector and stats store
  proto/agentv1/           generated gRPC types
web/src/                   React SPA
```

## Patterns worth adopting at the design level

These are **C — idea** unless stated otherwise. Each is explained with code in a
later document.

1. **Cache interval as a key, not a single timer.** Beszel's agent keeps maps
   keyed by requested cache-time, so a 1-second realtime chart and a 60-second
   dashboard poll can coexist without one starving the other. Gloski's
   heartbeat design C ("stats only while someone is looking, otherwise once a
   minute") is the natural home for this.
2. **Slow work off the request path.** Storage-pool utilities run in background
   with their own cache so a hung `zpool list` never delays a stats response.
   Auxiliary Docker registry checks are excluded from the metrics WaitGroup.
3. **Partial success is representable.** `CompleteBackends` and `SmartComplete`
   flags distinguish "no data" from "not finished collecting". This is the same
   principle as `docs/ideas-from-smdwin.md`'s "unknown is a first-class state".
4. **Wall-clock idle detection.** The agent counts suspend time toward "hub has
   not asked for stats", then pauses probes and expensive sensors. Cheap and
   effective on battery devices.
5. **Named min-version gates per feature** (`MinVersionZfsData`,
   `MinVersionNetworkMonitors`, …) rather than one global protocol version.
   Gloski's single `ProtocolVersion` can stay until features diverge; then
   split it the same way.
6. **Transport type stamped into the payload** (`Info.ConnectionType`) so the UI
   can show how a device is connected without a second query. Gloski's
   equivalent: unix socket vs Tailnet TCP.
7. **Capability checks before changes**, with explicit failure reasons; and
   **named install phases** in every error. Feeds the diagnose work directly.
8. **Bounded resources everywhere**: 61-bucket history rings, 60/100-point chart
   buffers, PTY output buffers, bounded journal reads, capped log frames.
9. **Generation guards** for asynchronous work: compare a counter before and
   after I/O so a stale callback from a previous connection cannot corrupt the
   new one. Used in realtime workers, SSH close paths and suspend detection.
10. **Versioned wire format with reserved slots**: integer keys, "do not reuse"
    comments, deprecated fields retained for one cycle. See
    [03 — Stats and data model](03-stats-data-model.md).

## Systemic things to avoid

- **PocketBase as the hub.** It is the single biggest source of non-portable
  code. Every hub manager takes `core.App`; rules are PB filter strings; cron,
  realtime, auth and backups are PB features. Adopting it would rewrite Gloski.
- **A god object on the device.** Beszel's own `agent.go` carries a TODO saying
  it is confusing. Gloski's `Tower` and `FilesContext` have the same problem
  (`docs/code-quality-plan.md`); do not import another one.
- **An alert mechanism per failure case.** The owner wants one general repair.
  Beszel's alert manager, quiet hours, incident state and notification services
  are a large machine for something Gloski deliberately does not do.
- **A combined health score.** Beszel's changelog records how hardware-database
  coverage dragged a clean run to 43/100. Roadmap says never a combined pass/fail.
- **Matrix sprawl.** Six container variants, Helm charts, Debian packages,
  Homebrew/Scoop/WinGet, 30+ GoReleaser targets. Gloski cross-builds three
  architectures and embeds them; keep that.
- **Security machinery that blocks legitimate use.** Beszel blocks private and
  CGNAT destinations in notification delivery. Gloski is internal; do not add
  equivalent guards unprompted, and do not remove existing protections without
  asking.

## Decision matrix applied to each Beszel subsystem

| Subsystem | Decision | Reason |
|---|---|---|
| Agent collectors | Selective; only after a real need | Big surface, platform-specific, shell checks cover most of it |
| Network probes | Take | New capability, self-contained, fits checks roadmap |
| Stats rollup/retention | Take the algorithm | Needed for history; storage can stay SQLite |
| Deadline/silence/reconnect rules | Take as ideas | Reliability first; no new transport |
| Frontend utilities/charts/tables | Take aggressively | Same stack; largest code reuse |
| Self-update pipeline | Take the verification/staging discipline | Master-owned lifecycle stays |
| Install script safety | Take; skip service-manager half | `/tmp` installs and master-owned lifecycle stay |
| Auth/multi-user/OAuth/alerts | Skip | Explicitly out of scope |
| MCP façade | Park | Low effort later, no current need |
| Container/Helm/packaging matrix | Skip | Conflicts with "keep it simple" |
