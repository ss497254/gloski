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
`docs/frontend.md`, `CONTRIBUTING.md`. `docs/ideas-from-smdwin.md` inventories
patterns from a Windows diagnostics tool; `docs/beszel/` is a topic-by-topic
sourcebook for borrowing from Beszel (MIT; its frontend is React 19, so UI code
is directly portable). Start at `docs/beszel/README.md`.

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
- **Installing agents under `/tmp` is deliberate.** Agents are installed and
  deleted often while the code changes. Do not flag it as a defect or push
  systemd; the owner will move to systemd when ready.
- **Keep it simple.** The owner pushed back when a plan grew a mechanism per
  failure case. Prefer one general repair (restart, reinstall) over many
  special ones, and leave rare cases out until they actually happen.
- **The owner wants decisions made, not menus.** When a design question comes
  up, pick the simplest approach that covers the failure cases, explain it
  briefly and build it ("use your brain"). Ask only when the answer cannot be
  worked out.
- **The master owns the agent lifecycle.** Not the device and not the agent:
  devices differ and may lack systemd or cron. The master installs, starts,
  restarts and updates agents over SSH; an incompatible agent just exits and
  the master runs the right one. Details in `docs/roadmap.md`.
- **Agents are never installed by hand.** Every device is deployed from the
  dashboard. Do not add manual install paths.
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
- Tailscale SSH authenticates deployment without SSH keys: `x/crypto/ssh` tries
  "none" auth first and the tailnet accepts it. Keys remain a fallback for
  devices without Tailscale SSH. This is the rescue path when an agent is down
  and no key is configured; never add a separate SSH fallback for tasks.
- Workloads: long-running jobs, scheduled jobs, the same command on many
  devices, and quick one-off commands.
- The master rejects cross-origin writes and WebSocket handshakes
  (`httpapi.SameOrigin`). Behind a proxy that does not forward `Host` this can
  return 403; set `GLOSKI_ALLOWED_ORIGINS=*` to turn the check off.

## Current state (2026-11-19)

`main` has local commits not yet pushed to `origin/main`.

The docs map is written: glossary, HTTP API reference, data model,
where-to-change-what, and a `CONTRIBUTING.md` that matches the main-only
workflow. The files service is split by operation, worker file handling lives in
`agent_files.go`, and delete and rename share one protected-path list. Delete
refuses whole trees (except `/tmp`); Rename refuses only the roots, so renames
inside system directories stay legal.

File contents no longer travel on the gRPC stream. A download or upload sends a
`FileTransfer` with a one-time id, and the agent makes one HTTP request to
`/agent/transfer/{id}` on the same agent port (unix socket for the local agent).
`ProtocolVersion` is 4 (native check probes); agents built before this change
are refused and reinstalled by the lifecycle. Listing, stat, rename, delete and
the editor's small text reads stay on gRPC.

Job logs are capped at 10 MB each and a daily maintenance pass in
`controltower/maintenance.go` deletes finished jobs older than 30 days
(`agents.JobRetention`). Device metric history, the fleet check matrix,
native TCP/HTTP/DNS/TLS probes and the installer/SPA hardening are in; see
`docs/beszel/11-implementation-plan.md` for what was done and what stays parked.

The master-owned agent lifecycle is built (`7b936a3`), and the protocol changes
from `7f86d0a` and `ab0abad` are in. Agents built before those commits must be
reinstalled; the install script removes an old `/tmp/gloski-agent` install by
itself.

**Code readability work is in `docs/code-quality-plan.md`.** Batches 1 and 2
are done; batch 3 (shrink `Tower`) is next when quality work resumes.

**Features are in `docs/roadmap.md`**, in order; start at the first unfinished
item of step 1.

**The reliability design is in `docs/reliability-plan.md`**: the master owns
the agent lifecycle and reinstalls over SSH whatever is disconnected or
outdated. Its Status section says what is verified and how to test it here.

Known gaps not covered by either plan:

- The dashboard's master client does not send the API key.
- Device identity is still keyed by name; a machine-id guard only blocks
  duplicates.
- Legacy database tables and config fields from standalone mode remain.
- The files package still carries unused chunked-upload and search code.
- `Tower` and `FilesContext` are still the god objects described in the
  readability plan.

## Working on the code

```bash
make build            # master (agents for 3 architectures embedded) and frontend
make proto            # regenerate server/proto/agentv1/*.pb.go
cd server && go test ./...
cd server && go test -race ./internal/...
cd web && bun run type-check && bun test && bun run build
```

Run `gofmt` on changed Go files. Verify changes against a live master and agent,
not only unit tests; the earlier batches found real bugs that way.

Before committing, run `/review` in OpenCode (or `scripts/review-checks.sh`
directly). The read-only `reviewer` subagent in `.opencode/agents/reviewer.md`
checks the recurring rules above; fix High findings before committing. When a
mistake recurs, add it to that checklist instead of relying on memory. A
`pre-commit` hook in `scripts/hooks/` enforces this: it runs the mechanical
checks and the reviewer agent, and blocks on a failed check or `VERDICT:
BLOCK`. Enable it per clone with `scripts/install-git-hooks.sh`.

API calls are correlated with `X-Request-ID`: the dashboard console and the
master's `api id=...` log line share the id. Bodies in logs are capped, JSON
only and redacted; file contents and uploads are never logged. Set
`GLOSKI_LOG_LEVEL=debug` to include read requests.

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
- There is no SSH server and Tailscale is not running. Use the test SSH server
  described in `docs/reliability-plan.md` to deploy agents locally.
