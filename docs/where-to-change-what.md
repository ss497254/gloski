# Where to change what

A map for common changes. If a change needs a file that is not listed here, fix
this page in the same commit.

## Add a browser endpoint

1. Handler: add it to the matching file in `server/internal/controltower/`:
   - device commands and jobs: `device_jobs.go`, `device_files.go`,
     `device_terminal.go`, `device_api.go`
   - fleet, dashboard, deployments: `handlers.go`, `dashboard.go`, `deploy.go`
2. Register the route in `server/internal/controltower/routes.go`. Use a method
   pattern such as `GET /api/devices/{id}/api/files`.
3. Decide the response shape (raw, wrapped, error). The three shapes are
   documented in [http-api.md](http-api.md).
4. Add the call to the frontend:
   - device-scoped: `web/src/shared/api/resources/`
   - master-scoped: `web/src/shared/tower/client.ts`
5. Document the method, path, body, response and errors in
   [http-api.md](http-api.md).
6. Test with `go test ./...`, then against a live master.

## Add an agent command

The browser never talks to an agent directly. A command crosses the gRPC stream.

1. Add messages and a payload case to
   `server/proto/agentv1/agent.proto`.
2. Regenerate: `make proto`. The generated code is checked in and must be
   committed with the change.
3. Master side: add a method to `server/internal/controltower/agents/`
   (`commands.go` or a new file) that sends the message and waits for its
   result.
4. Device side: dispatch the new `MasterMessage` case in
   `server/internal/worker/run.go`, and implement it in a focused file:
   `agent_jobs.go`, `agent_files.go`, `agent_terminal.go`, or a new file.
5. Add tests for the store/command logic and a live smoke test through the
   dashboard or HTTP API.
6. Update [backend.md](backend.md) and, if the browser can reach it, the
   endpoint tables in [http-api.md](http-api.md).

## Add a page

1. Create `web/src/features/<name>/` with `pages/<Name>Page.tsx`.
2. Add the route in `web/src/App.tsx`.
3. Add the nav item in `web/src/app/navigation.ts`.
4. Device-scoped pages go under `DeviceLayout` and redirect old `/servers/*`
   and `/fleet/*` URLs as the existing pages do.
5. Update [frontend.md](frontend.md).

## Change the database schema

1. Add a migration to `server/internal/database/migrations.go`. Versions are
   integers and must only increase; never edit an applied one.
2. Job tables are the exception: they are created at runtime by
   `agents.NewJobStore`. Change both that startup path and, when needed, a
   migration for existing databases.
3. Update the store methods in `server/internal/fleet/` or
   `server/internal/controltower/agents/jobs.go`.
4. Update [data-model.md](data-model.md).
5. Test on a fresh database file and on a copy of an existing one.

## Add a config setting

1. Add the field to `server/internal/config/config.go` with its JSON tag.
2. Add the environment override in `server/internal/config/env.go`.
3. Set the default in `DefaultConfig` or the mode-specific defaults.
4. Update `docs/tower.md` and the configuration table in
   [backend.md](backend.md).
5. Add a case to `server/tests/config/config_test.go`.

## Change file operations

1. Filesystem policy and operations: `server/internal/files/`.
2. Device-side gRPC handling: `server/internal/worker/agent_files.go` and
   `agent_jobs.go`.
3. Browser API translation: `server/internal/controltower/device_files.go`;
   downloads and uploads add `device_transfer.go`.
4. Frontend client and UI: `web/src/shared/api/resources/files.ts` and
   `web/src/features/files/`.
5. Update [http-api.md](http-api.md).

## Add or change a job status

1. Add the constant in `server/internal/controltower/agents/jobs.go`.
2. Update the transitions in `MarkRunning`, `Finish`, `AbandonMissing`,
   `Abandon`, and the queries that filter by status.
3. Update the frontend badge and filters in `web/src/features/jobs/`.
4. Update [data-model.md](data-model.md).
5. Test both the normal finish path and a reconnect that abandons a job.

## Where things live

| I need to understand... | Read |
|---|---|
| The whole system in one page | [architecture.md](architecture.md) |
| HTTP routes and payloads | [http-api.md](http-api.md) |
| SQLite tables and statuses | [data-model.md](data-model.md) |
| The master's browser API and lifecycle | `server/internal/controltower/` |
| The master half of the gRPC stream | `server/internal/controltower/agents/` |
| Device-side jobs, files, terminal | `server/internal/worker/` |
| Devices, deployments, fleet events | `server/internal/fleet/` |
| SSH install and uninstall | `server/internal/provisioning/` |
| Agent builds served by the master | `server/internal/agentdist/` |
| Stats collection | `server/internal/system/` |
| Frontend routes and pages | `web/src/App.tsx`, `web/src/features/` |
| Frontend API client | `web/src/shared/api/`, `web/src/shared/tower/client.ts` |

## Verify a change

```bash
make build
cd server && go test ./...
cd server && go test -race ./internal/...
cd web && bun run type-check && bun test && bun run build
```

Then run a live master and agent using the recipe in `AGENTS.md`. Unit tests
have not caught every transport bug in this project; a smoke test through the
dashboard is part of the definition of done.
