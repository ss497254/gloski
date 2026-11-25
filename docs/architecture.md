# Architecture Overview

Gloski is a master/agent control tower for Linux devices on a Tailnet. One
`gloski` master serves the dashboard and owns all state. Every device —
including the master itself — runs a stateless `gloski-agent`.

Terms are defined in [glossary.md](glossary.md). The HTTP contract is in
[http-api.md](http-api.md) and the database in [data-model.md](data-model.md).

## Components

| Component | Responsibility |
|---|---|
| `gloski` (master) | Serves the SPA and browser API, owns device/job/log/check/metric state, supervises the local agent, deploys and manages remote agents |
| `gloski-agent` | Executes jobs, serves files and terminal, runs native probes, collects and streams device usage |
| Browser | Talks only to the master over HTTP/WebSocket |

```
Browser ──HTTP/WS──▶ Master ──gRPC──▶ agent on A (master host)
                       │
                       ├───────gRPC──▶ agent on B
                       └───────gRPC──▶ agent on C
```

## Communication

The master and agents use a single bidirectional gRPC stream.

- Local agent: gRPC over a Unix socket (`<data_dir>/agent.sock`).
- Remote agents: gRPC h2c over Tailnet, port `9090` by default.

Agents send registration, heartbeats/stats, job output, file/process results
and native check results. The master sends jobs, file operations, transfers,
terminal I/O, probe requests and shutdown. The browser never speaks gRPC; the
master translates its HTTP/WS calls into agent commands.

File contents do not travel on the stream. For a download or upload the master
issues a one-time transfer ID on the stream, and the agent makes one plain HTTP
request to `/agent/transfer/{id}` on the same agent port, posting a download or
fetching an upload. Listing, stat, rename, delete and bounded editor text reads
stay on gRPC.

## State ownership

All durable state lives in the master's SQLite database:

- devices and deployment records
- jobs and their streamed logs
- check results and metric history samples
- device action history

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
  [Control Tower](tower.md).

## Data flows

**Stats** — agent collector → heartbeat on gRPC stream → master DB (plus
5-minute metric sampling) → dashboard and `/api/fleet/ws`.

**Jobs** — browser `POST /api/devices/{id}/api/jobs` → master stores job →
gRPC run command → agent executes → output chunks streamed back → master
persists logs → browser polls logs.

**Files / terminal** — browser HTTP/WS → master → matching gRPC request →
agent. Downloads and uploads switch to one HTTP request on the agent port;
small operations stay on the stream. The master injects the device token, so the
browser never holds an agent credential.

## Web routes

| Path | Purpose |
|---|---|
| `/` | Dashboard: fleet health verdict, needs attention, check matrix, devices, recent tasks |
| `/devices` | Device list and agent deployment |
| `/devices/:deviceId` | Device overview, metric history charts, processes, recent changes |
| `/devices/:deviceId/files` · `/terminal` · `/jobs` | Device tools |
| `/settings` | Appearance, Master health self-test, About |

Legacy `/fleet/*` and `/servers/*` URLs redirect to the `/devices/*` routes.

## Configuration

`gloski` runs in `master` mode; `gloski-agent` always runs in `agent` mode.
Tailnet peers come from tailscaled's LocalAPI over its Unix socket; no
`tailscale` CLI call is made. Remote provisioning runs over SSH directly from
the master.

See [Control Tower](tower.md) for the full configuration reference.

## Security model

- Keep the dashboard on the Tailnet; use `tailscale serve` for HTTPS.
- The agent gRPC port is h2c and must not be exposed off-Tailnet.
- New SSH host keys are trusted once and stored in `<data_dir>/known_hosts`.
- An optional `GLOSKI_API_KEY` or JWT public key protects the browser-facing
  master API.
