# Architecture Overview

Gloski is a control center for managing multiple Linux servers. This document describes the overall system architecture and how components interact.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              USER'S BROWSER                                 │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                     GLOSKI FRONTEND (React SPA)                       │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │  │
│  │  │  Dashboard  │  │    Files    │  │  Terminal   │  │    Tasks    │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘   │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │  │
│  │  │  Bookmarks  │  │    Notes    │  │  Snippets   │  │  Messages   │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘   │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │                    SERVERS STORE (Zustand)                       │ │  │
│  │  │  Server 1: { url, apiKey, status }                               │ │  │
│  │  │  Server 2: { url, apiKey, status }                               │ │  │
│  │  │  Server N: { url, apiKey, status }                               │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                    │                    │                    │
                    │ HTTPS/WSS          │ HTTPS/WSS         │ HTTPS/WSS
                    ▼                    ▼                    ▼
┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────┐
│   GLOSKI SERVER #1    │  │   GLOSKI SERVER #2    │  │   GLOSKI SERVER #N    │
│   (Your VPS/Server)   │  │   (Your VPS/Server)   │  │   (Your VPS/Server)   │
│                       │  │                       │  │                       │
│  ┌─────────────────┐  │  │  ┌─────────────────┐  │  │  ┌─────────────────┐  │
│  │   HTTP Server   │  │  │  │   HTTP Server   │  │  │  │   HTTP Server   │  │
│  │   (Go net/http) │  │  │  │   (Go net/http) │  │  │  │   (Go net/http) │  │
│  └────────┬────────┘  │  │  └────────┬────────┘  │  │  └────────┬────────┘  │
│           │           │  │           │           │  │           │           │
│  ┌────────┴────────┐  │  │  ┌────────┴────────┐  │  │  ┌────────┴────────┐  │
│  │    Services     │  │  │  │    Services     │  │  │  │    Services     │  │
│  │  ┌───────────┐  │  │  │  │  ┌───────────┐  │  │  │  │  ┌───────────┐  │  │
│  │  │   Files   │  │  │  │  │  │   Files   │  │  │  │  │  │   Files   │  │  │
│  │  │  System   │  │  │  │  │  │  System   │  │  │  │  │  │  System   │  │  │
│  │  │  Jobs     │  │  │  │  │  │  Jobs     │  │  │  │  │  │  Jobs     │  │  │
│  │  │ Terminal  │  │  │  │  │  │ Terminal  │  │  │  │  │  │ Terminal  │  │  │
│  │  └───────────┘  │  │  │  │  └───────────┘  │  │  │  │  └───────────┘  │  │
│  └─────────────────┘  │  │  └─────────────────┘  │  │  └─────────────────┘  │
│           │           │  │           │           │  │           │           │
│  ┌────────┴────────┐  │  │  ┌────────┴────────┐  │  │  ┌────────┴────────┐  │
│  │  Linux System   │  │  │  │  Linux System   │  │  │  │  Linux System   │  │
│  │  (filesystem,   │  │  │  │  (filesystem,   │  │  │  │  (filesystem,   │  │
│  │   processes,    │  │  │  │   processes,    │  │  │  │   processes,    │  │
│  │   systemd)      │  │  │  │   systemd)      │  │  │  │   systemd)      │  │
│  └─────────────────┘  │  │  └─────────────────┘  │  │  └─────────────────┘  │
└───────────────────────┘  └───────────────────────┘  └───────────────────────┘
```

The diagram above is the direct/standalone topology. In control tower mode the
browser talks only to a master, which serves the SPA and proxies to
`gloski-agent` processes on each device; see
[Control Tower Mode](#3-control-tower-mode) below.

## Key Design Decisions

### 1. Flexible Topology

Each server can run its own standalone Gloski instance, and the frontend can
connect directly to each server. For a managed fleet, one instance can instead
run as a **control tower** (`master`) that serves the SPA and proxies requests
to lightweight agents on each device. See [Control Tower](tower.md).

**Benefits:**

- Standalone servers have no single point of failure
- Servers work independently with no central database
- A fleet gets one dashboard, one registry, and one access path
- Easy to add/remove devices without browser-side credentials

**Trade-offs:**

- Workspace data (notes, bookmarks) is stored in browser localStorage
- No cross-device sync (yet)
- The master is a control-plane dependency for fleet views

### 2. Frontend-Managed Server Registry

The frontend maintains the list of servers in Zustand store (persisted to localStorage):

```typescript
// Stored in browser
interface Server {
  id: string
  name: string
  url: string // e.g., "https://server1.example.com"
  apiKey?: string // API key authentication
  apiPrefix?: string // API Prefix, Defaults to /api
  token?: string // JWT authentication
  status: 'online' | 'offline' | 'connecting' | 'unauthorized'
}
```

### 3. Control Tower Mode

```
Browser ──HTTPS/WSS──▶ gloski master ──HTTP──▶ gloski-agent (device A)
                         │      │      └─────▶ gloski-agent (device B)
                         │      └────────────▶ gloski-agent (device N)
                         └── SQLite device registry + SPA
```

- The master serves the built SPA from `web/dist`, so the dashboard is never
  hosted separately.
- Agents enroll with a join token and persist a per-device token locally.
- Heartbeats carry a cached stats snapshot; the fleet WebSocket fans changes
  out to browsers.
- `/api/devices/{id}/...` is a reverse proxy to the agent. The master injects
  the device token, so browser credentials never reach the agent.
- If the app detects a master (`GET /api/tower/info`), it adds fleet routes and
  virtual `tower-<id>` server entries that reuse the existing feature pages.
- Standalone mode remains the default and continues to work without a master.

### 4. Lightweight Server Design

Gloski servers use a local SQLite database for operational state (jobs, downloads, pinned folders) but don't store any user account data:

- No user accounts or session storage
- Authentication via API key or JWT
- SQLite database for job history, downloads, and pinned folders (auto-created)

This means:

- Easy deployment (single binary + auto-created SQLite file)
- No external database to configure
- No data migration concerns

### 5. Authentication Model

```
┌─────────────────────────────────────────────────────────────┐
│                    AUTHENTICATION FLOW                      │
└─────────────────────────────────────────────────────────────┘

Option A: API Key (Simple)
──────────────────────────
┌──────────┐                        ┌──────────────┐
│ Frontend │ ──── X-API-Key ──────▶ │ Gloski Server│
└──────────┘                        └──────────────┘

  - API key stored in frontend (localStorage)
  - Server validates against configured key
  - Good for: personal use, simple setups


Option B: JWT (Enterprise)
──────────────────────────
┌──────────┐      ┌────────────┐      ┌──────────────┐
│ Frontend │ ───▶ │ Auth Server│ ───▶ │    JWT       │
└──────────┘      │ (external) │      │   Token      │
                  └────────────┘      └──────┬───────┘
                                             │
┌──────────┐                                 │
│ Frontend │ ◀───────────────────────── ─────┘
└────┬─────┘
     │ Authorization: Bearer <token>
     ▼
┌──────────────┐
│ Gloski Server│  (verifies with public key)
└──────────────┘

  - Tokens signed by external auth service
  - Server only needs public key to verify
  - Good for: teams, SSO integration
```

## Data Flow

### System Stats Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    GLOSKI SERVER                            │
│                                                             │
│  ┌─────────────────┐    2s interval    ┌─────────────────┐  │
│  │ Stats Collector │ ────────────────▶ │   Ring Buffer   │  │
│  │  (goroutine)    │                   │  (150 samples)  │  │
│  └────────┬────────┘                   │  = 5 min history│  │
│           │                            └────────┬────────┘  │
│           │ reads                               │           │
│           ▼                                     │ provides  │
│  ┌─────────────────┐                            ▼           │
│  │  /proc/stat     │                   ┌─────────────────┐  │
│  │  /proc/meminfo  │                   │  System Service │  │
│  │  /proc/diskstats│                   └────────┬────────┘  │
│  │  /proc/net/dev  │                            │           │
│  └─────────────────┘                            │           │
│                                                 │           │
└─────────────────────────────────────────────────┼───────────┘
                                                  │
                            GET /api/system/stats │
                            GET /api/system/stats/history
                                                  │
┌─────────────────────────────────────────────────┼───────────┐
│                      FRONTEND                   │           │
│                                                 ▼           │
│  ┌─────────────────┐    polling     ┌─────────────────┐     │
│  │ Dashboard Widget│ ◀───(5s)────── │   API Client    │    │
│  │   (React)       │                └─────────────────┘     │
│  └─────────────────┘                                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Terminal WebSocket Flow

```
┌──────────────┐         WebSocket          ┌──────────────┐
│   Frontend   │ ◀═══════════════════════▶ │ Gloski Server│
│   (xterm.js) │                            │              │
└──────────────┘                            └──────┬───────┘
                                                   │
                                                   │ PTY
                                                   ▼
                                            ┌──────────────┐
                                            │    Shell     │
                                            │  (bash/zsh)  │
                                            └──────────────┘

Protocol:
- Binary frames: terminal I/O
- Control frames: [0x01, cols_hi, cols_lo, rows_hi, rows_lo] for resize
```

### File Operations Flow

```
┌──────────────┐                            ┌──────────────┐
│   Frontend   │                            │ Gloski Server│
└──────┬───────┘                            └──────┬───────┘
       │                                           │
       │ GET /api/files?path=/home                 │
       │ ─────────────────────────────────────────▶│
       │                                           │ os.ReadDir()
       │                  { entries: [...] }       │◀────────────
       │ ◀─────────────────────────────────────────│
       │                                           │
       │ GET /api/files/read?path=/etc/hosts       │
       │ ─────────────────────────────────────────▶│
       │                                           │ os.ReadFile()
       │                  { content: "..." }       │◀────────────
       │ ◀─────────────────────────────────────────│
       │                                           │
       │ POST /api/files/write                     │
       │ { path: "/tmp/file", content: "..." }     │
       │ ─────────────────────────────────────────▶│
       │                                           │ os.WriteFile()
       │                  { success: true }        │◀────────────
       │ ◀─────────────────────────────────────────│
       │                                           │
```

## Frontend Architecture

### Feature Module System

```
┌─────────────────────────────────────────────────────────────┐
│                    FEATURE REGISTRY                          │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ features: [                                              ││
│  │   { id: 'dashboard', path: '/', section: 'main' }       ││
│  │   { id: 'servers', path: '/servers', section: 'main' }  ││
│  │   { id: 'notes', path: '/notes', section: 'workspace' } ││
│  │   ...                                                    ││
│  │ ]                                                        ││
│  └─────────────────────────────────────────────────────────┘│
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    ROUTER                                ││
│  │  Generates routes from feature registry                  ││
│  └─────────────────────────────────────────────────────────┘│
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                   SIDEBAR                                ││
│  │  Generates navigation from feature registry              ││
│  └─────────────────────────────────────────────────────────┘│
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              COMMAND PALETTE                             ││
│  │  Generates commands from feature registry                ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### State Management

```
┌─────────────────────────────────────────────────────────────┐
│                    ZUSTAND STORES                            │
│                                                              │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │
│  │ ServersStore  │  │ SettingsStore │  │  NotesStore   │   │
│  │               │  │               │  │               │   │
│  │ - servers[]   │  │ - theme       │  │ - notes[]     │   │
│  │ - addServer() │  │ - sidebar     │  │ - addNote()   │   │
│  │ - updateServer│  │ - setTheme()  │  │ - updateNote()│   │
│  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘   │
│          │                  │                  │            │
│          └──────────────────┼──────────────────┘            │
│                             │                               │
│                             ▼                               │
│                    ┌─────────────────┐                      │
│                    │   localStorage  │                      │
│                    │   (persisted)   │                      │
│                    └─────────────────┘                      │
└─────────────────────────────────────────────────────────────┘
```

## Backend Architecture

### Request Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP REQUEST                              │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    MIDDLEWARE CHAIN                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Logging   │─▶│    CORS     │─▶│    Auth     │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                      ROUTER                                  │
│  GET  /api/system/stats    ──▶  SystemHandler.GetStats      │
│  GET  /api/files           ──▶  FilesHandler.List           │
│  POST /api/files/write     ──▶  FilesHandler.Write          │
│  GET  /api/terminal        ──▶  TerminalHandler.Handle      │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                     HANDLERS                                 │
│  - Parse request                                            │
│  - Validate input                                           │
│  - Call service                                             │
│  - Format response                                          │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                     SERVICES                                 │
│  - Business logic                                           │
│  - System interactions                                      │
│  - Error handling                                           │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                  LINUX SYSTEM                                │
│  - Filesystem (/proc, /sys, user files)                     │
│  - Processes (exec, pty)                                    │
│  - Systemd (systemctl, journalctl)                          │
└─────────────────────────────────────────────────────────────┘
```

### Service Dependencies

```
┌─────────────────────────────────────────────────────────────┐
│                      APP CONTAINER                           │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    Config                                ││
│  │  (loaded from env vars + config file)                    ││
│  └─────────────────────────────────────────────────────────┘│
│                           │                                  │
│           ┌───────────────┼───────────────┐                 │
│           ▼               ▼               ▼                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ AuthService │  │FilesService │  │TasksService │         │
│  │             │  │             │  │             │         │
│  │ - apiKey    │  │ - allowed   │  │ - tasks map │         │
│  │ - pubKey    │  │   Paths     │  │ - systemd   │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                  SystemService                           ││
│  │  ┌─────────────┐         ┌─────────────┐                ││
│  │  │  Collector  │────────▶│   Store     │                ││
│  │  │ (goroutine) │         │(ring buffer)│                ││
│  │  └─────────────┘         └─────────────┘                ││
│  └─────────────────────────────────────────────────────────┘│
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  (optional)              │
│  │PackagesServ │  │ CronService │                           │
│  │(apt/dnf/etc)│  │ (crontab)   │                           │
│  └─────────────┘  └─────────────┘                           │
└─────────────────────────────────────────────────────────────┘
```

## Security Model

### Network Security

```
┌─────────────────────────────────────────────────────────────┐
│                   RECOMMENDED SETUP                          │
│                                                              │
│  Internet                                                    │
│      │                                                       │
│      ▼                                                       │
│  ┌─────────────────┐                                        │
│  │  Reverse Proxy  │  (Caddy/nginx)                         │
│  │  - HTTPS/TLS    │                                        │
│  │  - Rate limiting│                                        │
│  └────────┬────────┘                                        │
│           │ localhost:8080                                   │
│           ▼                                                  │
│  ┌─────────────────┐                                        │
│  │  Gloski Server  │                                        │
│  │  - Binds to     │                                        │
│  │    127.0.0.1    │                                        │
│  └─────────────────┘                                        │
│                                                              │
│  Firewall: Only allow 443 (HTTPS) from internet             │
└─────────────────────────────────────────────────────────────┘
```

### Authentication Security

```
┌─────────────────────────────────────────────────────────────┐
│                  SECURITY LAYERS                             │
│                                                              │
│  1. Transport: HTTPS (TLS encryption)                       │
│                                                              │
│  2. Authentication:                                          │
│     - API Key: constant-time comparison                     │
│     - JWT: RS256 signature verification                     │
│                                                              │
│  3. Authorization:                                           │
│     - Allowed paths: restrict filesystem access             │
│     - CORS: restrict frontend origins                       │
│                                                              │
│  4. Input Validation:                                        │
│     - Path traversal prevention                             │
│     - Request size limits                                   │
└─────────────────────────────────────────────────────────────┘
```

## Deployment Patterns

### Control Tower

```
┌────────────────────┐        ┌──────────────────────────┐
│      Browser       │──WSS──▶│  gloski master (loopback) │
└────────────────────┘        │  + tailscale serve HTTPS  │
                              │  devices registry (SQLite)│
                              └───────┬─────────┬────────┘
                                      │         │
                             HTTP+token│         │HTTP+token
                                      ▼         ▼
                              ┌────────────┐ ┌────────────┐
                              │gloski-agent│ │gloski-agent│
                              │  device A  │ │  device B  │
                              └────────────┘ └────────────┘
```

The master and agents are separate binaries built from the same repository.
`tailscale serve` supplies HTTPS and tailnet access control while the master
stays bound to loopback. See [Control Tower](tower.md) for setup.

### Single Server

```
┌─────────────────────────────────────────────────────────────┐
│                     YOUR VPS                                 │
│                                                              │
│  ┌─────────────────┐     ┌─────────────────┐               │
│  │  Gloski Server  │     │   Your Apps     │               │
│  │  (port 8080)    │     │   (various)     │               │
│  └────────┬────────┘     └─────────────────┘               │
│           │                                                  │
│  ┌────────┴────────────────────────────────┐               │
│  │              Caddy/nginx                 │               │
│  │  gloski.example.com ──▶ localhost:8080  │               │
│  └─────────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────┘
```

### Multiple Servers

```
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│   Server 1    │  │   Server 2    │  │   Server 3    │
│   (US-East)   │  │   (EU-West)   │  │   (Asia)      │
│               │  │               │  │               │
│ ┌───────────┐ │  │ ┌───────────┐ │  │ ┌───────────┐ │
│ │  Gloski   │ │  │ │  Gloski   │ │  │ │  Gloski   │ │
│ │  Server   │ │  │ │  Server   │ │  │ │  Server   │ │
│ └───────────┘ │  │ └───────────┘ │  │ └───────────┘ │
└───────┬───────┘  └───────┬───────┘  └───────┬───────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │   Frontend (Browser)    │
              │   - Connects to all     │
              │   - Manages locally     │
              └─────────────────────────┘
```

### With Central Auth (Enterprise)

```
┌─────────────────────────────────────────────────────────────┐
│                    AUTH SERVER                               │
│  (Keycloak, Auth0, custom)                                  │
│  - Issues JWT tokens                                        │
│  - Manages users                                            │
└─────────────────────────────────────────────────────────────┘
        │ JWT tokens
        ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│   Server 1    │  │   Server 2    │  │   Server 3    │
│ ┌───────────┐ │  │ ┌───────────┐ │  │ ┌───────────┐ │
│ │  Gloski   │ │  │ │  Gloski   │ │  │ │  Gloski   │ │
│ │ (pub key) │ │  │ │ (pub key) │ │  │ │ (pub key) │ │
│ └───────────┘ │  │ └───────────┘ │  │ └───────────┘ │
└───────────────┘  └───────────────┘  └───────────────┘

All servers share the same public key to verify tokens.
```

## Future Considerations

### Potential Enhancements

1. **Sync Service**: Optional central server for syncing workspace data
2. **Server Groups**: Organize servers into logical groups
3. **Role-Based Access**: Different permission levels per server
4. **Audit Logging**: Track all actions for compliance
5. **Metrics Export**: Prometheus/Grafana integration
6. **Mobile App**: Native mobile client

### Scalability Notes

- Frontend: Static files, infinitely scalable via CDN
- Backend: Lightweight single binary, SQLite is local and auto-created
- Data: Currently browser-local, could add optional sync
