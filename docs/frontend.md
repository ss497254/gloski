# Frontend Architecture

The dashboard is a React 19 single-page app served by the master from
`web/dist`. The browser only talks to the master; device calls are proxied to
agents over gRPC.

## Tech stack

| Technology | Purpose |
|---|---|
| React 19 + TypeScript | UI |
| Vite | Dev server and production build |
| TailwindCSS v4 | Styling via theme tokens |
| shadcn/ui + Radix | Buttons, cards, tables, badges, alerts, dialogs, menus, tooltips |
| Zustand | Settings and device client list |
| Valtio | Reactive device status in the sidebar |
| lucide-react | Icons |
| xterm.js | Terminal emulator |

## Directory layout

```
web/src/
├── App.tsx                    # Router
├── app/navigation.ts          # Nav items and devicePath() helper
├── features/
│   ├── dashboard/             # Home dashboard and settings
│   ├── devices/               # Fleet list, device overview, deploy dialog
│   ├── files/                 # File browser and previews
│   ├── jobs/                  # Ad-hoc jobs and logs
│   └── terminal/              # xterm.js terminal
├── layouts/
│   ├── AppLayout.tsx          # Sidebar + outlet
│   ├── DeviceLayout.tsx       # Provides current device to device routes
│   ├── PageLayout.tsx         # Common page header/content
│   └── Sidebar/               # Nav, device list, device tabs, theme toggle
├── shared/
│   ├── api/                   # SDK: GloskiClient, Result<T>, resources
│   ├── components/            # Shared UI helpers (UsageBar, StatusBadge, etc.)
│   ├── context/DeviceContext  # Resolves current device from route
│   ├── lib/device.ts          # DeviceClient (per-device API client)
│   ├── store/devices.ts       # useDevicesStore, synced from master
│   ├── tower/                 # Master detection, fleet socket, API calls
│   └── services/              # Legacy health check
└── ui/                        # shadcn/ui primitives
```

`web/src/ui/` holds the shadcn primitives. Add more with
`scripts/add-shadcn.sh <name>` from the repo root — never call the CLI directly.
The wrapper exists because `bunx shadcn@latest add` imports `cn` from an npm
package, re-adds that dependency on every call, and overwrites `badge.tsx`
(losing its status variants). The wrapper fixes the import and dependency and
then runs type-check, lint and `review-checks.sh`.

The conventions are enforced by `web/tests/ui-conventions.test.ts`. It runs as
part of `bun test`, and `bun run build` runs it first via `prebuild`, so even
`make build-web` fails if the CLI's `cn` import or dependency returns, or if
Badge's status variants are lost. The pre-commit hook and `review-checks.sh`
run the same test. If a generated file exports a `cva` variant alongside
a component, move the variant into its own file (`toggle-variants.ts`) or lint's
`react-refresh/only-export-components` rule fails.

The primitives in active use are: button, card, badge, alert, input,
native-select, label, input-group, switch, toggle-group, table, tabs, dialog,
alert-dialog, breadcrumb, dropdown-menu, context-menu, tooltip, separator,
scroll-area, progress, skeleton, spinner, empty and sonner. Prefer them over
hand-rolled markup: form labels and selects, search fields with icons,
confirmation dialogs, breadcrumbs, progress bars and empty states all have a
primitive now. On top of them, `shared/components/status.tsx` provides
`StatusDot` and `StatusBadge`, and `shared/lib/status.ts` maps the API's status
strings to a shared `StatusTone`. Use these instead of hand-rolled dots and
coloured spans so a device looks the same in the sidebar, fleet table,
dashboard cards and device page.

## Routes

| Path | Purpose |
|---|---|
| `/` | Dashboard: device summary, issues, recent tasks |
| `/devices` | Device list and agent deployment |
| `/devices/:deviceId` | Device overview: CPU, memory, network, disks, processes |
| `/devices/:deviceId/files` | File browser |
| `/devices/:deviceId/terminal` | Terminal |
| `/devices/:deviceId/jobs` | Jobs and logs |
| `/settings` | Theme settings |

`/fleet/*` and `/servers/*` URLs redirect to the `/devices/*` routes for old bookmarks.

## State and data flow

- `TowerProvider` detects the master, subscribes to `/api/fleet/ws`, and keeps the
  device list. It also calls `syncDevices()` to maintain client entries.
- `useDevicesStore` holds `DeviceClient` objects; each one wraps `GloskiClient`
  with `apiPrefix: /api/devices/{id}/api`.
- `DeviceProvider` resolves the `:deviceId` route param into the current
  `DeviceClient` for device-scoped feature pages.
- `useSettingsStore` persists theme and sidebar state. Feature-local stores
  handle files and jobs.
- Stats on the device overview come from `useDeviceStats`, which opens the
  master's stats WebSocket for that device.

## API client

`shared/api` is the SDK used by all feature pages. Calls are wrapped in
`Result<T>`; pages use `unwrap()` and handle errors at the call site.

`shared/tower/client.ts` contains master-level endpoints: detection, fleet
WebSocket, tailnet peers, deployment, and device lifecycle. `diagnoseDevice(id)`
calls `POST /api/devices/{id}/diagnose` and returns a `Diagnosis`: an overall
status, a headline, one sentence of advice and an ordered list of `steps`. Step
statuses are `ok`, `warning`, `critical` and `skipped`; skipped is neutral. The
`DiagnoseButton` in `features/devices/components/DiagnoseDialog.tsx` opens a
dialog that runs the diagnosis on open, shows the headline and advice first,
then each step's summary with its raw detail, and can re-run or copy the report.
The dashboard's issues use the same advice-first shape: `DashboardIssue.action`
is the destination route and `advice` is the one-line explanation shown under
the detail. Both the dashboard and the diagnosis dialog have "Copy report".

`getSelfTest()` calls `GET /api/system/self-test`. Settings has a "Master
health" tab (`?tab=health`) with `MasterHealthSettings`: it re-runs the
read-only checklist, shows each line's summary and detail, and copies a report.
The dashboard shows a small Master health card only when the overall status is
`warning` or `critical`, linking to that tab. A skipped line is never shown as
a failure.

`listDeviceActions(id)` and `undoDeviceAction(id, actionId)` back the "Recent
changes" card on the device page (`features/devices/components/RecentChanges.tsx`).
Each entry shows the kind, the from → to states, the time and the result. An
Undo button appears only for successful, reversible changes that have not
already been undone; after an undo the card and the fleet state are reloaded.
Logging a failed action is deliberate: a failure is still a reason a person
may need to act.

`getDeviceHistory(id, { metric, range, limit })` calls
`GET /api/devices/{id}/history`. `useDeviceHistory` polls it once a minute and
`BtopView` renders the CPU, memory, network and first-disk series with the
shared `MetricsChart`. The pure adapter is
`features/devices/lib/history.ts`; it selects one series and computes a stable
axis ceiling, with colocated `bun:test` tests. History is independent of the
live stats WebSocket, which still drives the current-value cards.

Every call made through `shared/tower/client.ts` logs a request id, method,
path, duration and capped JSON bodies to the browser console. The same id is
sent as `X-Request-ID` and returned by the master, so a console line can be
matched to the `api id=...` line in the master log. Uploads, downloads and
credential fields are never printed; see the correlation section in
[http-api.md](http-api.md).

## Theming

`web/src/index.css` defines one semantic token set (background, foreground,
card, muted, primary, success, warning, info, destructive). Components use those
tokens rather than hardcoded colors, so light/dark switching applies everywhere.
The neutral slate theme is deliberate: `primary` is the same near-black in light
and near-white in dark, used for actions, active nav items and the brand mark;
success/warning/info keep their own meanings. Cards use `shadow-sm`. The default
theme is dark; `isDarkTheme()` in the settings store is shared by
theme-dependent code.

## Verification

```bash
cd web
bun run type-check
bun run format:check
bun run lint
bun test src/shared/api
bun run build
```

The production build is served by the master; there is no separate static host.
