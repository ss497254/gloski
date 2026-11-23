# Ideas worth borrowing from SMD Win

[SMD Win](https://github.com/paulchirila2026/SMDWin) is a portable Windows
diagnostics tool (C#/WPF, GPL-3.0). It solves a different problem, but several
of its patterns fit Gloski's reliability-first roadmap. This note records
patterns, not code: SMDWin is GPL-3.0 and none of its source is copied.

The three things SMDWin gets right, and Gloski should copy:

1. **Facts are separate from verdicts.** A check returns a summary, details and
   a severity; the UI decides how to present it.
2. **Unknown is a first-class state.** "Not applicable", "never ran" and "old
   result" are not failures, and must not look like failures.
3. **Every change is recorded and, where possible, reversible.**

## 1. A check result needs more than pass/fail

**In SMDWin:** `Services/DiagnosticOrchestrator.cs` defines
`DiagnosticPhaseResult`: `Severity` (`Ok`, `Warning`, `Critical`), `Summary`
(one factual line), `Details` (ordered key/value pairs), `Elapsed`, `Skip`,
plus flags carried in `Details`: `NotTested`, `Stale`, `Info`.
`views/HealthCheckWindow.cs` renders each state differently and only counts
judged checks in the verdict.

**Why Gloski needs it:** the roadmap says "a check is a saved command whose
exit code means healthy or not". An exit code alone cannot tell a failed check
from one that does not apply to this device (no Docker, no ZFS, no battery), or
one that has never run. False red lights are worse than no checks.

**Concrete design:**

- New `internal/checks` package with a `Result`:
  - `Status`: `ok` | `warning` | `critical` | `skipped` | `unknown`
  - `Summary` (one line, factual: "System drive 94% used — 12 GB free")
  - `Details` (key/value, shown expanded)
  - `RanAt`, `Duration`
  - `Stale` (computed: older than 2× the schedule interval)
- Shell convention, kept simple:
  - exit `0` = ok, `1` = warning, `2` = critical, anything else = unknown
  - first line `GLOSKI-SKIP: <reason>` = skipped
  - first line of output = summary, rest = details
- Store in a `check_results` table keyed by `(check_id, device_id)`; keep the
  last result and a short history.
- UI: coloured dot + summary; grey/hidden for skipped; "not run" in grey, not
  red; "old result" pill when stale; details on expand. Read-only by default.

**Where it fits:** step 3 (checks); also supplies the "reason" field for step
1.4. **Size:** small-medium.

## 2. "Needs attention" that says what to do

**In SMDWin:** `HealthCheckWindow.cs` has an `Advice` dictionary per check
("A drive reports problems. Back up important files soon."), sorts critical
before warning, and its verdict says "Start with the first item on the right —
it is the most important." A separate line reports tests that have not run, so
"unknown" never masquerades as "broken". It also has "Copy report".

**Why Gloski needs it:** the dashboard already lists issues, but a raw detail
("deployment failed: ...") is not a reason a person can act on. The owner's
stated pain is "some random issue blocks me".

**Concrete design:**

- Extend `dashboardIssue` with `advice` (one sentence: what it means / what to
  do), `action` (route to the device, check or deployment) and an explicit
  order.
- Built-in issues get a static advice string; a saved check can carry an
  optional `advice` field, otherwise show the last output line.
- Sort critical first, then warning, then age; show the top 8 with "+N more".
- Keep a separate "N checks not run" line.
- Add "Copy report" (plain text) on the dashboard and device page.

**Where it fits:** step 1.4 (clear reasons) and step 3. **Size:** small.

## 3. Diagnose the connection chain, not just "offline"

**In SMDWin:** `Services/NetworkDiagnosisService.cs` walks adapter → gateway →
internet → DNS → HTTP, stops at the first broken link, and returns a full
summary plus a short one ("Router responds, but nothing beyond it — likely an
ISP/upstream problem").

**Why Gloski needs it:** "agent is dead" is the exact pain the project exists
to remove. Today the master knows several pieces (tailnet, SSH, agent stream,
build hash, rejection reason) but the person sees only "offline".

**Concrete design:** a `POST /api/devices/:id/diagnose` endpoint that walks a
fixed chain and stops at the first failure:

1. Tailscale peer visible (existing `internal/tailnet`).
2. SSH TCP port reachable.
3. SSH authentication and host key accepted.
4. Install directory writable/executable (optional, over SSH).
5. Agent process running (deployment PID / install script).
6. Agent port reachable from the master.
7. gRPC handshake and protocol version accepted.
8. Device registered, connected, and running the master's build.

Each step returns ok/fail + detail; the first failure is the headline, the raw
error stays underneath. Reuse existing provisioning/agent code; add no new
transport. Add a "Diagnose" button next to "Start" / "Trust new key" on
offline devices.

**Where it fits:** step 1.4. **Size:** medium.

## 4. A self-test for the master

**In SMDWin:** `Services/SelfTestService.cs` is a read-only checklist with
pass/warn/fail lines and a copyable report. It checks the app's own health, not
just the machine: settings JSON valid, log folder actually writable (probe
file), log rotation not leaving old files, memory/handle trend across runs
(three points, absolute threshold), plus calculation self-checks.

**Why Gloski needs it:** the master is the single point of failure. Step 1.6
asks for systemd and backups; a runtime self-test is the counterpart that
catches a full disk, a corrupt database, a missing join token or a local agent
that is not the master's build before they bite.

**Concrete design:** a master-side `GET /api/system/self-test` (read-only) and
a page under Settings:

- `PRAGMA quick_check` on `master.db`, migration version current.
- Data directory writable (write and delete a probe file).
- Join token exists and mode `0600`.
- Backups present and fresh (once the nightly backup exists).
- Free space on the data directory.
- Local agent connected and build hash equal to the master's.
- Embedded agent builds for amd64, arm64 and 32-bit ARM present.
- Log writable; last maintenance run succeeded.
- Goroutine count and memory trend across self-test runs.
- Database and WAL size.

Show a "Master health" card on the dashboard when anything is warn/fail; full
page for details, re-run and copy report.

**Where it fits:** step 1.6 and 1.7. **Size:** small-medium.

## 5. History that stays small and only flags real trends

**In SMDWin:** `Services/SmartHistoryService.cs` writes at most one snapshot
per disk every 5 minutes, keeps full resolution for 7 days then one per day,
caps at 365 snapshots, writes atomically (temp file + replace), and sets a
corrupt file aside instead of failing. `CheckDegradation` compares the latest
value to a 30-day reference, not to the previous sample.
`Services/ConsistentGrowthDetector.cs` calls something a trend only when three
points are monotonic **and** the total growth exceeds an absolute threshold.

**Why Gloski needs it:** step 4 adds history. Without a retention rule the
database grows forever; without a reference window every normal fluctuation
looks like a problem.

**Concrete design:**

- `metric_samples(device_id, metric, ts, value)`; write at most one sample per
  metric every 5 minutes.
- Keep full resolution for 7 days, then one sample per day, cap at 365 days.
- For checks, compare the latest value to a 30-day reference, not the previous
  sample; call it degradation only on monotonic growth above an absolute
  threshold (disk free, SMART reallocated/pending sectors, swap, CPU temp).
- Run thinning in the existing daily maintenance pass.
- No alerts; the check turns red and the trend is visible on the device page.

**Where it fits:** step 4, feeding step 3 checks. **Size:** medium.

## 6. A fleet check matrix with group strips

**In SMDWin:** `HealthCheckWindow.cs` groups checks into Security, Storage,
Hardware and System. Each group has a count, a status dot and a strip of one
coloured block per check. Green only when every check has run and passed;
grey when some have not run; red/yellow for critical/warning.

**Why Gloski needs it:** once each device has ten checks, a flat issue list no
longer shows "every device, every check". The roadmap's home page should be
red first, at a glance.

**Concrete design:**

- Dashboard: devices as rows, check groups (Connectivity, Storage, Services,
  Updates, Backups) as columns; each cell is a strip of blocks, one per check.
  Sort rows with red cells first; click a cell to expand that device's checks.
- Device page: the same groups as cards, with details.
- Keep the flat "Needs attention" list above the matrix for the top problems.

**Where it fits:** step 3 UI. **Size:** medium.

## 7. Action log with undo where possible

**In SMDWin:** `Services/ServiceChangeLog.cs` records each service start-type
change with `From`, `To`, `When`, keeps the first "from" across repeated
changes, caps the list, and offers an Undo per entry.
`Services/OptimizeBackup.cs` saves the original value only once and restores it
exactly on revert. `Services/ToolActivityLog.cs` keeps the last 30 actions with
a short result note.

**Why Gloski needs it:** "what changed on this box?" is a common reliability
question. Step 4 adds service and power actions; those are exactly the changes
that need a way back.

**Concrete design:**

- `device_actions` table: `device_id`, `kind`, `target`, `from`, `to`, `when`,
  `result`, `undoable`.
- Log agent start/stop/restart now (the master already knows the wanted state),
  service start/stop in step 4, power actions, and file deletes/renames.
- Undo only where the previous state is known (service state, agent wanted
  state). File deletes are logged, not undoable.
- Show "Recent changes" on the device page with an Undo button per reversible
  entry.

**Where it fits:** step 4; agent actions can start now. **Size:** small-medium.

## Parked for later

- **Diagnostic bundle** (`DiagPackageWindow`, `ReportService`): one zip or
  standalone HTML with master info, devices, checks, recent jobs and logs. Nice
  when reporting a weird issue; not in the roadmap.
- **Plain-language error table** (`EventExplainer`): map common SSH/systemd/
  agent errors to one sentence. Overlaps ideas 2 and 3; add once the real
  errors are known.
- **Short in-memory sparklines** (`CircularBuffer`, `MiniSparkline`): last ~60
  heartbeats per device, no database. Cheap; could precede full history.
- **Service problem grouping** (`ServiceProblemsService`): group repeated
  failures by unit. Useful when step 4 services arrive.
- **Performance regression tracking** (`PageLoadRegressionChecker`): median
  baseline, 1.6× and 800 ms thresholds. Not needed for the current web UI.

## Do not copy

- **A single health score.** SMDWin's own comment says hardware-database
  coverage pulled a clean run down to 43/100, so the score is stability only.
  Gloski's roadmap already says never a combined pass/fail; show counts and
  severities instead.
- **Notification centre / alerts.** The owner does not want alerts; the
  dashboard is where you look.
- **Windows-specific features.** WMI, SMART attributes, BSOD, drivers, battery,
  Wi-Fi, localization, hardware drawings.
- **A repair mechanism per failure case.** SMDWin needs many special cases
  because Windows is heterogeneous. Gloski's one repair is reinstall; keep it.

## Suggested order

1. **Check result model** (1) — foundation for step 3.
2. **Attention list with advice** (2) — small, immediate improvement.
3. **Diagnose chain** (3) — biggest reliability win for the owner's pain.
4. **Master self-test** (4) — protects the single point of failure.
5. **Action log** (7) — small, useful for step 4.
6. **History** (5) and **matrix** (6) — with steps 3 and 4.

## Build order and status

The owner asked for the work in small tasks, done one by one. The connection
chain was chosen first because it needs no checks system and answers the
"agent is dead" pain directly:

1. **Diagnose chain** — done (2026-11-23).
2. **Attention list with advice** — done (2026-11-23).
3. **Check result model** — next; foundation for checks and stale/unknown states.
4. **Master self-test**.
5. **Action log with undo**.
6. **History and fleet matrix** — with roadmap steps 3 and 4.

### What task 1 built

- `POST /api/devices/{id}/diagnose` returns an overall status, a headline, one
  sentence of advice and an ordered list of steps. Step states are `ok`,
  `warning`, `critical` and `skipped`; skipped is never a failure. The first
  critical step is the headline, so the deepest *cause* is shown, not the
  symptom "agent offline".
- The chain is deployment record → Tailscale peer → SSH port → SSH login →
  install directory → agent process → gRPC session → build. It stops at the
  first failure; an agent that is already connected skips the SSH links, and a
  deliberately stopped agent stops at a warning.
- `provisioning.Installer.Probe` reads the install directory over SSH without
  changing it, including a `-v` run to catch a `noexec` mount.
- The dashboard and device page have a Diagnose button that opens a dialog with
  the headline, advice and raw detail for each step.

Two differences from the original design:

- There is no "agent port reachable from the master" step. Gloski agents dial
  out; they do not listen. The equivalent question is whether the agent stream
  is live, and the step reports the address the master listens on.
- An empty ssh-agent used to make SSH login fail outright (the client gave up
  before trying keys on disk). This is fixed in `provisioning.authMethods`; the
  diagnosis would otherwise report a false "SSH key refused".

### What task 2 built

- Every dashboard issue now carries one sentence of advice and a route to the
  device, its task log, or the fleet list. Nothing is just "deployment failed".
- Issues are sorted error → warning → info, then oldest first, so the top of
  the list is the most actionable problem. The UI shows eight and offers
  "+N more" instead of hiding the rest.
- "Copy report" on the dashboard copies the issues with their advice; the
  Diagnosis dialog has the same button for the per-device chain.
- The "N checks not run" line is deliberately absent until a checks system
  exists; an issue list cannot honestly claim checks have run.

### Live verification

Verified against a scratch master (`GLOSKI_PORT=18080`,
`GLOSKI_MASTER_AGENT_GRPC_HOST=127.0.0.1`) and the test SSH server from
`internal/provisioning/sshtest/sshd`:

- healthy local agent → `status: ok`, build hash matches;
- registered agent killed → critical headline with no session;
- deployed agent started, stopped on purpose → warning "stopped on purpose";
- SSH server stopped → critical SSH port step with `connection refused`;
- SSH host key changed after restarting the test server → critical login step
  with the "Trust new key" advice, and the trust action repaired it;
- installed binary deleted → critical install step naming `/tmp/gloski`;
- a failed deployment → dashboard issue with advice and a route.

Each path was checked by calling the endpoint and reading the JSON, not only by
unit tests. Two bugs found this way (duplicate `login` step, blank install
directory) are fixed and covered by tests.
