# 11 — Implementation plan

This is the bridge between the reference documents in this folder and actual
code. It is ordered by project need, not by how interesting the source code is.

## Status

- **Batch 1 — done** (commit `72720cc`): counter-rollback guards in the stats
  collector, `collector_test.go`, the sourcebook, README credit.
- **Batch 2 — storage, sampling, endpoint and charts done**; trend flags (step
  5) deferred until a real check needs them. `metric_samples` is migration 11,
  `internal/metrics` owns the store, heartbeats are sampled on the master, the
  daily maintenance pass thins history, and `GET /api/devices/{id}/history`
  feeds charts on the device page.
- **Current project state:** check result model, read-only master self-test and
  device action log are in. `docs/ideas-from-smdwin.md` says the next ideas are
  **history and the fleet matrix**, so those come first here too.
- **Decisions made (2026-11-23):** 5-minute samples with 7-day full / 365-day
  daily retention; native probes will cover TCP/HTTP/DNS/TLS first, ICMP later;
  chart colour/palette polish waits until history charts exist (now done);
  `gopsutil/v4` stays parked with batch 7.
- **Not done:** batch 2 step 5 and batches 3–7 below.

## Ground rules for every batch

1. One batch = one commit on `main`, small enough to revert. Never push unless
   asked. No branches.
2. **Idea first, code second.** Copy only self-contained code with tests; most
   items in this folder are idea-only. No new dependency until a batch needs it —
   `gopsutil/v4` stays parked with batch 7.
3. **Hook into existing machinery.** Pruning goes in `maintenance.go`; fan-out in
   the fleet WS; stats sampling in the existing heartbeat path; check results in
   `internal/checks`. No second scheduler, database, cache or transport.
4. **Verify every batch:** `scripts/review-checks.sh`, `go test ./...`,
   `go vet`, frontend type-check/tests/build, a live master + local agent on
   scratch ports, and (for installer work) the test SSH server. Then the reviewer
   agent must return `VERDICT: PASS`. Note: race tests need cgo/gcc, which this
   container lacks; record that rather than claiming they ran.
5. **Attribution:** a credit line in the project README, per owner decision. No
   notice file, no per-file headers.
6. **Per-file loop when code is copied:** copy source + test → remove CBOR tags
   and Beszel entity imports → substitute Gloski types → prune to the function
   being borrowed → run the borrowed test → wire one call site → verify.

## Batch 2 — History and retention (project's next item)

**Goal:** keep metrics for days, bounded in size, with trend detection that does
not cry wolf. Sources: `docs/ideas-from-smdwin.md` §5 and
[05 — Hub and storage](05-hub-and-storage.md).

| Step | Concrete work |
|---|---|
| 1 | Migration 11: `metric_samples(device_id, metric, ts, value)` with an index on `(device_id, metric, ts DESC)`. |
| 2 | Sample on the master when a heartbeat arrives, at most once per metric per 5 minutes. Do not write on every 2-second stats push. Keep the write in one small function with a unit test. |
| 3 | Extend `maintenance.pruneJobs()` (rename to `prune` if clearer): full resolution for 7 days, then one sample per day up to 365, thinning in the existing daily pass. Keep the existing 30-day job retention and check-history pruning untouched. |
| 4 | Add `GET /api/devices/{id}/api/system/history?metric=&range=` and feed the device page charts. Reuse the Chart.js components already present; the gap is storage, not charting. |
| 5 | **Deferred.** Trend flags: compare latest value to a 30-day reference, not the previous sample; call it degradation only on monotonic growth above an absolute threshold (disk free, swap, temperature). No alerts — the chart/check turns red. |

Likely files: `server/internal/database/migrations.go`,
`server/internal/controltower/maintenance.go`, a new history store under
`server/internal/system/`, `routes.go` / device API, web device charts.

**Verify with:** a live master running long enough to produce samples (shorten
the sample interval via config for the test), then check row counts and chart
output; confirm the DB does not grow unbounded; self-test still reports the last
maintenance run.

## Batch 3 — Fleet check matrix

**Goal:** "every device, every check, red first" at a glance. Sources:
`docs/ideas-from-smdwin.md` §6 and [07 — Frontend](07-frontend.md) table patterns.

| Step | Concrete work |
|---|---|
| 1 | Query the latest check result per `(device, check)` from `check_results`. |
| 2 | Dashboard matrix: devices as rows, check groups (Connectivity, Storage, Services, Updates, Backups) as columns, one coloured block per check. Grey for skipped/not run, not red. |
| 3 | Keep the existing "needs attention" list above the matrix; sort rows with critical cells first; click a cell to expand details. |
| 4 | Reuse the View/Layout/Sort/Fields pattern from Beszel's systems table rather than inventing a new control. |

Likely files: dashboard components, a new matrix component, check store query.
**Verify with:** seeded check results covering ok/warning/critical/skipped/
stale; visual check that unknown never looks like failure.

## Batch 4 — Packaging and reliability discipline

**Goal:** make reinstalls safe and failures explicit. Sources:
[08 — Packaging and release](08-packaging-and-release.md) and
[05 — Hub and storage](05-hub-and-storage.md).

| Step | Concrete work |
|---|---|
| 1 | Named install phases in every error, feeding the existing diagnose output. |
| 2 | Config-preserving reinstall: only overwrite fields the master explicitly supplies (the `*_PROVIDED` idea). A reinstall must never clobber the token or port. |
| 3 | Staged replacement: `mktemp` on the destination filesystem, keep `.bak`, atomic `mv`, trap-driven rollback. Simplify Beszel's version — no service-manager branch, since our agents are plain processes in `/tmp`. |
| 4 | Checksum manifest lookup by filename, `tar -tzf` pre-flight, non-empty binary check (adapt to our embedded builds). |
| 5 | SPA serving: 404 status for unknown routes while still rendering the SPA page, base-path rewriting and runtime config injection. |
| 6 | Trusted-proxy IP/CIDR parsing with `net/netip`; invalid entries narrow the allowlist; warn when unrestricted. |

Likely files: `server/internal/agentdist/install-agent.sh`,
`server/internal/provisioning/{scripts,installer}.go`,
`server/internal/controltower/spa.go`, `server/internal/httpapi/`.

**Verify with:** deploy and reinstall on the test SSH server; confirm token/port
survive; kill mid-install and confirm rollback; check SPA 404 status and proxy
header handling. This batch must not change the no-SSH-fallback rule.

## Batch 5 — Native check probes (opt-in)

**Goal:** let checks verify a URL, port or certificate from the device without a
fragile shell one-liner. Source: [06 — Checks, alerts and monitors](06-checks-alerts-and-monitors.md).

| Step | Concrete work |
|---|---|
| 1 | Port probe functions only: TCP connect, HTTP with accepted status ranges, DNS, TLS leaf expiry. ICMP last, if needed. |
| 2 | Bounded result aggregation (61 raw seconds + 61 minute buckets) into the existing `check_results` history, not a new store. |
| 3 | Idle suppression tied to the heartbeat: stop scheduled probes when nobody has asked for stats for ~3 minutes. |
| 4 | Stable FNV IDs for idempotent saved checks. |
| 5 | Results travel on the existing gRPC stream / heartbeat. No second channel, no notifications. |

Likely files: new `server/internal/checks/probes*.go`, `proto/agentv1/agent.proto`
(add one check-execute message pair), `server/internal/worker`, check UI.

**Verify with:** a local target that flips up and down; confirm ok/warning/
critical/skipped states and that no result appears while idle.
**Decision required:** shell checks first, or native probes now? Recommended:
probes only for things a shell command cannot answer cleanly.

## Batch 6 — Frontend utilities and polish

**Goal:** reuse the same-stack UI work with low risk. Source:
[07 — Frontend](07-frontend.md).

| Step | Concrete work |
|---|---|
| 1 | Port pure TypeScript helpers + their tests into `web/src/shared/lib/`: formatting, debounce, semver, filter groups, uptime. Extend the existing `utils.ts`; do not duplicate `formatBytes`. |
| 2 | Gap-aware chart data (`appendData`) and `fixedDomainTicks`/axis math; adapt to Chart.js. |
| 3 | Generated HSL series colours and Total tooltip row for many-series charts. |
| 4 | Table View/Layout/Sort/Fields dropdown, per-device page-size pagination, sticky headers. |
| 5 | Command palette (Cmd/Ctrl+K), pre-paint theme script, masked join token. |

Likely files: `web/src/shared/lib/`, device charts and tables, app shell,
`index.html`.
**Verify with:** `bun run type-check && bun test && bun run build`, plus visual
comparison against the live master.

## Batch 7 — Collector expansion (parked)

Only start when a real device needs a metric a shell command cannot answer.
Take one collector at a time — LXC cgroup helpers first, then diskstats wrap
handling, sensor timeout guards, SMART/ZFS parsers. Requires the `gopsutil/v4`
decision and the fixture/self-reexec test pattern from
[09 — Testing and quality](09-testing-and-quality.md).

## Decisions still open

1. **History resolution:** 5-minute samples with 7-day full / 365-day daily
   retention (smdwin rule) — confirm before migration 11.
2. **Checks:** native probes for which protocols, and is ICMP worth the raw
   socket complexity? Recommend TCP/HTTP/DNS/TLS first, ICMP later.
3. **Frontend:** adopt the generated HSL series colours and command palette now,
   or wait until history charts exist to judge them?
4. **gopsutil/v4:** stays parked unless batch 7 is approved.
