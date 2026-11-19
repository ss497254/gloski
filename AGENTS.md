# Gloski: notes for coding agents

This file is the project's shared memory. Any agent (Claude Code, opencode, or
another) should read it before working and update it when a decision, preference
or piece of project state changes. Keep project knowledge here or in `docs/`,
not in a tool's private memory.

## What Gloski is

A homelab "control tower". One `gloski` master serves the dashboard and API; a
stateless `gloski-agent` runs on each device and holds one bidirectional gRPC
stream to the master. Through it you run jobs, open terminals, browse and
transfer files, and view processes and stats on every device.

- Backend: Go 1.24+, module `github.com/ss497254/gloski`, in `server/`.
  Binaries: `cmd/gloski` (master) and `cmd/gloski-agent`.
- Protocol: `server/proto/agentv1/agent.proto`, generated code is checked in.
  The master's own device uses a unix socket (`<data_dir>/agent.sock`); remote
  agents use h2c on port 9090.
- Master state: SQLite through `modernc.org/sqlite`. Pragmas must be written as
  `_pragma=name(value)` in the DSN or they are silently ignored.
- Frontend: React 19, TypeScript, Vite (dev port 3000), Tailwind v4, shadcn,
  xterm.js, managed with Bun, in `web/`.
- Remote agents are installed over SSH (`server/internal/provisioning`) and
  found through Tailscale.

Deeper reading: `docs/architecture.md`, `docs/tower.md`, `docs/backend.md`,
`docs/frontend.md`, `CONTRIBUTING.md`.

## Owner's priorities and preferences

These came from the owner directly. Follow them unless told otherwise.

- **Reliability comes first.** The pain to remove is "I want to do a task and
  the agent is dead, or some random issue blocks me". Things must work, and when
  they cannot, the reason must be stated plainly.
- **Security is not a concern for now.** The project is internal and not public.
  Do not propose or prioritise hardening unprompted, and watch for existing
  protections that block legitimate use. Do not remove existing protections
  without asking.
- **Commit straight to `main`.** No per-feature or per-batch branches. Do not
  push unless asked.
- **The `/tmp` + `nohup` agent install is deliberate.** Agents are installed and
  deleted often while the code changes. Do not flag it as a defect or push
  systemd; the owner will move to systemd when ready.
- **No SSH fallback for actions.** If an agent is down the master should revive
  it, not run the task over SSH instead.
- **No offline alerts** for now.
- The owner likes to settle design through short question-and-answer rounds
  before a large build, and wants a written plan first.

## Deployment context

- The dashboard sits behind a reverse proxy with HTTP basic auth. The master
  listens on localhost only. `GLOSKI_API_KEY` is unset.
- Fleet: x86 servers and mini PCs, Raspberry Pi and other ARM boards, VMs and
  LXC containers, NAS appliances. Agents run as a mix of root and normal users.
- Every device can run Tailscale. By default the master binds the agent port to
  its Tailscale IPs only (`agent_grpc_host` empty) and retries until Tailscale
  is up.
- Workloads: long-running jobs, scheduled jobs, the same command on many
  devices, and quick one-off commands.
- The master rejects cross-origin writes and WebSocket handshakes
  (`httpapi.SameOrigin`). Behind a proxy that does not forward `Host` this can
  return 403; set `GLOSKI_ALLOWED_ORIGINS=*` to turn the check off.

## Current state (2026-11-19)

`main` holds five commits that are not yet pushed to `origin`:

1. `59d6c6a` removed the legacy standalone mode (master mode only).
2. `0adf0c1` origin check, Tailscale-only agent port, JSON content-type check.
3. `fbe2fdc` reliable shutdown, working SQLite pragmas, device lifecycle fixes.
4. `7f86d0a` jobs survive disconnects and master restarts (sequence-numbered
   output, 4 MB replay buffer on the agent, reconciliation at register).
5. `ab0abad` bounded terminal output with resize, streamed ranged file
   download and chunked upload.

Agents built before these commits must be redeployed; the protocol changed.

**Next work is `docs/reliability-plan.md`.** It lists the failure scenarios, the
intended behaviour for each, the build order and the open questions. Nothing in
it is built yet. Update that file as phases land.

Known gaps not covered by the plan:

- The dashboard's master client does not send the API key.
- Device identity is still keyed by name; a machine-id guard only blocks
  duplicates.
- Legacy database tables and config fields from standalone mode remain.

## Working on the code

```bash
make build            # master, agent and frontend
make proto            # regenerate server/proto/agentv1/*.pb.go
cd server && go test ./...
cd server && go test -race ./internal/...
cd web && bun run type-check && bun test && bun run build
```

Run `gofmt` on changed Go files. Verify changes against a live master and agent,
not only unit tests; the earlier batches found real bugs that way.

Live test recipe:

- Run `bin/gloski` with `GLOSKI_DATA_DIR` set to a scratch directory,
  `GLOSKI_PORT=18080` and `GLOSKI_MASTER_AGENT_GRPC_HOST=127.0.0.1`. Without the
  last one the agent port only opens on Tailscale addresses.
- Start extra agents with `GLOSKI_AGENT_MASTER_URL=http://127.0.0.1:<port>` and
  the join token from `<data_dir>/master-join-token`.

Pitfalls already hit:

- `pkill -f <pattern>` can match and kill the shell running it. Use `pkill -x`
  or `pgrep -x`.
- The Vite dev server is on port 3000, not 5173.
- A terminal test that waits for a marker must compute it (for example
  `FLOOD-$((1+1))-DONE`), because the typed command is echoed back.

### The current development machine

These are facts about one machine, not requirements of the project.

- Go 1.27.4 is at `/usr/local/go`, linked into `/usr/local/bin`.
- protoc 3.21.12 with `protoc-gen-go` v1.36.6 and `protoc-gen-go-grpc` v1.5.1
  is under `/root/.local/share/gloski-tools`. These reproduce the checked-in
  generated code byte for byte:
  `PATH=/root/.local/share/gloski-tools/bin:/root/.local/share/gloski-tools/protoc/bin:$PATH make proto`
- `python3` is not installed; script with awk, perl or bun.
- There is no SSH server and Tailscale is not running, so SSH deploy and revive
  cannot be tested live here without installing one.
