# Backend Architecture

The Go side has two binaries: `gloski` (the master/control tower) and
`gloski-agent` (the device-side worker). They communicate only over gRPC.

Terms in this document are defined in [glossary.md](glossary.md). Routes and
payloads are in [http-api.md](http-api.md); tables and statuses are in
[data-model.md](data-model.md).

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
    │   ├── handlers.go       # Fleet, device info, fleet WebSocket
    │   ├── dashboard.go      # Summary and issues
    │   ├── deploy.go         # SSH deployment and lifecycle endpoints
    │   ├── device_api.go     # Shared device helpers, stats, processes
    │   ├── device_jobs.go    # Job endpoints
    │   ├── device_files.go   # File endpoints
    │   ├── device_transfer.go# Downloads and uploads from the browser
    │   ├── file_transfer.go  # Transfer registry and agent HTTP endpoints
    │   ├── device_terminal.go# Terminal WebSocket bridge
    │   ├── agent_http.go     # Plain HTTP health on the agent port
    │   ├── agent_transport.go# gRPC listeners and endpoint selection
    │   ├── local_agent.go    # Supervised agent on the master host
    │   ├── lifecycle.go      # Reinstall disconnected/outdated agents
    │   ├── fleet_loop.go     # Periodic fleet status broadcast
    │   └── spa.go            # SPA static files and fallback
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
    │   ├── agent_jobs.go     # Job execution and streamed output
    │   ├── agent_files.go    # Small file operations over the stream
    │   ├── agent_transfer.go # File contents over HTTP
    │   ├── agent_terminal.go # PTY sessions
    │   └── agent_dial.go     # Dial master over Unix socket or h2c
    ├── config/               # Config types, defaults, env overrides
    ├── database/             # SQLite migrations
    ├── checks/               # Check result model and its store
    ├── httpapi/              # JSON responses, auth and logging middleware
    ├── agentdist/            # Agent builds and install scripts
    ├── auth/                 # Optional API key and JWT checks
    └── files/, system/       # Shared filesystem and stats helpers
```

## Main types

| Type | Meaning |
|---|---|
| `controltower.Tower` | Owns the master DB, HTTP API, gRPC listeners, and local agent process |
| `fleet.Registry` | Source of truth for enrolled devices, heartbeats, and online status |
| `agents.Service` | Master-side stream registry; sends commands to a specific device |
| `agents.connection` | One device's gRPC stream with serialized sends and pending request channels |
| `provisioning.Installer` | Runs the install and uninstall scripts on a device over SSH |
| `agentdist.Dist` | Agent builds per architecture and the install and uninstall scripts the master serves |
| `controltower.lifecycle` | Master loop that reinstalls any deployed device whose agent is disconnected or outdated |
| `provisioning/sshtest` | SSH server for tests that runs commands locally, standing in for a device |
| `worker.Runtime` | Device-side executor: handles jobs, files, terminal, and heartbeats |
| `tailnet.Client` | Reads tailnet peers from tailscaled's LocalAPI socket |
| `checks.Result` | One check run: status, summary, ordered details, timing and staleness |
| `checks.Store` | Saves results and reads the latest result, per-device results and short history |

`controltower/agents` and `worker` are the two ends of the same gRPC stream:
the former runs in the master, the latter on each device.

## Connection lifecycle

1. Master starts gRPC listeners: a Unix socket for its own agent and TCP h2c for remote agents.
2. Master starts `gloski-agent` as a supervised child process pointed at the Unix socket.
3. Remote agents are installed by the script from `agentdist` and pointed at the TCP listener, which can also carry plain HTTP.
4. The agent opens `Agent.Connect` and sends `RegisterRequest` with the join token.
5. Master registers the device, issues a device token, and stores the stream under that device ID.
6. Agent sends heartbeats with cached stats; master stores them and broadcasts fleet updates.
7. If the stream drops, the agent reconnects with backoff, reports the jobs it still
   holds, and resends the output the master is missing. If the master's local agent has no
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
- **Native checks** — configured `CheckRequest` sent from the heartbeat hook →
  agent runs a TCP/HTTP/DNS/TLS probe → `CheckResult` on the same stream →
  bounded aggregation → `check_results` → fleet matrix. Probes stop after three
  minutes without a dashboard or device-stats request.
- **Jobs** — browser POST → master stores job → `RunJob` command → agent executes →
  output chunks stream back → master persists logs in `agent_job_logs`.
- **Files** — master translates `/api/devices/{id}/api/files/*` into gRPC `FileRequest`
  for listing, stat, rename, delete and small text reads/writes. Download and upload
  contents use a `FileTransfer` message instead: the master sends a one-time id on the
  stream and the agent makes one plain HTTP request to `/agent/transfer/{id}` on the
  agent port, posting a download or fetching an upload. The bytes never share the
  terminal and job lanes, and an upload replaces its target only when the whole body
  arrived. Only images, audio, video and PDF are served inline; everything else is a
  download.
- **Terminal** — text frames are forwarded as `TerminalInput` and the 5-byte binary resize
  frame as `TerminalResize`. PTY output is streamed back as `TerminalOutput` into a bounded
  buffer per session, so a slow browser cannot stall the device's stream.

## State

All durable state is in the master's SQLite database:

| Table | Contents |
|---|---|
| `devices` | Device identity, token, URL label, cached stats, master flag, deployment link |
| `deployments` | SSH deployment records and status |
| `agent_jobs` / `agent_job_logs` | Master-owned jobs and streamed output |
| `schema_migrations` | Applied migrations |

Agents hold no durable state. Credentials and running jobs live only in memory.

## Configuration

Configuration comes from an optional JSON file, then environment overrides
(`internal/config/env.go`). Key settings:

| Variable | Default | Purpose |
|---|---|---|
| `GLOSKI_MODE` | `master` | The `gloski` binary only runs `master`; use `gloski-agent` for the device side |
| `GLOSKI_HOST` / `GLOSKI_PORT` | `127.0.0.1:8080` | Dashboard listener |
| `GLOSKI_MASTER_WEB_DIR` | `web/dist` | SPA directory |
| `GLOSKI_MASTER_AGENT_GRPC_HOST` / `_PORT` | Tailscale IPs, `9090` | Remote agent listener |
| `GLOSKI_ALLOWED_ORIGINS` | none | Extra browser origins allowed to call the API |
| `GLOSKI_MASTER_AGENT_BINARY` | next to master | Used only when the master was built without embedded agents |
| `GLOSKI_MASTER_LOCAL_AGENT` | `true` | Run the master's own agent |
| `GLOSKI_TAILSCALE_SOCKET` | auto | tailscaled LocalAPI socket |
| `GLOSKI_API_KEY` | none | Optional browser API auth |

See [Control Tower](tower.md) for the full list.

## Build and test

```bash
cd server
go test ./...
go vet ./...
```

Build with `make build-server` from the repository root. It first cross-builds
the agents into `internal/agentdist/builds/` and then embeds them in the
master. A plain `go build ./cmd/gloski` works too, but that master can only
install agents if a `gloski-agent` binary sits next to it.

Both binaries stay under 30 MB. Generated protobuf code is checked in, so
`protoc` is not required to build. After editing `agent.proto`, run `make proto`
(protoc 3.21 with `protoc-gen-go` v1.36.6 and `protoc-gen-go-grpc` v1.5.1 on
`PATH`).
