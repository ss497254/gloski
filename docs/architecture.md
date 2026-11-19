# Architecture Overview

Gloski is a master/agent control tower for Linux devices on a Tailnet. One
`gloski` master serves the dashboard and owns all state. Every device —
including the master itself — runs a stateless `gloski-agent`.

Terms are defined in [glossary.md](glossary.md). The HTTP contract is in
[http-api.md](http-api.md) and the database in [data-model.md](data-model.md).

## Components

| Component | Responsibility |
|-----------|----------------|
| `gloski` (master) | Serves the SPA and browser API, owns device/job/log state, supervises the local agent, deploys and manages remote agents |
| `gloski-agent` | Executes jobs, serves files and terminal, collects and streams device usage |
| Browser | Talks only to the master over HTTP/WebSocket |

```
Browser ──HTTP/WS──▶ Master ──gRPC──▶ agent on A (master host)
                       │
                       ├───────gRPC──▶ agent on B
                       └───────gRPC──▶ agent on C
```

## Code map

| Package | Side | Role |
|---|---|---|
| `internal/controltower` | Master | `Tower`: dashboard, browser API, gRPC listeners, local agent supervision |
| `internal/controltower/agents` | Master | Master half of the agent stream: connections, commands, job dispatch |
| `internal/fleet` | Master | `Registry`: device identities, heartbeats, deployments, event hub |
| `internal/provisioning` | Master | `Installer`: SSH copy/start/stop of remote agents |
| `internal/tailnet` | Master | Tailnet peer discovery via tailscaled LocalAPI |
| `internal/worker` | Device | The `gloski-agent` implementation: `Runtime`, jobs, files, terminal |

`controltower/agents` and `worker` are the two ends of the same gRPC stream.
The word “agent” is product terminology; `worker` is simply the Go package that
implements the device-side binary.

## Communication

The master and agents use a single bidirectional gRPC `Agent.Connect` stream.

- Local agent: gRPC over `unix://<data_dir>/agent.sock`.
- Remote agents: gRPC h2c over Tailnet, port `9090` by default.

Agents send registration, heartbeats/stats, job output, and file/process
results. The master sends jobs, file requests, terminal I/O, and shutdown.
The browser never speaks gRPC; the master translates its HTTP/WS calls into
agent commands.

## State ownership

All durable state lives in the master's SQLite database:

- devices and deployment records
- jobs and their streamed logs
- cached device statistics

Agents keep nothing on disk. Credentials and running jobs live only in memory.
After a restart an agent re-registers with the master and receives a fresh
device token. Jobs survive a dropped stream or a master restart: the agent
keeps running them, buffers their output, and replays it on reconnect.

## Device lifecycle

- **Master device:** the master starts `gloski-agent` as a supervised child
  process. It checks the gRPC session every 5 seconds; if no session exists for
  20 seconds, it kills the process and starts a new one.
- **Remote devices:** the master owns the agent's lifecycle. It installs the
  agent over SSH and then watches every deployed device; when an agent is
  disconnected for 45 seconds, or runs a different build, the master runs the
  install script again. Agents do not restart or update themselves. See
  [tower.md](tower.md).

## Data flows

**Stats** — agent collector → heartbeat on gRPC stream → master DB → dashboard
and `/api/fleet/ws`.

**Jobs** — browser `POST /api/devices/:id/api/jobs` → master stores job →
`RunJob` on gRPC stream → agent executes → output chunks streamed back →
master persists logs → browser polls logs.

**Files / terminal** — browser HTTP/WS → master → matching gRPC request →
agent. The master injects the device token, so the browser never holds an agent
credential.

## Web routes

| Path | Purpose |
|------|---------|
| `/` | Dashboard: device summary, issues, recent tasks |
| `/devices` | Device list and deployment |
| `/devices/:deviceId` | Device overview |
| `/devices/:deviceId/files` · `/terminal` · `/jobs` | Device tools |
| `/settings` | Theme settings |

Legacy `/fleet/*` and `/servers/*` URLs redirect to the `/devices/*` routes.

## Configuration

`gloski` runs in `master` mode by default. `gloski-agent` always runs in
`agent` mode. Tailnet peers come from tailscaled's LocalAPI over its Unix
socket; no `tailscale` CLI call is made. SSH uses `golang.org/x/crypto/ssh`.

See [Control Tower](tower.md) for the full configuration reference.

## Security model

- Keep the dashboard on the Tailnet; use `tailscale serve` for HTTPS.
- The agent gRPC port is h2c and must not be exposed off-Tailnet.
- New SSH host keys are trusted once and stored in `<data_dir>/known_hosts`.
- An optional `GLOSKI_API_KEY` or JWT public key protects the browser-facing
  master API.

## Future work

- Managed services and long-running processes per device
- Cross-device file copy as a first-class task
- Support for devices outside Tailnet
- Optional master-to-agent TLS/mTLS
