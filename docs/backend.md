# Backend Architecture

The Go backend produces two binaries: `gloski` (the master) and `gloski-agent`
(the device-side agent). They communicate only over gRPC, except for bulk file
transfers which use a one-time HTTP endpoint on the agent port.

Terms are defined in [glossary.md](glossary.md). Routes and payloads are in
[http-api.md](http-api.md); tables and lifecycles are in
[data-model.md](data-model.md).

## Tech stack

| Technology | Purpose |
|---|---|
| Go 1.24+ | Language |
| `net/http` | Browser-facing API, SPA serving, and transfer endpoint |
| `gorilla/websocket` | Browser terminal and fleet/stats streams |
| gRPC + protobuf | Master ↔ agent communication |
| `golang.org/x/crypto/ssh` | In-process SSH for agent installation |
| `modernc.org/sqlite` | Master SQLite database |
| `creack/pty` | PTY for terminal sessions |

## Responsibilities

| Side | Subsystem | Responsibility |
|---|---|---|
| Master | Browser API & SPA | HTTP/WebSocket endpoints, auth/origin/logging middleware, SPA fallback |
| Master | Agent stream & transfer | Tracks connected agent sessions, dispatches commands, bridges HTTP file transfers |
| Master | Fleet & deployments | Device registry, heartbeats, SSH deployment records, live fleet event fan-out |
| Master | Lifecycle & provisioning | Local child-process supervision, 15s remote health/reinstall loop, SSH installer |
| Master | Tailnet discovery | Reads tailnet peers directly from `tailscaled` LocalAPI |
| Master | Jobs, checks, metrics, actions | Persists job logs, probe/check results, 5-minute metric samples, and device action undo log |
| Master | Maintenance | Daily pass pruning finished jobs (>30d), thinning check history (>20/pair) and metric samples (7d full / 365d daily) |
| Agent | Agent runtime | Connects to master, runs jobs with output replay, handles small file ops, streams HTTP transfers, manages PTYs, runs native probes, collects stats |

## Connection lifecycle

1. Master starts gRPC listeners: a Unix socket for its own agent and TCP h2c
   for remote agents (sharing the port with plain HTTP health and transfer
   endpoints).
2. Master starts `gloski-agent` as a supervised child process pointed at the
   Unix socket.
3. Remote agents are installed over SSH using embedded binaries (`amd64`,
   `arm64`, `arm`) and pointed at the master's TCP listener.
4. The agent opens the bidirectional gRPC stream and registers with the join
   token, machine identity, architecture, build hash, and protocol version.
5. Master registers the device, issues a fresh device token, and binds the
   stream to that device ID.
6. Agent sends periodic heartbeats with system stats; master updates the device
   record, samples metrics (at most once per 5 minutes), triggers due native
   probes if the dashboard was recently active, and broadcasts fleet updates.
7. If the stream drops, the agent reconnects with backoff, reports jobs it
   still holds, and replays any missing output chunks. If the local agent has no
   session for 20 seconds, master kills and restarts it.

## Request flows

```
Browser ──HTTP/WS──▶ Master HTTP API ──gRPC command──▶ Agent runtime
```

- **Stats** — agent collector → heartbeat → master DB → dashboard and fleet
  WebSocket (live stats WebSocket pushes cached stats every 2 seconds).
- **Native probes** — configured via `GLOSKI_PROBES`; triggered on heartbeat →
  agent runs TCP/HTTP/DNS/TLS probe → bounded aggregation → stored as check
  results → fleet check matrix. Probes pause after 3 minutes without a
  dashboard or device-stats request.
- **Jobs** — browser POST → master persists job → agent executes → sequenced
  output chunks stream back → master stores logs (capped at 10 MB per job).
- **Files** — listing, stat, rename, delete, and small text reads/writes (up to
  10 MB) cross the gRPC stream. Downloads and uploads use a one-time transfer
  ID: the agent makes a plain HTTP request to `/agent/transfer/{id}` on the
  agent port so file bytes never block terminal or job traffic. Uploads replace
  their target atomically only after the full body arrives.
- **Terminal** — WebSocket text frames forward keystrokes and 5-byte binary
  frames resize the PTY; agent PTY output streams back into a bounded session
  buffer so a slow browser cannot stall the stream.

## State

All durable state lives in the master's SQLite database (WAL mode, foreign keys
on, immediate transactions):

| Table | Contents |
|---|---|
| `devices` | Device identity, token, URL label, cached stats, master flag, deployment link |
| `deployments` | SSH deployment records and status |
| `check_results` | Latest and historical check/probe verdicts |
| `device_actions` | Master-owned device actions and undo state |
| `metric_samples` | 5-minute metric history and daily rollups |
| `agent_jobs` / `agent_job_logs` | Master-owned jobs and streamed output |
| `schema_migrations` | Applied migrations |

Agents hold no durable state. Credentials and running jobs live only in memory.

## Configuration

Configuration comes from an optional JSON file, then `GLOSKI_*` environment
overrides. See [Control Tower](tower.md) for the complete environment variable
reference.

## Build and test

```bash
cd server
go test ./...
go vet ./...
```

Build with `make build-server` from the repository root, which cross-builds
agents for `amd64`, `arm64`, and `arm` and embeds them into `gloski`. A plain
`go build ./cmd/gloski` works too, but that master can only install agents if a
`gloski-agent` binary sits next to it.

Generated protobuf code is checked in under `server/proto/agentv1/`, so
`protoc` is not required to build. After editing the proto definition, run
`make proto`.
