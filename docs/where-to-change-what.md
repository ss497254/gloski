# Where to change what

A high-level map of subsystems for common changes.

## Subsystems

| Area | Where |
|---|---|
| Master HTTP API, dashboard summary, self-test, lifecycle loop | `server/internal/master/` |
| Master-side gRPC agent stream & command dispatch | `server/internal/master/agentstream/` |
| Device-side agent runtime (jobs, files, transfers, terminal, probes) | `server/internal/agent/` |
| gRPC protocol definitions and generated code | `server/proto/agentv1/` |
| SQLite migrations | `server/internal/database/` |
| Devices, deployments, and fleet events | `server/internal/fleet/` |
| SSH installation and scripts | `server/internal/provisioning/`, `server/internal/agentdist/` |
| Jobs, checks/probes, metrics, actions | `server/internal/jobs/`, `server/internal/checks/`, `server/internal/metrics/`, `server/internal/actions/` |
| Filesystem policy & operations | `server/internal/files/` |
| System stats collection | `server/internal/system/` |
| Config defaults and env overrides | `server/internal/config/` |
| Frontend router, navigation, pages, and feature modules | `web/src/App.tsx`, `web/src/app/`, `web/src/features/` |
| Frontend API clients | `web/src/shared/api/`, `web/src/shared/tower/` |

## Common workflows

- **Add a browser endpoint:** implement and register the route in
  `server/internal/master/`, add the client call in `web/src/shared/api/`
  (device-scoped) or `web/src/shared/tower/` (master-scoped), and update
  [http-api.md](http-api.md).
- **Add an agent command:** update `server/proto/agentv1/agent.proto`,
  regenerate with `make proto`, wire the master send/wait in
  `server/internal/master/agentstream/`, handle the command in
  `server/internal/agent/`, and update [backend.md](backend.md) /
  [http-api.md](http-api.md).
- **Add a frontend page:** add the feature page under `web/src/features/`,
  register its route and navigation entry, keep legacy `/fleet/*` and
  `/servers/*` redirects working if device-scoped, keep the master's SPA route
  list aligned, and update [frontend.md](frontend.md).
- **Change the database schema:** append a new numbered migration in
  `server/internal/database/`, update the owning store package, and update
  [data-model.md](data-model.md).
- **Add a configuration setting:** add the field, env override, and default in
  `server/internal/config/`, add a config test, and update [tower.md](tower.md)
  and [backend.md](backend.md).

## Verification

```bash
make build
cd server && go test ./...
cd server && go test -race ./internal/...
cd web && bun run type-check && bun test && bun run build
```

Also verify transport, lifecycle, job, file, or terminal changes against a
live master and agent before committing.
