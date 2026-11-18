# Gloski

A Tailnet-first control tower for Linux devices. One `gloski` master serves the
dashboard and coordinates stateless `gloski-agent` workers on every device,
including itself.

## Quick start

Prerequisites: Go 1.24+, Bun.

```bash
make build
GLOSKI_MASTER_WEB_DIR="$PWD/web/dist" bin/gloski
```

Open the dashboard, then use **Deploy agent** to copy `gloski-agent` to a
Tailnet device over SSH and start it from `/tmp`. The master supplies the
agent's configuration and gRPC endpoint; no manual agent setup is needed.

See [Control Tower](docs/tower.md) for setup and configuration.

## What it does

- **Dashboard** — device summary, issues, and recent tasks
- **Device overview** — CPU, memory, network, disks, and top processes
- **Device tools** — files, terminal, and ad-hoc jobs on any device
- **Lifecycle** — deploy, stop, and restart agents from the dashboard
- **Stateless agents** — no agent-side database or credential files; the master
  holds all state
- **Fast transport** — one gRPC stream per agent (Unix socket locally, h2c over
  Tailnet remotely)

The browser talks only to the master over HTTP/WS. Master and agents do not use
HTTP between them.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_MODE` | `master` | `master` or `agent` |
| `GLOSKI_HOST` / `GLOSKI_PORT` | `127.0.0.1:8080` | Dashboard listener |
| `GLOSKI_DATA_DIR` | `~/.gloski/data` | Database, tokens, socket, known_hosts |
| `GLOSKI_MASTER_WEB_DIR` | `web/dist` | Built SPA |
| `GLOSKI_MASTER_AGENT_BINARY` | next to master | Agent binary for deployments |
| `GLOSKI_MASTER_AGENT_GRPC_PORT` | `9090` | Remote agent gRPC port |
| `GLOSKI_API_KEY` | (none) | Optional dashboard API auth |

All settings are documented in [docs/tower.md](docs/tower.md).

## Documentation

- [Architecture Overview](docs/architecture.md)
- [Control Tower](docs/tower.md)
- [Frontend](docs/frontend.md)
- [Backend](docs/backend.md)
- [SDK](docs/sdk.md)
