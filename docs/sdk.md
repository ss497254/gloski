# Browser API Client

The dashboard uses a typed TypeScript client inside `web/src/shared/api`
alongside master-level helpers in `web/src/shared/tower` to talk to the master's
HTTP and WebSocket endpoints. There is no standalone npm package; it is built
and consumed directly within the web app.

## Design

- **Non-throwing results:** request methods return a discriminated result
  (`{ data, error }`) so callers handle HTTP, auth, and network failures without
  `try/catch`.
- **Per-device scoping:** each enrolled device gets a client instance whose API
  prefix points at `/api/devices/{id}/api`, translating browser calls into
  master-proxied agent commands.
- **Request correlation:** master-level calls attach `X-Request-ID` headers and
  log redacted request/response summaries to the browser console so clicks can
  be matched against master logs.

## Active vs. legacy stubs

Only endpoints backed by the master's HTTP router are live:

| Area | Status on master | Capabilities |
|---|---|---|
| Master & fleet | Active | Info, dashboard summary, fleet WebSocket, tailnet peers, deploy/start/stop/trust-host, diagnose, action history & undo, device metric history, master self-test |
| System | Active (`stats`, `processes`, `stats/ws`) | Current stats, live 2-second stats WebSocket, running processes (device-scoped `status`, `history`, and `info` methods on the client are unbacked legacy stubs) |
| Files | Active (except `pinned`) | Directory listing, bounded text read, write, mkdir, rename, delete, streaming upload, and Range-capable download (`files.pinned` has no backend route) |
| Jobs | Active | List, start, inspect, fetch logs, stop, delete |
| Terminal | Active | Interactive WebSocket PTY with auto-reconnect and resize frames |
| Legacy stubs | Unimplemented (no backend routes) | `auth`, `search`, `packages`, `cron`, `downloads` |

## Verification

```bash
cd web
bun run type-check
bun test src/shared/api
```
