# Gloski

A control center for managing multiple Linux servers with terminal access, file management, and system monitoring.

## Quick Start

### Prerequisites

- Go 1.22+
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
3. Enter API key: `1234`

## Features

- **Multi-Server Support**: Connect to and manage multiple servers
- **Real-time Dashboard**: Live CPU, memory, disk, network metrics
- **Terminal Access**: Full PTY terminal via WebSocket
- **File Browser**: Navigate, view, edit, upload and download files
- **Task Management**: Start/stop background tasks, view logs
- **Systemd Integration**: Manage systemd services

## Configuration

### Server Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_HOST` | `127.0.0.1` | Server bind address |
| `GLOSKI_PORT` | `8080` | Server port |
| `GLOSKI_API_KEY` | (none) | API key for authentication |
| `GLOSKI_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |

## Documentation

- [Architecture Overview](docs/architecture.md)
- [Frontend Architecture](docs/frontend.md)
- [Backend Architecture](docs/backend.md)
- [SDK Documentation](docs/sdk.md)

## License

MIT
