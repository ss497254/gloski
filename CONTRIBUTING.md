# Contributing to Gloski

This guide will help you get started with developing Gloski.

## Prerequisites

- **Go 1.24+** - for the server and agent
- **Bun** - for the frontend
- **Git** - for version control

## Development Setup

### 1. Clone and Install

```bash
git clone https://github.com/ss497254/gloski.git
cd gloski

# Install frontend dependencies
cd web && bun install
```

### 2. Run Development Servers

You need two terminals:

**Terminal 1 - Backend:**
```bash
cd server
go run ./cmd/gloski
```

**Terminal 2 - Frontend:**
```bash
cd web
bun run dev
```

### 3. Access the App

Open http://localhost:3000. The dev server proxies API calls to the master on
`127.0.0.1:8080`; the master starts its own local agent, so one device appears
without any setup.

## Project Structure Overview

```
gloski/
├── server/          # Go backend
├── web/             # React frontend
├── docs/            # Documentation
├── README.md        # Project overview
└── CONTRIBUTING.md  # This file
```

See [docs/frontend.md](docs/frontend.md) and [docs/backend.md](docs/backend.md) for detailed architecture.

## Code Style

### Frontend (TypeScript/React)

- **Formatting**: Prettier (runs on save)
- **Linting**: ESLint with TypeScript rules
- **Naming**:
  - Components: `PascalCase` (e.g., `FilePreview.tsx`)
  - Hooks: `camelCase` with `use` prefix (e.g., `useFilesPage.ts`)
  - Stores: `camelCase` (e.g., `files.ts`)
  - Utils: `camelCase` (e.g., `formatBytes`)

### Backend (Go)

- **Formatting**: `gofmt` / `goimports`
- **Naming**: Standard Go conventions
- **Structure**: Clean architecture with `internal/` packages

## Making Changes

### Frontend Changes

1. **Feature code** goes in `web/src/features/{feature-name}/`
2. **Shared components** go in `web/src/shared/components/`
3. **Shared hooks** go in `web/src/shared/hooks/`
4. **UI primitives** (shadcn) go in `web/src/ui/`

### Backend Changes

1. **Browser endpoints** go in `server/internal/controltower/` and are registered in `routes.go`
2. **Master-to-agent commands** go in `server/internal/controltower/agents/`
3. **Device-side handling** goes in `server/internal/worker/`
4. **Protocol changes** go in `server/proto/agentv1/agent.proto` (regenerate the checked-in Go code)

## Testing Your Changes

### Frontend

```bash
cd web

# Type checking
bun run type-check

# Build (catches errors)
bun run build

# Lint
bun run lint
```

### Backend

```bash
cd server

# Run tests
go test ./...

# Build
go build ./cmd/gloski
```

## Common Tasks

### Adding a New Feature Page

1. Create feature folder: `web/src/features/{name}/`
2. Add page component: `pages/{Name}Page.tsx`
3. Add the nav item in `web/src/app/navigation.ts`
4. Add the route in `web/src/App.tsx`

### Adding a New API Endpoint

1. Add the handler in `server/internal/controltower/`
2. Register the route in `server/internal/controltower/routes.go`
3. Add the client method in `web/src/shared/api/` (device calls) or `web/src/shared/tower/client.ts` (master calls)

### Adding a Shared Component

1. Create component in `web/src/shared/components/`
2. Export from `web/src/shared/components/index.ts`

## Git Workflow

1. Create a feature branch: `git checkout -b feature/my-feature`
2. Make your changes
3. Test: `bun run build` (frontend) and `go build ./cmd/gloski` and `go build ./cmd/gloski-agent` (backend)
4. Commit with clear messages
5. Push and create a PR

### Commit Message Format

```
type: short description

- Detail 1
- Detail 2
```

Types: `feat`, `fix`, `refactor`, `docs`, `style`, `test`, `chore`

## Need Help?

- Check the [docs/](docs/) folder for detailed documentation
- Look at existing code for patterns
- Open an issue for questions

## Documentation

- [Frontend Architecture](docs/frontend.md)
- [Backend Architecture](docs/backend.md)
- [Responsive Design Patterns](docs/responsive-design.md)
