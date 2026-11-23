# Beszel sourcebook

This folder is a working reference for borrowing from
[beszel](https://github.com/henrygd/beszel). It replaces the single-file
`docs/ideas-from-beszel.md`, which was too small for the amount of material in
that codebase.

Source of truth: shallow clone of `main` at
`6702f013f8ceed0c48d95c1988a58b6d970b5022` (v0.21.2), 640 files, 333 Go files,
164 frontend files, 66 agent test files. All paths in these documents are
relative to that clone, **not** to this repository. Re-check paths if you
refresh the clone.

Beszel is MIT licensed (Copyright © 2024 henrygd). This is an internal project
and the source is credited in the project README; there is no separate notice
file or per-file header (owner decision, 2026-11-23). Revisit if the project is
ever distributed publicly. Two vendored items are not MIT:
`supplemental/licenses/smartmontools/LICENSE` (GPL-2.0) and
`supplemental/licenses/LibreHardwareMonitor/LICENSE` (MPL-2.0) — do not copy
their code.

## How to use these documents

Every row carries a reuse class:

| Class | Meaning |
|---|---|
| **A — copy** | Self-contained; lift the file, swap imports/types, keep the MIT notice. |
| **B — adapt** | Sound algorithm/component, but tied to Beszel types, CBOR, Recharts, nanostores or PocketBase. |
| **C — idea** | Borrow the pattern, constant, edge case or data shape; do not copy code. |
| **D — skip** | Irrelevant, already in Gloski, or conflicts with an owner decision. |

Non-negotiables that override every table (from `AGENTS.md` and `docs/roadmap.md`):

- No alerts or notifications; no multi-user, OAuth, roles or sharing; security
  is not a priority for now.
- The master owns the agent lifecycle. No agent self-update, no systemd push,
  no manual install path, no SSH fallback for actions.
- `/tmp/gloski` installs are deliberate. Keep it simple; one general repair.
- Gloski's differentiator is remote *action* (jobs, terminal, files, processes,
  run-on-many). Beszel does not do any of that. Protect it.

## Index

| Doc | Covers |
|---|---|
| [01 — Design and architecture](01-design-and-architecture.md) | Product comparison, architectural decisions, patterns worth adopting, systemic things to avoid. |
| [02 — Agent and metrics](02-agent-and-metrics.md) | Every collector: CPU, memory, disk I/O, GPU, SMART, ZFS/btrfs, sensors, batteries, Wi-Fi, Docker, package updates, health file. |
| [03 — Stats and data model](03-stats-data-model.md) | Full `Stats`/`Info`/`Details` field tables, entity types, encoding and versioning rules. |
| [04 — Protocol and transport](04-protocol-and-transport.md) | CBOR envelope, actions, auth/fingerprint, deadlines, reconnect, realtime fan-out, version gates. |
| [05 — Hub and storage](05-hub-and-storage.md) | Startup/CLI, PocketBase collections, migrations, rollups, retention, self-update, SPA serving, proxy handling, MCP. |
| [06 — Checks, alerts and monitors](06-checks-alerts-and-monitors.md) | Network probes and bounded history, alert evaluation ideas, mapping to Gloski's checks roadmap. |
| [07 — Frontend](07-frontend.md) | Stack comparison, portable TypeScript, charts, tables, UX details, component catalogue, tests. |
| [08 — Packaging and release](08-packaging-and-release.md) | Install-script safety tricks, binary replacement, build/release matrix, what to skip and why. |
| [09 — Testing and quality](09-testing-and-quality.md) | Fixture seams, self-reexec command fakes, `testing/synctest`, frontend tests, CI gap. |
| [10 — Do not borrow](10-do-not-borrow.md) | Consolidated exclusion list with reasons and conflicts. |
| [11 — Implementation plan](11-implementation-plan.md) | Batch-by-batch plan with concrete tasks, files, verification gates and settled decisions. |

## Highest-value items, ranked

| # | Item | Source | Class | Where it lands in Gloski |
|---|---|---|---|---|
| 1 | Formatting/debounce/semver/filter helpers + tests | `internal/site/src/lib/utils.ts` | A | `web/src/shared/` |
| 2 | Gap-aware chart data and axis math | `.../routes/system/chart-data.ts`, `.../ui/chart.tsx` | A/B | device charts |
| 3 | Network probes + 2-tier history + stable IDs | `agent/network_monitor_*.go` | A | new `internal/checks` |
| 4 | Verify → stage → rename → rollback binary replacement | `internal/ghupdate/*.go`, `install-agent.sh` | A/B | `internal/provisioning` |
| 5 | Reinstall preserves config not explicitly supplied | `install-agent.sh` | C | `agentdist/install-agent.sh` |
| 6 | Generic delta tracker | `agent/deltatracker/` | A | `server/internal/system/` |
| 7 | LXC cgroup CPU/memory and `/proc/uptime` helpers | `agent/{cpu_linux,memory_linux,uptime_linux}.go` | A | `server/internal/system/` |
| 8 | Tiered rollups and retention with peak preservation | `internal/records/*.go` | B | `controltower/maintenance.go` |
| 9 | Deadline/silence rules for slow collectors | `internal/hub/ws/ws.go`, `internal/hub/systems/system.go` | C | heartbeat work |
| 10 | Table View/Layout/Sort/Fields cluster and pagination | `.../systems-table/*` | B | devices/jobs/files tables |
| 11 | Command palette, pre-paint theme, runtime config injection | `.../command-palette.tsx`, `index.html`, `server.go` | B | web app shell |
| 12 | Sensor timeout + inflight guard, defensive SMART parsers | `agent/sensors.go`, `internal/entities/smart/smart.go` | A | agent collector/checks |
| 13 | Realtime backpressure and generation-stable workers | `internal/hub/systems/system_realtime.go` | B | fleet WS |
| 14 | Fingerprint with known-bad-UUID guard | `agent/fingerprint.go` | C | closes the `machine_id` gap |
| 15 | Test seams, fixture self-reexec, `testing/synctest` | 66 `agent/*_test.go` | A | server tests |

## Suggested borrow order

1. **Utilities and test patterns** — no product surface, no new dependencies.
2. **Packaging and reliability discipline** — named install phases, config-preserving
   reinstall, staged rollback, proxy CIDR parsing, SPA 404/base-path handling.
3. **Check result model, then probes** — roadmap step 3; results travel with the
   existing heartbeat.
4. **Charts and tables** — gap-aware data, generated series colours, View dropdown.
5. **History and retention** — roadmap step 4; extend the existing daily pass.
6. **Collector expansion** — only if a real device needs a metric a shell cannot
   answer. Requires `gopsutil/v4` and a deliberate decision.

Parked: MCP façade, outbound heartbeat/pinger, package-update parsers, i18n,
fingerprint-based identity.

## Open decisions for the owner

1. Native collectors or shell-based checks only? Recommended: probes (batch 3)
   yes, full collector expansion (batch 6) only on demonstrated need.
2. Adopt `github.com/shirou/gopsutil/v4`? It unlocks most collector code.
3. Want an outbound dead-man heartbeat to an external uptime service? Different
   from the alerts you declined, but still a new external dependency.
4. ~~Attribution file.~~ Decided: credit Beszel in the project README; no
   separate notices file.
