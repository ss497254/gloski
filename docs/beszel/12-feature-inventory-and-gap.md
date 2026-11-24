# 12 — Beszel feature inventory and Gloski gap

This is a **reference for comparison**, not a to-do list. It inventories what
Beszel actually has at the source snapshot used by this folder
(`6702f013f8ceed0c48d95c1988a58b6d970b5022`, v0.21.2, MIT © 2024 henrygd) and
places it beside Gloski's current state. Every "skip" below is an owner decision
recorded in `README.md`, `10-do-not-borrow.md` or `11-implementation-plan.md`;
none of it is an accusation that Gloski is behind where it intended to be.

Sources: `01`–`11` in this folder. Where a row says "idea only", the sourcebook
classifies it as C (borrow the pattern, not the code).

---

## 1. Product and system design

| Concern | Beszel | Gloski |
|---|---|---|
| Purpose | Monitoring platform: collect metrics, keep history, graph them, alert | Control tower: run jobs, open terminals, manage files, inspect processes, repair agents |
| Centre of gravity | Observability | Remote **action** |
| Hub/master | `beszel hub`: PocketBase app with cron, collections, realtime broker, admin UI, API | `gloski`: `net/http` + WebSocket + gRPC listeners + SQLite + embedded agent builds |
| Agent | `beszel-agent`: stateless collector with cache-time tiers and an SSH fallback server | `gloski-agent`: stateless worker — jobs, PTYs, file transfers, probes, stats |
| Transport | WebSocket + CBOR, one request/response per action, SSH fallback | One bidirectional gRPC `Agent.Connect` stream; unix socket locally, h2c over Tailnet |
| Browser | React SPA → PocketBase REST + SSE + custom `rt_metrics` topic | React SPA → master HTTP/WS only |
| Identity | Token + persisted fingerprint; auto-registration from universal tokens | Name-keyed device with a `machine_id` duplicate guard |
| Lifecycle | Agent self-updates from GitHub releases; systemd/OpenRC/procd/rc.d/cron | Master installs, restarts and upgrades over SSH; one general repair (reinstall) |
| Auth/users | Multi-user, roles, OAuth/OIDC, sharing, collection rules | Single optional API key/JWT; dashboard behind the owner's basic-auth proxy |
| Notifications | Alerts, quiet hours, shoutrrr, email | None, by decision |

**Design implication:** Beszel optimises for breadth of hardware and low
operating cost; Gloski optimises for reliability of action. The architectures
overlap only at "hub and agent". Adopting PocketBase, agent self-update or an SSH
fallback would undo Gloski's core decisions. The sourcebook's rule is selective
borrowing, not convergence.

### Beszel system shape (from `01`)

```
agent/
  agent.go                god-struct; gatherStats with cache intervals
  client.go               WebSocket client, auth challenge, CBOR
  connection_manager.go   WS <-> SSH state machine
  cpu/disk/gpu/smart/zfs/btrfs/sensors/battery/docker/...  collectors
  network_monitor_*.go    active probes, bounded history
  deltatracker/           generic rate calculator
internal/
  hub/                    PocketBase app, API, agent_connect, realtime, systems
  records/                tiered rollups and retention
  alerts/                 evaluation, history, quiet hours, delivery
  ghupdate/               self-update with checksum and rollback
  entities/               compact CBOR/JSON data model
  site/                   React SPA embedded in the binary
```

---

## 2. Metrics and collectors

| Beszel capability | Detail | Gloski today |
|---|---|---|
| CPU | Cross-platform sampling, per-cache-time baselines, counter rollback, per-core % | CPU total + per-core from the agent's stats; rollback guards borrowed |
| LXC cgroup CPU | lxcfs-aware `/proc/stat`, `cpu.stat`, cpuset/quota core count | Not built (parked collector batch) |
| Memory | Byte arrays, ZFS ARC subtraction, cgroup v1 sentinel, saturating math | Total/used/free/available/cached/buffers + % |
| Load average | 1m/5m/15m | `load1` sampled and charted |
| Uptime | `/proc/uptime` (lxcfs-safe), rejects NaN/Inf/negative | Uptime from stats |
| Disks / usage | Root + extra filesystems, immutable-OS `/sysroot`, ZFS dataset mounts | Per-mount usage %, `disk:<mount>` history |
| Disk I/O | read/write B/s, time %, util %, r_await/w_await, weighted %, 32-bit wrap handling | **Not collected** |
| Network | per-NIC deltas, sum-of-deltas, skips lo/docker/veth/bond, Jetson counter handling | Aggregate rx/tx rates sampled and charted |
| GPU | AMD sysfs, NVML, nvtop, Intel `intel_gpu_top`/sysfs, macOS powermetrics (~3.5k LOC) | **Not collected** |
| SMART / disk health | smartctl JSON, nvme/scsi/ata protocol detection, virtual-device rejection, mdraid/eMMC heuristics | **Not collected** |
| ZFS / btrfs pools | pool capacity, health, scrub, IO stats, ioctl fallback | **Not collected** |
| Sensors / fans | temperature, RPM, timeout + inflight guards, duplicate-name disambiguation | **Not collected** |
| Battery | primary selection, charge state, per-battery % | **Not collected** |
| Wi-Fi | association from BSS status, RSSI dBm, OpenWrt `iwinfo` | **Not collected** |
| Docker / Podman | per-container CPU/mem/net, state, bounded logs, registry checks | **Not collected** |
| Package updates | apt/dnf/zypper/pacman/apk/xbps parsers with hourly cache | **Not collected** (shell checks could cover it) |
| Health file | touch every 90 s, newest-mtime read | Live gRPC heartbeat covers this |
| Fingerprint | persisted SHA-256 host ID, known-bad UUID guard, hostname+CPU fallback | `machine_id` stored, but identity is still name-keyed |

Gloski's collector surface is intentionally small; the owner's rule is to add a
native collector only when a real device needs a metric a shell command cannot
answer. That is batch 7, still parked.

---

## 3. Stats data model and retention

| Beszel | Gloski |
|---|---|
| Compact integer CBOR keys + short JSON keys; reserved slots ("do not reuse"); deprecated fields kept one cycle | Protobuf field numbers; no reserved-field or deprecation policy yet |
| `Stats` carries CPU%, mem%, swap, disk%, temperatures, extra filesystems, GPU, bandwidth, load, battery, per-NIC rates, disk I/O, CPU breakdown, per-core, disk-I/O stats, fans, batteries, ZFS, Wi-Fi, mem/swap bytes | `SystemStats`: CPU (total/per-core), memory, swap, disks, network, load1, uptime, processes |
| Hub-only peaks marked `cbor:"-"`; fixed tuples `[2]`, `[3]`, `[6]` with documented order | No peak preservation |
| `Info` (dashboard table) and `Details` (static once per connection) | One `FleetDevice` shape |
| `CombinedData` bundles Stats, Info, Containers, Systemd, Details, Monitors | Separate API payloads |
| Capability flags (`Complete`, `SmartComplete`, `SystemdServicesUpdated`, `SystemdLogs`) distinguish "no data" from "not collected" | No capability flags |
| Tiered rollups: 1m→10m (≥9 samples), 10m→20m (≥2), 20m→120m (≥6), 120m→480m (≥4), +5 min boundary fudge | 5-minute samples; 7 days full, then one/day up to 365 days |
| Retention: 1m 1h, 10m 12h, 20m 24h, 120m 7d, 480m 30d; alert history newest 200; containers 10 min, systemd 20 min | `metric_samples` thinned daily; jobs capped 10 MB and 30 days |
| Averaging preserves CPU/memory/network/bandwidth/disk-I/O peaks and uses per-key counters for intermittently-present sensors | No averaging/peaks; plain samples |

The portable idea here is the **retention algorithm plus peak preservation**, not
the PocketBase collections or the field list.

---

## 4. Protocol and transport

| Beszel | Gloski |
|---|---|
| WS + CBOR; `HubRequest` with integer keys 0/1/2 and generic raw-payload slot 7 | gRPC protobuf oneof, typed messages |
| Request `Id` multiplexes concurrent requests; `Error` string per response | Stream RPCs; no generic raw escape hatch |
| `DataRequestOptions{CacheTimeMs, IncludeDetails}` — 1 s realtime vs 60 s poll on one socket | Fixed heartbeat/stats cadence |
| Signed Ed25519 auth challenge; agent verifies a **key ring**, supporting rotation | Join token + device token |
| Token resolution tree: universal token → fingerprints → auto-create device; reject fingerprint mismatch | Name-keyed registration; `machine_id` only blocks duplicates |
| Deadlines: hub read 100 s, agent 120 s, data fetch 30 s, max silence 3 min, down-state delay 5 s, stagger `interval/N`, jitter 51–95% | gRPC keepalive + reconnect backoff; no "slow collector ≠ dead agent" state yet |
| Slow fetch pings to extend deadline without requiring a reply; `DeadlineExceeded` never closes the socket | Not implemented |
| WS→SSH fallback with a three-state connection manager | **Deliberately absent** (owner: no SSH fallback for actions) |
| Realtime fan-out: lazy worker, generation-based stop, claim/finish in-flight snapshots so slow agents are skipped | `/api/fleet/ws` and per-device stats WS; skip-slow/generation ideas not ported |
| Named minimum versions (`MinVersionZfsData`, `MinVersionNetworkMonitors`, …) | One `ProtocolVersion` (currently 4) |
| Secret-safe errors; proxy dialer bound to agent lifetime; `EXIT_ON_DNS_ERROR` | Not applicable to the same degree |

---

## 5. Checks, monitors and alerts

| Beszel capability | Detail | Gloski today |
|---|---|---|
| Probe types | ICMP (raw→datagram→exec ping fallback), TCP (DNS outside timing, shared deadline), HTTP (accepted status ranges, fail-closed on bad config), DNS (custom server) | TCP, HTTP, DNS, TLS leaf expiry via `GLOSKI_PROBES`; **no ICMP** |
| Scheduling | random 50–100% stagger, serial checks, hub-idle gate after 3 min, suspend/clock-jump generation guard | Stagger + serial + 3-minute idle gate; suspend guard not ported |
| Bounded history | 61 raw samples + 61 wall-clock minute buckets, ~1 hour constant memory | 61-sample / 61-minute aggregation into `check_results` |
| Result shape | avg/min/max µs, 1h windows, loss %, sample count, cert info | Status, summary, details, ran-at, stale |
| Stable IDs | `FNV32a(systemID,target,protocol,…)` formatted `%06s` | FNV stable IDs, same intent |
| Sync protocol | full replace on connect + incremental upsert/delete; version gate | One new proto message pair on the existing stream |
| TLS cert check | hourly dial, `InsecureSkipVerify`, report leaf expiry, one-shot delivery | TLS probe implemented |
| Network-monitor UI | dedicated `/monitors` page, table + monitor dialog + compare sheet, bulk CSV add | **No check-configuration UI**; matrix is filled by built-in signals + opt-in probes |
| Alerts | 17 alert types, hysteresis, averaging windows, confirm-down timers, incident history, quiet hours, shoutrrr/email | **None, by decision** |
| Outbound heartbeat | POST/GET/HEAD to external uptime service with fleet status JSON | **None, by decision** |

Gloski's matrix now has built-in Connectivity/Health/Agent/Jobs blocks derived
from dashboard facts, plus stored results. Saved checks cannot yet be created
from the UI — that is the biggest functional gap in this area.

---

## 6. Hub, storage and operations

| Beszel | Gloski |
|---|---|
| PocketBase collections: systems, users, fingerprints, stats tiers, monitors, alerts, quiet hours, user settings, containers, systemd, SMART, ZFS, details, universal tokens, API keys | SQLite tables: devices, deployments, check_results, device_actions, metric_samples, agent_jobs/logs, legacy tables |
| PocketBase cron, rules, admin UI, realtime, backups (PRO) | Own scheduler/maintenance pass, own handlers, own diagnose chain; no admin UI |
| Self-update from GitHub releases with streaming SHA-256, path containment, `.old` rename, EXDEV fallback | Master verifies embedded SHA-256 and stages replacement; no GitHub fetch |
| Installers: systemd/OpenRC/procd/rc.d/cron, `/opt`/`/usr/local/sbin`, Homebrew/Scoop/WinGet/Debian, six container images, Helm | One `/tmp` install script, three embedded architectures, master-owned lifecycle |
| SPA serving: embedded FS, runtime info injection, base-path rewrite, assets cached 30 days, 404 status with SPA body | Embedded/served `web/dist`, 404-with-body, `index.html` no-cache, hashed assets immutable |
| Proxy trust CIDRs, trusted auth header, opaque 404 | `GLOSKI_ALLOWED_ORIGINS=*` escape hatch; security hardening explicitly not a priority |
| ~25 API routes under `/api/beszel`, static API keys, stateless MCP façade with 10 read-only tools | Focused API; no MCP (parked — diagnose chain already returns structured state) |
| "Automatic backups" are a PocketBase admin feature, not Beszel code | Nightly backup still a roadmap item; self-test reports it as not configured |

---

## 7. Frontend

| Beszel surface | Gloski today |
|---|---|
| React 19.1 + Vite 7 + Tailwind 4 + shadcn/Radix | Same stack |
| Recharts + d3-time; Chart.js render layer would be swapped | Chart.js + gap-aware data/axis math |
| TanStack Table with **grid and list** views; one View dropdown with Layout/Status/Sort/Fields panes; hover-preloads | Own table with one Sort/Fields/page-size dropdown; sticky header, pagination; **grid Layout view skipped** |
| Chart catalogue: CPU, memory, network, disk, GPU, sensors, storage pools, monitors, load average, Wi-Fi; fullscreen drill-down sheets; `syncId` cursors | CPU, memory, network, disk-usage charts on the device page; no fullscreen sheets or range selector |
| 1 s realtime range with request-ID guard, 60/100-point caps, stale-cache check | Fixed range history; no 1 s realtime tier |
| `chart-time-select` hides ranges above the agent's minimum version | No range selector |
| Command palette (Cmd/Ctrl+K), three-state theme toggle, masked token with reveal/copy, favicon active badge, toast viewport | Pre-paint theme script and simple toggle; **command palette, masked token and favicon badge skipped** |
| Systems table with meters, status counts, filters, mobile defaults to grid | Fleet health screen + devices table; no grid view or filter sidebar in use |
| Containers, systemd, monitors, SMART, ZFS, package-update, alert-history tables | **None of these pages exist** |
| PWA manifest, country-flag font, Lingui i18n with 34 catalogues | No PWA/i18n (out of scope) |
| Generated HSL series colours and Total tooltip | **Skipped** — no many-series chart today |
| Utility library: formatting, debounce, semver, filter groups, uptime, package-update parsing, monitor compare, Wi-Fi helpers | Subset ported; semver/filter-group/package/Wi-Fi helpers deliberately skipped as low-value |
| tests: 8 files, ~958 LOC | 150 tests across 11 files (current) |

---

## 8. Packaging, release and testing

| Beszel | Gloski |
|---|---|
| 1,473-LOC `install-agent.sh` across Linux/systemd/OpenRC/procd, FreeBSD, OPNsense, pfSense, macOS, Windows | One config-preserving install script with named phases, staged replace, checksum, rollback |
| Agent self-update via cron/timers; version resolution from GitHub; checksums manifest | Master is the only source of binaries; agents never self-update |
| Six Dockerfiles, Helm charts, Debian packages, Homebrew/Scoop/WinGet, 30+ GoReleaser targets | Three embedded architectures, no containers/Helm/packages |
| 6-image Docker CI, Helm CI, vulncheck, dependabot; **no general Go test CI** | `go test ./...`, race tests where cgo exists, `bun test`, type-check, build, reviewer gate |
| Test patterns: build-tag helpers, variable-as-seam, self-reexec fake commands, `testing/synctest`, 33 real fixtures, table-driven tests | Some patterns adopted; fixture corpus and self-reexec fakes still limited |
| `agent.go` is acknowledged as a god-struct | `Tower` and `FilesContext` are the known god objects (`docs/code-quality-plan.md`) |

---

## 9. Condensed gap table

Legend: **done** · **partial** · **not built** · **skip** (owner decision).

| Feature | Status in Gloski |
|---|---|
| Hub-and-agent fleet control | done |
| Master-owned agent lifecycle, SSH deploy/reinstall/rollback | done |
| One gRPC stream with protocol gate | done |
| Jobs, terminal, files, processes, run-on-many | done (Gloski's differentiator; Beszel has none) |
| Diagnose chain + dashboard issues with advice | done |
| Metric history + device charts | done (simpler retention, no peaks) |
| Fleet check matrix + native TCP/HTTP/DNS/TLS probes | partial (no saved-check UI, no ICMP) |
| Fleet health verdict / attention screen | done (new) |
| Table controls, sticky headers, pagination, theme | partial (no grid layout/command palette) |
| Disk I/O, sensors, GPU, SMART, ZFS, containers, Wi-Fi, battery, package updates | not built (batch 7 parked) |
| Alerts, quiet hours, notifications, multi-user/OAuth | skip |
| PocketBase, WebSocket/CBOR, SSH fallback, agent self-update | skip |
| Systemd/containers/Helm/package-manager install matrix | skip |
| Chart range selector, 1 s realtime, fullscreen drill-down | not built |
| MCP façade, outbound heartbeat, i18n, PWA | skip/parked |
| Nightly backups | not built |

---

## 10. How to read this when deciding what is next

1. **Visible monitoring surface** is the area where Gloski is furthest from
   Beszel: collectors and rich charts. The owner's rule keeps it parked until a
   device actually needs a metric.
2. **Checks** are structurally present but user-configuration is missing. A
   saved-check UI would make the matrix and probes actionable without adding a
   collector.
3. **Frontend polish** is deliberately selective: no command palette, grid
   layout, generated colours or i18n. These were judged low-value for this
   fleet's size.
4. **Never copy**: alerts, PocketBase, WS/CBOR transport, SSH fallback, agent
   self-update, service-manager installers, a combined health score, or the
   security guards that would block legitimate internal destinations.

Attribution: Beszel is MIT licensed; the project is credited in the repository
README. No separate notice file or per-file headers, per owner decision.
