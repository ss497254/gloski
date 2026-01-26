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

## Key Design Decisions

### 1. Decentralized Architecture

Each server runs its own Gloski server instance. There is no central server - the frontend connects directly to each server.

**Benefits:**

- No single point of failure
- Servers work independently
- No central database to maintain
- Easy to add/remove servers

**Trade-offs:**

- Workspace data (notes, bookmarks) stored in browser localStorage
- No cross-device sync (yet)

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

### 3. Stateless Server Design

Gloski servers are stateless - they don't store any user data:

- No database
- No user accounts
- No session storage
- Authentication via API key or JWT

This means:

- Easy deployment (single binary)
- Easy scaling (just add more servers)
- No data migration concerns

### 4. Authentication Model

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
- Backend: Stateless, scale by running more instances
- Data: Currently browser-local, could add optional sync
