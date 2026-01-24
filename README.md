# Gloski

A control center for managing multiple Linux servers with terminal access, file management, system monitoring, and more.

## Features

### Server Management
- **Multi-Server Support**: Connect to and manage multiple servers from one interface
- **Real-time Dashboard**: Live CPU, memory, disk, network metrics with history
- **Terminal Access**: Full PTY terminal via WebSocket
- **File Browser**: Navigate, view, edit, create, delete, upload and download files
- **Search**: Global file search with content search support
- **Task Management**: Start/stop background tasks, view logs
- **Systemd Integration**: Manage systemd services

### Workspace Features
- **Bookmarks**: Save and organize links with folders and tags
- **Notes**: Markdown notes with auto-save
- **Snippets**: Code snippets with syntax highlighting
- **Messages**: Notification inbox
- **Activity**: Action timeline across all features
- **Settings**: Appearance, shortcuts, data management

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                 FRONTEND (Control Center)                   │
│          React App - deployed anywhere (local/cloud)        │
└──────────────┬─────────────────────────────────────────────┬┘
               │ HTTPS/WSS                                   │
               ▼                                             ▼
┌──────────────────────────┐          ┌─────────────────────────────┐
│   Gloski Server (VPS #1) │          │   Gloski Server (VPS #2)    │
│   API Key Auth           │          │   API Key Auth              │
└──────────────────────────┘          └─────────────────────────────┘
```

## Tech Stack

**Frontend:**
- React 19 + TypeScript
- Vite (build tool)
- TailwindCSS v4
- shadcn/ui components
- Zustand (state management)
- xterm.js (terminal)

**Server (Go):**
- Go 1.22+ (uses new router patterns)
- Minimal dependencies
- API key + JWT (RS256) authentication
- WebSocket for terminal
- Background stats collector with time-series history

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+

### Development

```bash
# Clone the repository
git clone https://github.com/ss497254/gloski.git
cd gloski

# Start server (terminal 1)
cd server && GLOSKI_API_KEY=your-api-key go run ./cmd/gloski

# Start frontend (terminal 2)
cd web && npm install && npm run dev
```

Open http://localhost:5173:
1. Click "Add Server"
2. Enter server URL: `http://127.0.0.1:8080`
3. Enter API key: `your-api-key`

## Configuration

### Server Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_HOST` | `127.0.0.1` | Server bind address |
| `GLOSKI_PORT` | `8080` | Server port |
| `GLOSKI_API_KEY` | (none) | API key for authentication |
| `GLOSKI_JWT_PUBLIC_KEY` | (none) | PEM-encoded RSA public key for JWT verification |
| `GLOSKI_JWT_PUBLIC_KEY_FILE` | (none) | Path to PEM file with RSA public key |
| `GLOSKI_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `GLOSKI_SHELL` | `$SHELL` or `/bin/bash` | Shell for terminal |

> **Note:** At least one authentication method (`GLOSKI_API_KEY` or `GLOSKI_JWT_PUBLIC_KEY`) is required.

### Server Config File

Create `~/.config/gloski/config.json`:

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "api_key": "your-secure-api-key",
  "jwt_public_key_file": "/path/to/public-key.pem",
  "allowed_origins": ["https://your-frontend.example.com"],
  "allowed_paths": ["/home/user/projects"],
  "log_level": "info"
}
```

## Project Structure

```
gloski/
├── server/                         # Go server
│   ├── cmd/gloski/main.go          # Entry point
│   └── internal/
│       ├── api/                    # HTTP server setup
│       │   ├── handlers/           # HTTP handlers
│       │   ├── middleware/         # Auth, CORS, logging
│       │   └── routes/             # Route definitions
│       ├── app/                    # Application container & lifecycle
│       ├── auth/                   # API key + JWT authentication
│       ├── config/                 # Configuration loading
│       ├── cron/                   # Cron job management
│       ├── files/                  # File operations service
│       ├── logger/                 # Structured logging
│       ├── packages/               # Package manager integration
│       ├── system/                 # System stats with background collector
│       │   ├── collector.go        # Background stats collection
│       │   ├── store.go            # Time-series ring buffer
│       │   ├── service.go          # API-facing service
│       │   └── types.go            # Stats data structures
│       ├── tasks/                  # Process & systemd management
│       └── terminal/               # PTY terminal WebSocket
│
├── web/                            # React frontend
│   └── src/
│       ├── app/                    # Application configuration
│       │   └── feature-registry.ts # Central feature registry
│       │
│       ├── features/               # Feature modules (isolated)
│       │   ├── dashboard/          # Main dashboard
│       │   │   └── components/     # Dashboard widgets
│       │   ├── servers/            # Server management
│       │   ├── files/              # File browser
│       │   ├── terminal/           # Terminal emulator
│       │   ├── tasks/              # Tasks/processes
│       │   ├── search/             # File search
│       │   ├── bookmarks/          # Bookmark management
│       │   │   ├── components/     # BookmarkCard, BookmarkDialog
│       │   │   ├── hooks/          # useBookmarksPage
│       │   │   ├── pages/          # BookmarksPage
│       │   │   └── stores/         # Zustand store
│       │   ├── notes/              # Markdown notes
│       │   │   ├── components/     # NoteListItem, NoteEditor
│       │   │   ├── hooks/          # useNotesPage
│       │   │   ├── pages/          # NotesPage
│       │   │   └── stores/         # Zustand store
│       │   ├── snippets/           # Code snippets
│       │   │   ├── components/     # SnippetCard, SnippetDialog, LanguageFilter
│       │   │   ├── pages/          # SnippetsPage
│       │   │   └── stores/         # Zustand store
│       │   ├── messages/           # Notifications
│       │   ├── activity/           # Activity timeline
│       │   └── settings/           # Settings page
│       │
│       ├── shared/                 # Shared across features
│       │   ├── components/         # Reusable UI components
│       │   │   ├── SearchInput     # Search input with icon
│       │   │   ├── FilterSidebar   # Sidebar filter navigation
│       │   │   ├── EmptyState      # Empty state placeholder
│       │   │   ├── StatCard        # Stats display card
│       │   │   └── ProgressBar     # Progress indicators
│       │   ├── hooks/              # Reusable hooks
│       │   │   ├── useFilter       # Generic filter/search/sort
│       │   │   ├── useDialog       # Dialog state management
│       │   │   ├── useSelection    # Master-detail selection
│       │   │   ├── useAsync        # Async state handling
│       │   │   └── useInterval     # Polling intervals
│       │   ├── lib/                # Utilities
│       │   │   ├── utils.ts        # formatBytes, generateId, etc.
│       │   │   └── types.ts        # Shared TypeScript types
│       │   └── services/           # API client
│       │
│       ├── layouts/                # Layout components
│       │   ├── AppLayout.tsx       # Main app shell
│       │   ├── PageLayout.tsx      # Page wrapper
│       │   └── Sidebar/            # Navigation sidebar
│       │
│       └── ui/                     # Pure UI components (shadcn)
│
└── README.md
```

## URL Structure

```
/                           → Dashboard (all servers grid)
/servers                    → Server list
/servers/:serverId          → Server detail (full stats)
/servers/:serverId/files    → File browser
/servers/:serverId/terminal → Terminal
/servers/:serverId/tasks    → Tasks + Systemd
/servers/:serverId/search   → Search

/bookmarks                  → Bookmark management
/notes                      → Notes editor
/snippets                   → Code snippets
/messages                   → Notification inbox
/activity                   → Activity timeline
/settings                   → Settings
```

## API Reference

### Authentication

Requests authenticate via API Key or JWT token:

**API Key:**
- Header: `X-API-Key: your-api-key`
- Query param: `?api_key=your-api-key`

**JWT Token (RS256, asymmetric):**
- Header: `Authorization: Bearer <token>`
- Query param: `?token=<jwt-token>`

The server verifies JWT tokens using a configured RSA public key. Tokens are signed externally (e.g., by a central auth service) and only verified by Gloski servers.

```
GET  /api/auth/status       { "authenticated": true }
```

### System Stats

```
GET /api/system/stats
Response: {
  "cpu": { "usage_percent": 45.2, "cores": 8, ... },
  "memory": { "total": 16000000000, "used_percent": 50.0, ... },
  "disks": [{ "mount_point": "/", "used_percent": 50.0, ... }],
  "network": { "total_rx": 1000000, "total_tx": 5000000, ... },
  "load_avg": { "load1": 0.5, "load5": 0.4, "load15": 0.3 },
  "uptime": 86400,
  "hostname": "server"
}

GET /api/system/stats/history?duration=5m
Response: {
  "samples": [{ "stats": {...}, "timestamp": "2024-01-20T10:00:00Z" }, ...],
  "count": 150,
  "duration": "5m0s"
}

GET /api/system/info
GET /api/system/processes?limit=100
```

### Files

```
GET    /api/files?path=/path/to/dir
GET    /api/files/read?path=/path/to/file
POST   /api/files/write        { "path": "string", "content": "string" }
POST   /api/files/mkdir        { "path": "string" }
DELETE /api/files?path=/path/to/file
POST   /api/files/upload       (multipart form)
GET    /api/files/download?path=/path/to/file
```

### Search

```
GET /api/search?path=/start/path&q=query&content=true&limit=100
```

### Terminal

```
WebSocket /api/terminal?api_key=<key>&cwd=/optional/path

Messages:
- Binary data: terminal input/output
- Resize: [0x01, cols_high, cols_low, rows_high, rows_low]
```

### Tasks

```
GET    /api/tasks
POST   /api/tasks              { "command": "string", "cwd": "string" }
GET    /api/tasks/{id}
GET    /api/tasks/{id}/logs
DELETE /api/tasks/{id}
```

### Systemd

```
GET  /api/systemd?user=true
POST /api/systemd/{unit}/{action}?user=true   (action: start, stop, restart, enable, disable)
GET  /api/systemd/{unit}/logs?user=true&lines=100
```

## Frontend Architecture

### Feature Module Pattern

Each feature is self-contained:
```
features/{name}/
├── components/     # Pure presentational components
├── hooks/          # Feature-specific logic hooks
├── pages/          # Page containers
└── stores/         # Zustand state management
```

### Shared Hooks

| Hook | Purpose |
|------|---------|
| `useFilter` | Generic filtering, searching, and sorting |
| `useDialog` | Dialog open/close state with editing item |
| `useSelection` | Master-detail selection pattern |
| `useAsync` | Async operation state (loading, error, data) |
| `useInterval` | Polling with automatic cleanup |

### Shared Components

| Component | Purpose |
|-----------|---------|
| `SearchInput` | Search input with icon and clear button |
| `FilterSidebar` | Sidebar navigation with filters |
| `EmptyState` | Empty state placeholder with icon |
| `StatCard` | Stats display card |
| `ProgressBar` | Progress indicators |

## Deployment

### Server (Your VPS)

1. Build the binary:
```bash
cd server && go build -o gloski ./cmd/gloski
```

2. Create systemd service:
```ini
# /etc/systemd/system/gloski.service
[Unit]
Description=Gloski Server
After=network.target

[Service]
Type=simple
User=youruser
Environment=GLOSKI_HOST=0.0.0.0
Environment=GLOSKI_PORT=8080
Environment=GLOSKI_API_KEY=your-secure-api-key
ExecStart=/usr/local/bin/gloski
Restart=always

[Install]
WantedBy=multi-user.target
```

3. Start:
```bash
sudo systemctl enable --now gloski
```

4. Configure reverse proxy (Caddy):
```
gloski.yourdomain.com {
    reverse_proxy localhost:8080
}
```

### Frontend

Deploy to Netlify, Vercel, or any static host:
- Build command: `npm run build`
- Publish directory: `dist`

## Security

1. **Always use HTTPS** in production
2. **Use strong API keys** - treat them like passwords
3. **Restrict allowed paths** in config to limit filesystem access
4. **Configure CORS** to only allow your frontend domain
5. **Firewall** - don't expose port 8080 directly, use reverse proxy

## Building

```bash
# Build server
cd server && go build -o gloski ./cmd/gloski

# Build frontend
cd web && npm run build
```

## License

MIT
