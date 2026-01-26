# Contributing to Gloski

This guide will help you get started with developing Gloski.

## Prerequisites

- **Go 1.22+** - for the server
- **Node.js 20+** - for the frontend
- **Git** - for version control

## Development Setup

### 1. Clone and Install

```bash
git clone https://github.com/ss497254/gloski.git
cd gloski

# Install frontend dependencies
cd web && npm install
```

### 2. Run Development Servers

You need two terminals:

**Terminal 1 - Backend:**
```bash
cd server
GLOSKI_API_KEY=dev-key go run ./cmd/gloski
```

**Terminal 2 - Frontend:**
```bash
cd web
npm run dev
```

### 3. Access the App

1. Open http://localhost:5173
2. Click "Add Server"
3. URL: `http://127.0.0.1:8080`
4. API Key: `dev-key`

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

1. **Handlers** go in `server/internal/api/handlers/`
2. **Services** go in `server/internal/{service-name}/`
3. **New routes** are registered in `server/internal/api/routes/routes.go`

## Testing Your Changes

### Frontend

```bash
cd web

# Type checking
npm run typecheck

# Build (catches errors)
npm run build

# Lint
npm run lint
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
3. Add to feature registry: `web/src/app/feature-registry.ts`
4. Add route: `web/src/app/routes.tsx`

### Adding a New API Endpoint

1. Create handler in `server/internal/api/handlers/`
2. Register route in `server/internal/api/routes/routes.go`
3. Add API method in `web/src/shared/services/api.ts`

### Adding a Shared Component

1. Create component in `web/src/shared/components/`
2. Export from `web/src/shared/components/index.ts`

## Git Workflow

1. Create a feature branch: `git checkout -b feature/my-feature`
2. Make your changes
3. Test: `npm run build` (frontend) and `go build ./cmd/gloski` (backend)
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
