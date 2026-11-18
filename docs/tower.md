# Control Tower

Gloski is master/agent only. One `gloski` master serves the dashboard and keeps
all state; `gloski-agent` runs on every device, including the master host.

```
Browser ──HTTP/WS──▶ Master (dashboard + state + proxy)
                         │ gRPC (one bidi stream per device)
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
   agent on A        agent on B        agent on C
   (master host,     (/tmp, SSH)       (/tmp, SSH)
    child process)
```

A, B, and C all appear in Fleet. The master device is badged **Master**. Every
dashboard action goes to the master, which forwards it to the right agent.

## Communication

- **Master ↔ agent:** gRPC only, one bidirectional `Agent.Connect` stream.
  - Local agent: gRPC over `unix://<data_dir>/agent.sock`.
  - Remote agents: gRPC h2c over Tailnet on `0.0.0.0:9090` by default.
  - The agent sends registration, heartbeats/stats, job output, and file/process
    results; the master sends jobs, file requests, terminal I/O, and shutdown.
- **Browser → master:** plain HTTP/WS. The master translates those calls into
  gRPC commands. No HTTP agent API exists anymore.

## Ephemeral agents

Agents keep no state. There is no `agent.json`, no local database, and no
job log files on devices. Credentials and in-flight job maps live in memory
only. On restart the agent re-registers with the master and receives a fresh
device token. If an agent dies, the master marks its jobs abandoned and keeps
the logs.

Installing an agent means copying the binary to `/tmp/gloski-agent` over SSH
and starting it with `nohup`. It will be gone after a reboot — that is
intentional for now.

## Master-owned state

The master database stores devices, deployments, jobs, and streamed job logs.
Cached device stats arrive through heartbeats. Job output is streamed from the
agent over gRPC and persisted by the master, so it survives agent restarts.

## Local agent lifecycle

Master starts `gloski-agent` as a **child process** (no shell). If it exits
unexpectedly, master logs it and restarts it with backoff; the dashboard stays
up. On master shutdown, the child is terminated first. Set
`GLOSKI_MASTER_LOCAL_AGENT=false` to disable it.

## Quick start

```bash
make build

GLOSKI_MODE=master \
GLOSKI_HOST=0.0.0.0 \
GLOSKI_PORT=8080 \
GLOSKI_MASTER_WEB_DIR="$PWD/web/dist" \
bin/gloski
```

Open the dashboard, choose **Deploy agent**, pick a tailnet peer, and provide
the SSH user. SSH keys come from `SSH_AUTH_SOCK` or `~/.ssh/id_*`.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_MODE` | `master` | `gloski` runs as master by default |
| `GLOSKI_HOST` / `GLOSKI_PORT` | `127.0.0.1:8080` | Dashboard HTTP listener |
| `GLOSKI_DATA_DIR` | `~/.gloski/data` | DB, tokens, agent socket, known_hosts |
| `GLOSKI_MASTER_WEB_DIR` | `web/dist` | Built SPA |
| `GLOSKI_MASTER_JOIN_TOKEN` | generated | Agent enrollment token |
| `GLOSKI_MASTER_AGENT_BINARY` | next to master | Agent binary for deployments |
| `GLOSKI_MASTER_SSH_KEY_FILE` | (none) | Explicit SSH key |
| `GLOSKI_MASTER_AGENT_GRPC_HOST` | `0.0.0.0` | Remote agent gRPC bind |
| `GLOSKI_MASTER_AGENT_GRPC_PORT` | `9090` | Remote agent gRPC port |
| `GLOSKI_MASTER_AGENT_GRPC_URL` | auto | Override the agent-facing gRPC URL |
| `GLOSKI_TAILSCALE_SOCKET` | auto | tailscaled LocalAPI socket |
| `GLOSKI_API_KEY` | (none) | Optional dashboard user auth |

## Control-plane API (browser → master)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/tower/info` | Detection and fleet summary |
| `GET` | `/api/tailnet/peers` | Tailnet peers from LocalAPI |
| `POST` | `/api/devices/deploy` | SSH deploy an agent |
| `POST` | `/api/devices/{id}/start` / `/stop` | Lifecycle |
| `GET` | `/api/devices/{id}/api/jobs` | Master-owned jobs |
| `POST` | `/api/devices/{id}/api/jobs` | Run a task |
| `GET` | `/api/devices/{id}/api/jobs/{jobId}/logs` | Job logs |
| `GET` | `/api/devices/{id}/api/files` | File listing |
| `GET` | `/api/devices/{id}/api/files/read` | Read a file |
| `GET` | `/api/devices/{id}/api/terminal` | Terminal WebSocket |

## Logs and health

- Master log: wherever stdout of `gloski` is redirected. It records local agent
  start/stop, gRPC session state, health checks, and forced restarts.
- Local agent log: `<data_dir>/local-agent.log`.
- Remote agent log: `/tmp/gloski-agent.log` on the device.

The master checks the local agent's gRPC session every 5 seconds. If it is not
connected for 20 seconds, the master kills the child process and starts a new
one with backoff. Remote agents reconnect to the master on their own.

## Security

Keep the dashboard on the tailnet and use `tailscale serve` for HTTPS. The
agent gRPC port is h2c and should not be exposed off-tailnet. Host keys are
trusted on first use and stored in `<data_dir>/known_hosts`.
