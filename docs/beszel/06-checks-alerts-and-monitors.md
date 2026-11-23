# 06 — Checks, alerts and monitors

This is the most valuable product-level borrowing area. Beszel's network
monitors are a complete, self-contained subsystem that fits Gloski's roadmap
step 3 (checks) without any of the alert/notification machinery Gloski does not
want. The alert engine is worth reading for its evaluation rules but should be
treated as **C — idea only**.

## Network monitors — subsystem inventory

| File | LOC | Role | Class |
|---|---:|---|---|
| `internal/entities/monitor/monitor.go` | 151 | `Config`, `Result`, `CertInfo`, sync request/response | B |
| `internal/hub/network_monitors.go` | 237 | Stable-ID generation and record hooks | B |
| `internal/hub/systems/network_monitors.go` | 106 | Hub sync RPC wrappers, version gating | B |
| `agent/network_monitor.go` | 219 | Task lifecycle, full/incremental sync, hub-idle pause, `runNow` | B |
| `agent/network_monitor_schedule.go` | 66 | Staggered serial scheduling | **A** |
| `agent/network_monitor_task.go` | 183 | In-flight dedup, generation guard, cert one-shot | **A** |
| `agent/network_monitor_probe.go` | 246 | ICMP/TCP/HTTP/DNS probes | **A** |
| `agent/network_monitor_ping.go` | 311 | ICMP fallback chain and reply correlation | **A** |
| `agent/network_monitor_history.go` | 274 | Two-tier bounded history | **A** |
| `agent/network_monitor_cert.go` | 75 | TLS leaf expiry check | **A** |
| `agent/network_monitor_resume.go` | 88 | Suspend/clock-jump guard | C |
| `network_monitors_test.go` + `system/network_monitor_*_test.go` | ~1,800 | Sync, stats, auth, SSH coverage | reference |

## Probe types and concrete rules

`agent/network_monitor_probe.go` defines
`monitorProbe func(ctx, Config) (int64 /* microseconds */, error)` for four
protocols:

- **icmp**: raw socket → datagram socket → exec `ping` fallback, detected once
  per address family and cached. A 16-byte random token plus sequence number
  correlates replies even when the sequence wraps; Linux ping-socket ID is
  mapped to a port. Localised Windows RTT strings are parsed.
- **tcp**: DNS resolution happens **outside** the timing window. The remaining
  deadline is shared across resolved IPs by division
  (`time.Until(deadline) / (len(ips)-i)`), so a dead first address cannot consume
  the whole budget.
- **http**: GET with configurable accepted status codes/ranges such as
  `200-299,401`; empty means `<400`; **unparseable entries match nothing** so a
  bad config fails closed rather than hiding an outage. Six prebuilt clients
  cover IP-version × TLS-mode combinations.
- **dns**: custom resolver to a given `server:53`, A or AAAA by IP version.

**Timeouts:** minimum interval 30 s; TCP and DNS probes 3 s; HTTP 10 s;
`RunNow` sync budget 5 s + 10 s probe.

## Scheduling and idle suppression

`agent/network_monitor_schedule.go` (66 LOC) is short enough to study directly:

- Initial delay = random **50–100%** of the interval (`getStagger`).
- One timer for the first run, then one ticker.
- Checks run **serially**, so slow checks naturally drop missed ticks instead of
  building an execution backlog.
- `pm.hubIdle()` gates every tick: after **3 minutes** with no stats request
  from the hub, scheduled probes stop. Immediate `runNow` probes still run.
- A resume guard snapshots a generation counter before and after I/O; if it
  changed (suspend or clock jump), the sample is discarded rather than recorded
  with a bogus latency.

The idle rule maps directly onto Gloski's heartbeat design C ("stats only while
someone is looking at the device, otherwise once a minute"). Adopt it before
adding any scheduled work to the agent.

## Bounded history and result shape

`agent/network_monitor_history.go` (274 LOC), class **A**:

- **61 raw samples covering the last second-level window** plus **61 wall-clock
  one-minute buckets** — constant memory for roughly an hour.
- Short realtime windows with no samples fall back to the most recent sample, so
  live charts never show a blank gap.
- Aggregates: sum/min/max/count, response average, packet loss
  `round((total-success)/total*10000)/100` (two decimals).
- `SampleCount` exists specifically for alert warm-up when intervals exceed
  20 minutes — Gloski can ignore that part.
- Buckets merge with `mergeSameTimestamps` on the frontend
  (`src/lib/network-monitor-utils.ts`).

`Result` fields: avg/min/max response in µs, 1h avg/min/max, packet loss % and
1h loss %, `LastProbeAt` in Unix ms, lifetime `SampleCount`, current-window
counts/sum, and `Cert`.

## Stable IDs and the sync protocol

`internal/hub/network_monitors.go:15-33`:

```
ID = fmt.Sprintf("%06s", FNV32a(systemID, target, protocol,
        [port if tcp], [server if dns], [ipVersion if explicit]))
```

Identity fields are deliberately limited. Changing interval, TLS-skip or
accepted status codes updates the same record in place; changing target or
protocol creates a new record and deletes the old one. `%06s` preserves
PocketBase's six-character minimum without altering existing longer IDs.

`SyncRequest{Action: Replace|Upsert|Delete, Config, Configs, RunNow}`:
full replace on every WebSocket connect (flagged `monitorsNeedSync`, retried on
failure), incremental upsert/delete on record hooks. Old agents get no sync;
newer ones silently skip unsupported IP-version configs rather than reporting
misleading results. Monitors belonging to another system are skipped.

**Gloski adaptation:** a single `SyncChecks` message on the gRPC stream, sent on
registration and when the saved-check list changes. Results travel with the
heartbeat as just another field — no new channel.

## TLS certificate checks

`agent/network_monitor_cert.go` (75): hourly TLS dial with
`InsecureSkipVerify: true`, so expired and self-signed certificates are
**reported rather than rejected**; stores leaf `NotAfter` and issuer CN. Delivered
once via a `certUnsent` flag, and only on the normal 60 s stats fetch — the 1 s
realtime fetches must not consume it.

This is directly reusable as a saved check (a shell command can do it, but the
"report, don't reject" semantics and one-shot delivery are worth copying).

## Alert evaluation — ideas only

Gloski does not want alerts, but these rules are useful for the check-result
model in `docs/ideas-from-smdwin.md` §1.

| Pattern | Source | Detail |
|---|---|---|
| Hysteresis | `internal/alerts/alerts_system.go` (452) | Skip when `(not triggered && val<=threshold) || (triggered && val>threshold)`; battery is inverted. |
| Averaging window | same | With duration `min>1`, average `type='1m'` samples from `now - min - 90s` and require `sampleCount >= min/1.2`. `min==1` fires on the current value. |
| Confirm-down timers | `internal/alerts/alerts_status.go` (239) | On down, `AfterFunc(max(1,min) minutes)` keyed by alert ID; cancel on up; re-queue pending timers after restart; re-read state before sending. |
| Pending preserved on address change | `internal/hub/systems/system_manager.go` | A non-up→up transition deliberately keeps the status alert so recovery still fires. |
| Per-target incident IDs | `internal/alerts/alerts_network_monitors.go` (392) | Persist `map[monitorID]historyID` in JSON, so incident history survives restarts and retention. |
| Result readiness | same | A monitor result is alertable only after ≥3 samples and within `min(1h, max(3×interval, 3min))`. |
| Cross-midnight quiet hours | `internal/alerts/alerts.go` | `end < start ⇒ now>=start || now<end`. Only if quiet hours are ever wanted. |
| Notification routing, shoutrrr, SSRF guard | `alerts.go`, `notification_client.go` (150) | **D — skip.** No notifications wanted, and the destination blocking would reject legitimate internal services. |

## Outbound heartbeat / dead-man switch

`internal/hub/heartbeat/heartbeat.go` (303), class **B**:

- Enabled by `HEARTBEAT_URL`; `HEARTBEAT_INTERVAL` default 60 s; method GET,
  HEAD or POST (default POST), HTTP timeout 10 s; immediate send on startup then
  ticker.
- Payload: `{status: ok|warn|error, timestamp, msg, systems:{total,up,down,
  paused,pending}, down_systems[], triggered_alerts[], beszel_version}`.
  Status is `error` if any system is down, else `warn` if alerts are triggered.
- URLs are redacted to `scheme://host` in logs; admin endpoints read config and
  send a test ping.

This is **not** an alert — it is an opt-in "the master is alive" signal to an
external uptime service. It fits the owner's single-point-of-failure concern,
but it is an external dependency. Listed as a decision in the README.

## Mapping to Gloski's roadmap

1. First implement the **check result model** from `docs/ideas-from-smdwin.md` §1
   (status, summary, details, ran-at, stale, skipped) — do not jump straight to probes.
2. Then port `network_monitor_{probe,ping,history,cert,task,schedule}.go` with
   type substitutions: `MonitorResult` becomes a check result; the history ring
   becomes `check_results` history in SQLite.
3. Add idle suppression tied to the existing heartbeat so checks stop when
   nobody is watching.
4. Add stable IDs from the FNV rule for idempotent saved checks.
5. Keep everything result-based and read-only. No notification routing.
