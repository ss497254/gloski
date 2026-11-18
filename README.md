# Gloski

Lightweight Go-based server management tool with terminal access, file browser, process control, real-time metrics, and an optional control tower for a whole fleet of devices.

## Quick Start

### Prerequisites

- Go 1.24+
- Bun

### Development

```bash
# Clone the repository
git clone https://github.com/ss497254/gloski.git
cd gloski

# Start server (terminal 1)
make dev-server

# Start frontend (terminal 2)
cd web && bun install && bun run dev
```

Open http://localhost:4000:

1. Click "Add Server"
2. Enter server URL: `http://127.0.0.1:8080`
3. Enter API key: `your-api-key`

## Control Tower

One `gloski` master serves the dashboard and holds all state. Every device —
including the master itself — runs a stateless `gloski-agent`. Master and agents
communicate over one gRPC bidirectional stream (Unix socket locally, h2c over
Tailnet remotely); the browser only talks HTTP/WS to the master.

The Fleet page discovers peers through Tailscale's LocalAPI and deploys agents
over SSH to `/tmp`.

```bash
make build
GLOSKI_MODE=master GLOSKI_MASTER_WEB_DIR="$PWD/web/dist" bin/gloski
```

See [Control Tower](docs/tower.md) for details.

## Features

- **Multi-Server Support**: Connect to and manage multiple servers
- **Control Tower**: Enroll devices, monitor a live fleet, and proxy their APIs through one dashboard
- **btop-style Fleet View**: CPU, memory, network, disks, and processes for every device
- **Real-time Dashboard**: Live CPU, memory, disk, network metrics
- **Terminal Access**: Full PTY terminal via WebSocket
- **File Browser**: Navigate, view, edit, upload and download files
- **Job Management**: Start, stop, and inspect background jobs
- **Systemd Integration**: Manage systemd services

## Configuration

### Server Environment Variables

| Variable                | Default     | Description                         |
| ----------------------- | ----------- | ----------------------------------- |
| `GLOSKI_MODE`           | `standalone`| `standalone`, `master`, or `agent`  |
| `GLOSKI_HOST`           | `127.0.0.1` | Server bind address                 |
| `GLOSKI_PORT`           | `8080`      | Server port                         |
| `GLOSKI_DATA_DIR`       | `~/.gloski/data` | Database, downloads, and state |
| `GLOSKI_API_KEY`        | (none)      | API key for authentication          |
| `GLOSKI_LOG_LEVEL`      | `info`      | Log level: debug, info, warn, error |

Master and agent settings are documented in [docs/tower.md](docs/tower.md).

## Documentation

- [Architecture Overview](docs/architecture.md)
- [Control Tower](docs/tower.md)
- [Frontend Architecture](docs/frontend.md)
- [Backend Architecture](docs/backend.md)
- [SDK Documentation](docs/sdk.md)
