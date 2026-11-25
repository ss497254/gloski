# Frontend Architecture

The dashboard is a React 19 single-page app served by the master from
`web/dist`. The browser only talks to the master; device calls are proxied to
agents over gRPC.

## Tech stack

| Technology | Purpose |
|---|---|
| React 19 + TypeScript | UI |
| Vite | Dev server and production build |
| TailwindCSS v4 | Styling via semantic theme tokens |
| shadcn/ui + Radix | UI primitives (buttons, cards, tables, badges, alerts, dialogs, menus, tooltips) |
| Zustand + Valtio | Persisted UI settings and reactive fleet device state |
| lucide-react | Icons |
| xterm.js | Terminal emulator |
| Chart.js | Device metric history charts |

## UI primitives and conventions

Shared UI primitives live in `web/src/ui/`. Always add new primitives with
`scripts/add-shadcn.sh <name>` from the repository root — never call the
shadcn CLI directly, because it overwrites customized badge status variants and
breaks import conventions. Build-time and review tests enforce these rules.

Prefer shared primitives and shared status indicators over hand-rolled markup
so device and check states look consistent across the sidebar, tables,
dashboard cards, and device pages.

## Routes

| Path | Purpose |
|---|---|
| `/` | Dashboard: fleet health verdict, needs attention, master health, check matrix, devices, recent tasks |
| `/devices` | Device list and agent deployment |
| `/devices/:deviceId` | Device overview: live stats, metric history charts, processes, recent changes |
| `/devices/:deviceId/files` | File browser and text/media preview |
| `/devices/:deviceId/terminal` | Interactive PTY terminal |
| `/devices/:deviceId/jobs` | Ad-hoc jobs and logs |
| `/settings` | Appearance, Master health self-test, About |

`/fleet/*` and `/servers/*` URLs redirect to the `/devices/*` routes for old
bookmarks.

## Data flow and features

- **Fleet sync:** on load, the app detects the master, subscribes to the fleet
  WebSocket (`/api/fleet/ws`), and keeps the live device list synchronized.
- **Dashboard & fleet health:** displays a verdict card with counts (`ok`,
  `warning`, `critical`, `unknown` — never a combined numeric score), an
  always-present "Needs attention" card with factual details, advice, and
  explicit actions, a Master health summary card, and the fleet check matrix.
  The matrix shows devices × check groups (`Connectivity`, `Health`, `Agent`,
  `Jobs`, etc.), sorting red rows first while keeping skipped, stale, and
  never-run states neutral grey.
- **Devices list:** supports sorting, column visibility, page size selection,
  pagination, sticky headers, inline SSH/host-key actions, and one-click
  connection chain diagnosis.
- **Device overview:** combines a live stats WebSocket (2-second updates for
  current CPU, memory, network, disks, and processes) with 1-minute polled
  metric history charts (with gap breaks for missing intervals) and a "Recent
  changes" action card supporting undo for reversible agent actions.
- **Master health:** Settings → Master health displays the read-only self-test
  checklist grouped by section (`Data & storage`, `Agents & builds`, `Other`)
  with per-section summaries and a copyable report.
- **Request correlation:** every master API call sends an `X-Request-ID` header
  and logs request ID, method, path, duration, and redacted JSON summaries to
  the browser console so browser actions match `api id=...` lines in the master
  log.

## Navigation

The desktop and mobile sidebar groups links into labelled sections: **Fleet**
(Dashboard, Devices), **This device** when viewing a device route (or
**Devices** showing the first five devices otherwise), and **Master** in the
footer. Settings shows a live warning/failed badge from the master self-test,
while device availability stays off the sidebar to avoid offline alert noise.

## Theming

Semantic color tokens (`background`, `foreground`, `card`, `muted`, `primary`,
`success`, `warning`, `info`, `destructive`) live in the global stylesheet so
light/dark switching applies consistently. The default theme is dark, and a
pre-paint script applies the saved theme before first render to avoid flash.

## Verification

```bash
cd web
bun run type-check
bun run format:check
bun run lint
bun test
bun run build
```

The production build is served directly by the master. Known browser routes are
kept in sync between the router and the master's SPA handler so known routes
return `200`, unknown extensionless paths return `404` while still rendering
the SPA not-found flow, and missing assets or `/api/*` routes return `404`.
