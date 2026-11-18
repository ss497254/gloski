# Backend Architecture

The Go side has two binaries: `gloski` (the master/control tower) and
`gloski-agent` (the device-side worker). They communicate only over gRPC.

## Tech stack

| Technology | Purpose |
|---|---|
| Go 1.24+ | Language |
| `net/http` | Browser-facing API and SPA |
| `gorilla/websocket` | Browser terminal and fleet streams |
| gRPC + protobuf | Master ↔ agent communication |
| `golang.org/x/crypto/ssh` | In-process SSH for agent installation |
| `modernc.org/sqlite` | Master database |
| `creack/pty` | PTY for terminal sessions |

## Directory layout

```
server/
├── cmd/
│   ├── gloski/               # Master entry point
│   └── gloski-agent/         # Device agent entry point
├── proto/agentv1/            # Generated gRPC types (checked in)
└── internal/
    ├── controltower/         # Master: dashboard, browser API, listeners
    │   ├── agents/           # Master side of the agent gRPC stream
    │   ├── server.go         # Tower type, construction, shutdown
    │   ├── routes.go         # Browser routes and auth wrapper
    │   ├── spa.go            # SPA static files and fallback
    │   ├── local_agent.go    # Supervised agent on the master host
    │   ├── agent_transport.go# gRPC listeners and endpoint selection
    │   ├── fleet_loop.go     # Periodic fleet status broadcast
    │   ├── deploy.go         # SSH deployment/lifecycle endpoints
    │   ├── dashboard.go      # Summary and issues
    │   ├── device_api.go     # Shared device API helpers
    │   ├── device_jobs.go    # Job endpoints
    │   ├── device_files.go   # File endpoints
    │   └── device_terminal.go# Terminal WebSocket bridge
    ├── fleet/                # Device registry, deployments, event hub
    │   ├── service.go        # Registry: enrollment, heartbeats, status
    │   ├── store.go          # SQLite persistence for devices
    │   ├── deployment.go     # Deployment records and store
    │   ├── hub.go            # EventHub for fleet WebSockets
    │   └── types.go          # Device/deployment types
    ├── provisioning/         # Installs and controls remote agents
    │   ├── installer.go      # Installer: copy binary, start, stop
    │   ├── ssh.go            # Auth, host-key trust, SSH execution
    │   └── scripts.go        # Remote shell scripts
    ├── tailnet/              # Tailscaled LocalAPI discovery
    │   └── client.go
    ├── worker/               # Device agent implementation
    │   ├── run.go            # Runtime lifecycle and command dispatch
    │   ├── agent_jobs.go     # Job execution and file operations
    │   ├── agent_terminal.go # PTY sessions
    │   └── agent_dial.go     # Dial master over Unix socket or h2c
    ├── config/               # Config types, defaults, env overrides
    ├── database/             # SQLite migrations
    ├── httpapi/              # JSON responses, auth and logging middleware
    └── files/, system/       # Shared filesystem and stats helpers
```

## Main types

| Type | Meaning |
|---|---|
| `controltower.Tower` | Owns the master DB, HTTP API, gRPC listeners, and local agent process |
| `fleet.Registry` | Source of truth for enrolled devices, heartbeats, and online status |
| `agents.Service` | Master-side stream registry; sends commands to a specific device |
| `agents.connection` | One device's gRPC stream with serialized sends and pending request channels |
| `provisioning.Installer` | Copies `gloski-agent` to `/tmp` over SSH and controls it by PID |
| `worker.Runtime` | Device-side executor: handles jobs, files, terminal, and heartbeats |
| `tailnet.Client` | Reads tailnet peers from tailscaled's LocalAPI socket |

`controltower/agents` and `worker` are the two ends of the same gRPC stream:
the former runs in the master, the latter on each device.

## Connection lifecycle

1. Master starts gRPC listeners: a Unix socket for its own agent and TCP h2c for remote agents.
2. Master starts `gloski-agent` as a supervised child process pointed at the Unix socket.
3. Remote agents are installed to `/tmp` by `provisioning.Installer` and pointed at the TCP listener.
4. The agent opens `Agent.Connect` and sends `RegisterRequest` with the join token.
5. Master registers the device, issues a device token, and stores the stream under that device ID.
6. Agent sends heartbeats with cached stats; master stores them and broadcasts fleet updates.
7. If the stream drops, the agent reconnects with backoff. If the master's local agent has no
   session for 20 seconds, master kills and restarts the child process.

## Request flows

**Browser → device**
```
Browser ──HTTP/WS──▶ Tower route ──gRPC command──▶ agents.connection
                                                   │
                                                   ▼
                                             worker.Runtime
```

- **Stats** — agent collector → heartbeat → master DB → dashboard and `/api/fleet/ws`.
- **Jobs** — browser POST → master stores job → `RunJob` command → agent executes →
  output chunks stream back → master persists logs in `agent_job_logs`.
- **Files** — master translates `/api/devices/{id}/api/files/*` into gRPC `FileRequest`.
- **Terminal** — browser WebSocket frames are forwarded as `TerminalInput`; resize frames
  are decoded by the agent. PTY output is streamed back as `TerminalOutput`.

## State

All durable state is in the master's SQLite database:

| Table | Contents |
|---|---|
| `devices` | Device identity, token, URL label, cached stats, master flag, deployment link |
| `deployments` | SSH deployment records and status |
| `agent_jobs` / `agent_job_logs` | Master-owned jobs and streamed output |
| `schema_migrations` | Applied migrations |

Agents hold no durable state. Credentials and in-flight job maps live only in memory.

## Configuration

Configuration comes from an optional JSON file, then environment overrides
(`internal/config/env.go`). Key settings:

| Variable | Default | Purpose |
|---|---|---|
| `GLOSKI_MODE` | `master` | `master` or `agent` |
| `GLOSKI_HOST` / `GLOSKI_PORT` | `127.0.0.1:8080` | Dashboard listener |
| `GLOSKI_MASTER_WEB_DIR` | `web/dist` | SPA directory |
| `GLOSKI_MASTER_AGENT_GRPC_HOST` / `_PORT` | `0.0.0.0:9090` | Remote agent listener |
| `GLOSKI_MASTER_AGENT_BINARY` | next to master | Binary used for deployments |
| `GLOSKI_MASTER_LOCAL_AGENT` | `true` | Run the master's own agent |
| `GLOSKI_TAILSCALE_SOCKET` | auto | tailscaled LocalAPI socket |
| `GLOSKI_API_KEY` | none | Optional browser API auth |

See [Control Tower](tower.md) for the full list.

## Build and test

```bash
cd server
go test ./...
go vet ./...
CGO_ENABLED=0 go build -o ../bin/gloski ./cmd/gloski
CGO_ENABLED=0 go build -o ../bin/gloski-agent ./cmd/gloski-agent
```

Both binaries stay under 30 MB. Generated protobuf code is checked in, so
`protoc` is not required to build.
