# Backend Architecture

The Gloski server is a Go application that provides APIs for system management, file operations, terminal access, and more.

## Tech Stack

| Technology | Purpose |
|------------|---------|
| Go 1.22+ | Language (uses new router patterns) |
| net/http | HTTP server |
| gorilla/websocket | WebSocket for terminal |
| golang-jwt/jwt | JWT token verification |
| creack/pty | PTY for terminal |

## Directory Structure

```
server/
├── cmd/
│   └── gloski/
│       └── main.go           # Entry point
│
└── internal/
    ├── app/
    │   └── app.go            # Application container
    │
    ├── api/
    │   ├── server.go         # HTTP server setup
    │   ├── handlers/         # HTTP handlers
    │   │   ├── auth.go
    │   │   ├── files.go      # File ops + pinned folders
    │   │   ├── system.go
    │   │   ├── jobs.go
    │   │   ├── systemd.go
    │   │   ├── terminal.go
    │   │   ├── cron.go
    │   │   ├── packages.go
    │   │   ├── downloads.go
    │   │   ├── health.go
    │   │   └── response.go
    │   ├── middleware/       # HTTP middleware
    │   │   ├── auth.go
    │   │   ├── cors.go
    │   │   ├── logging.go
    │   │   ├── ratelimit.go
    │   │   └── chain.go
    │   ├── routes/
    │   │   └── routes.go     # Route definitions
    │   └── response/
    │       └── response.go   # Response helpers
    │
    ├── auth/
    │   └── service.go        # API key + JWT authentication
    │
    ├── config/
    │   └── config.go         # Configuration loading
    │
    ├── database/
    │   └── migrations.go     # SQLite migrations
    │
    ├── files/
    │   └── service.go        # File operations
    │
    ├── system/
    │   ├── service.go        # System stats API
    │   ├── collector.go      # Background stats collector
    │   ├── store.go          # Time-series ring buffer
    │   └── types.go          # Data structures
    │
    ├── jobs/
    │   └── service.go        # Job/process management
    │
    ├── systemd/
    │   └── service.go        # Systemd service management
    │
    ├── terminal/
    │   └── terminal.go       # PTY terminal
    │
    ├── cron/
    │   └── service.go        # Cron job management
    │
    ├── packages/
    │   └── service.go        # Package manager integration
    │
    ├── downloads/
    │   └── service.go        # Download manager
    │
    └── logger/
        └── logger.go         # Structured logging
```

## Application Container

The `App` struct in `internal/app/app.go` is the central dependency container:

```go
type App struct {
    Config *config.Config
    DB     *sql.DB  // SQLite database

    // Core services (always available)
    Auth   *auth.Service
    Files  *files.Service
    System *system.Service
    Jobs   *jobs.Service

    // Optional services (may be nil)
    Packages  *packages.Service
    Cron      *cron.Service
    Systemd   *systemd.Service
    Downloads *downloads.Service

    // Background services
    statsCollector *system.Collector
}
```

### Initialization

```go
func New(cfg *config.Config) (*App, error) {
    // 1. Create stats store (ring buffer for 5 min history)
    statsStore := system.NewStore(150)
    
    // 2. Start background collector (2s interval)
    collector := system.NewCollector(statsStore, 2*time.Second)
    collector.Start()
    
    // 3. Initialize core services
    app.Auth = auth.NewService(cfg)
    app.Files = files.NewService(cfg)
    app.System = system.NewService(statsStore)
    app.Tasks = tasks.NewService()
    
    // 4. Initialize optional services (may fail gracefully)
    app.Packages, _ = packages.NewService()
    app.Cron, _ = cron.NewService()
    
    return app, nil
}
```

## Authentication

The server supports two authentication methods:

### 1. API Key

Simple static key authentication:

```
Header: X-API-Key: your-api-key
Query:  ?api_key=your-api-key
```

### 2. JWT (RS256)

Asymmetric JWT verification - server only needs the public key:

```
Header: Authorization: Bearer <token>
Query:  ?token=<jwt-token>
```

Tokens are signed externally (e.g., by a central auth service).

### Auth Service

```go
// internal/auth/service.go
type Service struct {
    apiKey       string
    jwtPublicKey *rsa.PublicKey
}

// Validate API key
func (s *Service) ValidateAPIKey(key string) error

// Validate JWT token (RS256)
func (s *Service) ValidateToken(tokenString string) error
```

### Auth Middleware

```go
// internal/api/middleware/auth.go
func Auth(authService *auth.Service) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Try API key first (header or query)
            if apiKey := getAPIKey(r); apiKey != "" {
                if authService.ValidateAPIKey(apiKey) == nil {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            
            // Try JWT token (header or query)
            if token := getToken(r); token != "" {
                if authService.ValidateToken(token) == nil {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            
            http.Error(w, "Unauthorized", 401)
        })
    }
}
```

## Configuration

Configuration is loaded from environment variables and/or config file.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_HOST` | `127.0.0.1` | Bind address |
| `GLOSKI_PORT` | `8080` | Port |
| `GLOSKI_API_KEY` | (none) | API key |
| `GLOSKI_JWT_PUBLIC_KEY` | (none) | PEM-encoded RSA public key |
| `GLOSKI_JWT_PUBLIC_KEY_FILE` | (none) | Path to public key PEM file |
| `GLOSKI_LOG_LEVEL` | `info` | Log level |
| `GLOSKI_SHELL` | `$SHELL` | Shell for terminal |
| `GLOSKI_ALLOWED_ORIGINS` | `*` | CORS origins (comma-separated) |
| `GLOSKI_ALLOWED_PATHS` | `/` | Allowed filesystem paths |

### Config File

`~/.config/gloski/config.json`:

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "api_key": "your-secure-key",
  "jwt_public_key_file": "/path/to/public.pem",
  "allowed_origins": ["https://your-frontend.com"],
  "allowed_paths": ["/home/user"],
  "log_level": "info"
}
```

## Services

### Files Service

```go
// internal/files/service.go
type Service struct {
    allowedPaths []string
}

func (s *Service) List(path string) ([]FileEntry, error)
func (s *Service) Read(path string) (string, error)
func (s *Service) Write(path, content string) error
func (s *Service) Mkdir(path string) error
func (s *Service) Delete(path string) error
func (s *Service) Rename(oldPath, newPath string) error
func (s *Service) Search(root, pattern string, content bool) ([]SearchResult, error)
```

### System Service

```go
// internal/system/service.go
type Service struct {
    store *Store  // Ring buffer for history
}

func (s *Service) GetStats() (*Stats, error)
func (s *Service) GetHistory(duration time.Duration) ([]Sample, error)
func (s *Service) GetProcesses(limit int) ([]ProcessInfo, error)
func (s *Service) GetInfo() (*SystemInfo, error)
```

### Stats Collector

Background goroutine that collects system stats:

```go
// internal/system/collector.go
type Collector struct {
    store    *Store
    interval time.Duration
    stop     chan struct{}
}

func (c *Collector) Start()  // Starts collection goroutine
func (c *Collector) Stop()   // Stops collection
```

### Stats Store (Ring Buffer)

```go
// internal/system/store.go
type Store struct {
    samples  []Sample
    capacity int
    head     int
    count    int
    mu       sync.RWMutex
}

func (s *Store) Add(stats *Stats)
func (s *Store) GetHistory(duration time.Duration) []Sample
func (s *Store) GetLatest() *Sample
```

### Jobs Service

```go
// internal/jobs/service.go
type Service struct {
    db *sql.DB  // SQLite for persistence
    mu sync.RWMutex
}

func (s *Service) Start(command, cwd string) (*Job, error)
func (s *Service) Get(id string) (*Job, error)
func (s *Service) List() []*Job
func (s *Service) Stop(id string) error
func (s *Service) Delete(id string) error
func (s *Service) GetLogs(id string) (string, error)
```

### Systemd Service

```go
// internal/systemd/service.go
type Service struct {}

func (s *Service) List(userMode bool) ([]SystemdUnit, error)
func (s *Service) Action(unit, action string, userMode bool) error
func (s *Service) Logs(unit string, userMode bool, lines int) (string, error)
func (s *Service) Status(unit string, userMode bool) (*SystemdUnit, error)
```

### Terminal

WebSocket-based PTY terminal:

```go
// internal/terminal/terminal.go
type Terminal struct {
    pty  *os.File
    cmd  *exec.Cmd
    conn *websocket.Conn
}

func New(shell, cwd string, conn *websocket.Conn) (*Terminal, error)
func (t *Terminal) Run() error  // Blocks until terminal closes
```

## HTTP Handlers

### Handler Pattern

```go
// internal/api/handlers/files.go
type FilesHandler struct {
    service *files.Service
}

func NewFilesHandler(service *files.Service) *FilesHandler {
    return &FilesHandler{service: service}
}

func (h *FilesHandler) List(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Query().Get("path")
    if path == "" {
        path = "/"
    }
    
    entries, err := h.service.List(path)
    if err != nil {
        response.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    response.JSON(w, map[string]interface{}{
        "path":    path,
        "entries": entries,
    })
}
```

### Response Helpers

```go
// internal/api/response/response.go
func JSON(w http.ResponseWriter, data interface{})
func Error(w http.ResponseWriter, message string, status int)
func Success(w http.ResponseWriter, message string)
```

## Routes

Routes are defined in `internal/api/routes/routes.go` using Go 1.22's enhanced router:

```go
func Setup(cfg Config) http.Handler {
    mux := http.NewServeMux()
    
    // Public routes
    mux.HandleFunc("GET /api/health", healthHandler.Check)
    
    // Protected routes
    mux.Handle("GET /api/system/stats", requireAuth(systemHandler.GetStats))
    mux.Handle("GET /api/files", requireAuth(filesHandler.List))
    mux.Handle("POST /api/files/write", requireAuth(filesHandler.Write))
    
    // Routes with path parameters
    mux.Handle("GET /api/tasks/{id}", requireAuth(tasksHandler.Get))
    mux.Handle("DELETE /api/tasks/{id}", requireAuth(tasksHandler.Stop))
    
    // Apply global middleware
    return middleware.Chain(
        middleware.Logging,
        middleware.CORS(corsConfig),
    )(mux)
}
```

## Middleware

### Middleware Chain

```go
// internal/api/middleware/chain.go
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
    return func(final http.Handler) http.Handler {
        for i := len(middlewares) - 1; i >= 0; i-- {
            final = middlewares[i](final)
        }
        return final
    }
}
```

### CORS Middleware

```go
// internal/api/middleware/cors.go
type CORSConfig struct {
    AllowedOrigins []string
    AllowedMethods []string
    AllowedHeaders []string
    MaxAge         int
}

func CORS(config CORSConfig) func(http.Handler) http.Handler
```

### Logging Middleware

```go
// internal/api/middleware/logging.go
func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        wrapped := &responseWriter{ResponseWriter: w}
        
        next.ServeHTTP(wrapped, r)
        
        logger.Info("%s %s %d %v",
            r.Method, r.URL.Path,
            wrapped.status, time.Since(start))
    })
}
```

## Adding a New Endpoint

### 1. Create Service Method

```go
// internal/myservice/service.go
func (s *Service) DoSomething(input string) (Result, error) {
    // Business logic
}
```

### 2. Create Handler

```go
// internal/api/handlers/myservice.go
type MyHandler struct {
    service *myservice.Service
}

func NewMyHandler(s *myservice.Service) *MyHandler {
    return &MyHandler{service: s}
}

func (h *MyHandler) DoSomething(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Input string `json:"input"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    result, err := h.service.DoSomething(req.Input)
    if err != nil {
        response.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    response.JSON(w, result)
}
```

### 3. Register Route

```go
// internal/api/routes/routes.go
myHandler := handlers.NewMyHandler(cfg.MyService)
mux.Handle("POST /api/my-endpoint", requireAuth(http.HandlerFunc(myHandler.DoSomething)))
```

### 4. Initialize Service in App

```go
// internal/app/app.go
app.MyService = myservice.NewService()
```

## Error Handling

Services return errors that handlers translate to HTTP responses:

```go
// Service
var ErrNotFound = errors.New("not found")
var ErrPermissionDenied = errors.New("permission denied")

// Handler
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    result, err := h.service.Get(id)
    if err != nil {
        switch {
        case errors.Is(err, service.ErrNotFound):
            response.Error(w, "Not found", http.StatusNotFound)
        case errors.Is(err, service.ErrPermissionDenied):
            response.Error(w, "Permission denied", http.StatusForbidden)
        default:
            response.Error(w, err.Error(), http.StatusInternalServerError)
        }
        return
    }
    response.JSON(w, result)
}
```

## Building

```bash
cd server

# Development
go run ./cmd/gloski

# Build binary
go build -o gloski ./cmd/gloski

# Build for Linux (cross-compile)
GOOS=linux GOARCH=amd64 go build -o gloski ./cmd/gloski

# Run tests
go test ./...
```

## Logging

```go
// internal/logger/logger.go
func Debug(format string, args ...interface{})
func Info(format string, args ...interface{})
func Warn(format string, args ...interface{})
func Error(format string, args ...interface{})

// Usage
logger.Info("Server started on %s:%d", host, port)
logger.Error("Failed to read file: %v", err)
```

Log level is controlled by `GLOSKI_LOG_LEVEL` env var.
