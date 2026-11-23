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
| shadcn/ui + Radix | Dialogs, menus, tooltips, primitives |
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
│   ├── components/            # Shared UI helpers (UsageBar, ProgressBar, etc.)
│   ├── context/DeviceContext  # Resolves current device from route
│   ├── lib/device.ts          # DeviceClient (per-device API client)
│   ├── store/devices.ts       # useDevicesStore, synced from master
│   ├── tower/                 # Master detection, fleet socket, API calls
│   └── services/              # Legacy health check
└── ui/                        # shadcn/ui primitives
```

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

## Theming

`web/src/index.css` defines one semantic token set (background, foreground,
card, muted, primary, success, warning, info, destructive). Components use those
tokens rather than hardcoded colors, so light/dark switching applies everywhere.
The default theme is dark; `isDarkTheme()` in the settings store is shared by
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
