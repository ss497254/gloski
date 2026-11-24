# Fleet health: closing the SMDWin UI gap

Patterns borrowed from SMDWin (GPL-3.0): **concrete ideas, never code**. Its
source lives at `/tmp/opencode/SMDWin` while this work is in progress.

## Why this plan exists

Tasks 1–5 delivered correct facts and rules, but the dashboard still looks and
feels almost unchanged on a healthy fleet. SMDWin's strongest quality is not
its data model — it is the screen that answers “is everything OK?” at a glance.
We left that screen (our task 6) out. This plan brings the visible part forward.

## What SMDWin does that we are missing

| SMDWin pattern | Source | Our gap |
|---|---|---|
| Verdict card: “x of y passed”, plain-language state, legend counting OK / to check / critical / info / not tested | `views/HealthCheckWindow.cs` (`BuildVerdictCard`, `RefreshSummaryCore`) | Dashboard has four bare metric tiles, one of them just “Issues” |
| “What to do, most important first”: item count, critical first, advice under serious rows, explicit action button, “+N more”, positive “Nothing needs your attention” state | `views/HealthCheckWindow.cs` (`BuildAttention`, `BuildAttentionCard`) | Flat issue list with advice text and a clickable row; no positive state, no action labels |
| Four group cards with count, status label and one coloured block per check; grey when some have not run, green only when all ran and passed | `views/HealthCheckWindow.cs` (`Groups`, `FillStrip`) | A `FleetCheckMatrix` component exists, but with no configured probes it renders five columns of `—`. The most recognisable view is effectively empty |
| Status marks: filled circle with tick / ! / i, not coloured pills; plain words in normal text colour unless warning | `views/StatusMark.cs`, CHANGELOG 0.4.2 | We use bare coloured dots |
| Detail tooltip on every row; right-click copies one row; Copy report grouped by section | `views/HealthCheckWindow.cs` (`Details`, `CopyRow`, `CopyReport`) | Detail only inside a modal; no per-row copy |
| Short diagnosis summary attached to the failing line automatically | `Services/NetworkDiagnosisService.cs` (`ShortSummary`) | Diagnosis hidden behind an icon-only button |
| Read-only badge, last-run time and duration on the health screen | `views/HealthCheckWindow.cs` (`_txtStatus`, verdict footer) | No “last checked” context |
| Sectioned self-test report with warn/fail counts | `Services/SelfTestService.cs` | Flat list in a Settings tab |
| Cached results with “result from HH:mm” and Re-run | `views/HealthCheckWindow.cs` (`RunAsync`) | Self-test runs only on demand, no visible freshness |

Explicitly **not** borrowed: the health score / dial, alerts and notification
centre, Windows-only diagnostics, stress tests, and per-failure repair flows.

## Plan

### Phase 1 — Fleet health screen (do first)

1. Make the existing matrix useful without configured probes. The backend
   already returns `check_matrix` and `FleetCheckMatrix` renders it, but no
   results means no blocks. Add **built-in signals** derived from facts the
   dashboard already loads — connectivity (session), health (CPU/memory),
   agent (deployment/build), jobs (recent failures) — so every device row has
   real blocks. Stored check results still take priority when they exist.
2. Add a `verdict` object to `GET /api/dashboard` with counts of `ok`,
   `warning`, `critical` and `unknown`, derived from the matrix blocks and
   issues. Counts only — no score. Keep `issues` unchanged for compatibility.
   Nothing in this path opens SSH or a new transport.
3. Build `FleetHealth.tsx` on the dashboard:
   - Verdict card with counts and plain-language state, “Start with the first
     item” hint, and a “last updated” line. No numeric score.
   - Attention card with an item count, explicit action button labels, the
     positive “Nothing needs your attention” state, and “+N more”.
   - Matrix: devices as rows, four group columns, coloured strips from the
     new cells; red rows first. Clicking a cell opens the device page.
4. Make Diagnose a labelled button in the fleet list and show a short
   connection headline under offline devices without opening the dialog.
5. Docs: `http-api.md`, `frontend.md`, `data-model.md` if needed.

### Phase 2 — Make existing features visible

6. Master health card always present on the dashboard, with counts and
   “All clear” when healthy; clicking opens Settings → Master health.
7. Self-test report with sections and warn/fail counts, mirroring
   `SelfTestService` rather than a flat list.
8. Status marks (tick / ! / i in a circle) and detail tooltips in the
   attention list and matrix; per-row copy.

### Phase 3 — Only when the underlying features exist

9. Full fleet matrix with saved checks once checks are configured (the
   backend and `FleetCheckMatrix` already handle this).
10. Sparklines, diagnostic bundle, plain-language error table, service problem
    grouping — the ideas already parked in `ideas-from-smdwin.md`.

## Status

Phase 1 is done. The dashboard now returns a verdict and a populated matrix;
the attention card is always present with a positive state and labelled
actions; Diagnose is labelled in the fleet list.

## Full SMDWin inventory highlights

A wider scan of SMDWin (`README.md`, `CHANGELOG.md`, `Services/`, `views/`)
added these patterns to the backlog. Ranked by visible impact:

1. Health screen with verdict + group strips (`views/HealthCheckWindow.cs`).
2. Windows-style status marks: ✓ / ! / i in a filled circle, never colour alone
   (`views/StatusMark.cs`).
3. Sidebar category strips with per-page badges (`views/MainWindow.xaml`,
   `MainWindow.Navigation.cs`).
4. One row per component: value, short history, state (`MainWindow.DashRows.cs`,
   `MiniSparkline.cs`).
5. Start cards for empty states: what it does, how long, what it needs, big
   action (`views/StartCard.cs`).
6. Detail panels that show `—` with a reason, never another row's data
   (`MainWindow.AppDetail.cs`).
7. Explicit freshness: “result from 14:32 · run again for fresh values”
   (`HealthCheckWindow.cs`).
8. Copy confirmations on the button itself (“✓ Copied”), plus grouped reports
   (`MainWindow.BenchmarkPage.cs`, `HealthCheckWindow.cs`).
9. Page-scoped shortcuts: Ctrl+F, F5, Ctrl+E, Esc (`MainWindow.WindowChrome.cs`).
10. One search field that filters navigation and settings (`MainWindow.UX.cs`).
11. Responsive fitting by dropping low-priority columns, never clipping text
    (`views/ResponsiveLayout.cs`).
12. One chart grammar and one range menu (`ChartGrid.cs`, `ChartRangeMenu.cs`).
13. Settings as compact cards showing non-default options with a “Default”
    reset (`MainWindow.SettingsPolish.cs`).
14. Calendar-aligned time buckets for event history (`Services/EventTimeline.cs`).
15. Named snapshots and per-item technician notes (`SnapshotService.cs`,
    `AppNotesStore.cs`).

Do not borrow: Windows-local diagnostics, notification centre/alerts, the
single score, single-machine tray UX, any GPL source or assets.

## Rules for this work

- Counts and severities, never a combined score.
- Unknown / skipped are grey and excluded from problem counts; green means
  “ran and passed”, not “no data”.
- Every visual state must carry a plain-language sentence, not just a colour.
- Keep the request path read-only and fast; expensive probes stay on demand.
