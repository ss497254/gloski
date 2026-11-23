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

See [docs/frontend.md](docs/frontend.md) and [docs/backend.md](docs/backend.md) for detailed architecture. For the vocabulary, the HTTP contract and the common change paths, start with the [glossary](docs/glossary.md), [HTTP API](docs/http-api.md), [data model](docs/data-model.md) and [where to change what](docs/where-to-change-what.md).

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

This is a solo project. Commit straight to `main`; do not create feature
branches, pull requests or force-pushes. Do not push unless the owner asks.

1. Make the change on `main`.
2. Run the backend and frontend checks below.
3. Commit with a clear message. The `pre-commit` hook runs
   `scripts/review-checks.sh` and then the read-only `reviewer` subagent; it
   fails the commit on a check failure or a `VERDICT: BLOCK`. Enable it once
   per clone with `scripts/install-git-hooks.sh` (or `make install-hooks`). For
   an emergency, `GLOSKI_SKIP_REVIEW=1 git commit ...` bypasses it; say why
   when you do.
4. Keep `main` buildable after every commit. Split unrelated work into
   separate commits.

You can also run `/review` in OpenCode at any time, or
`scripts/review-checks.sh` on its own. The reviewer's checklist lives in
`.opencode/agents/reviewer.md`. When the same kind of mistake shows up twice,
add a rule there so the next review catches it.

### When another agent shares the clone

The git index belongs to the whole checkout, not to one agent. A bare
`git add -A` or `git commit` commits files another agent staged, and their
half-finished edits can fail your pre-commit checks. Prefer one worktree per
agent (`worktree_create` and `session_move` in OpenCode). If agents must share
a checkout, never run a bare `git commit` — use:

```bash
scripts/commit-only.sh "type: message" path/to/file.ts other/path.go
```

It runs `git commit --only` for exactly those paths. The pre-commit hook then
checks only the files in that commit (`GLOSKI_CHECK_STAGED=1`), so unrelated
staged work cannot block you or ride along.

### Commit Message Format

```
type: short description

- Detail 1
- Detail 2
```

Types: `feat`, `fix`, `refactor`, `docs`, `style`, `test`, `chore`

## Documentation Checklist

Code changes are not done until the matching docs are true:

- New or changed HTTP route: update [docs/http-api.md](docs/http-api.md).
- Schema or status change: update [docs/data-model.md](docs/data-model.md).
- New package or exported Go symbol: add its doc comment.
- New page, route or client: update [docs/frontend.md](docs/frontend.md).
- New term or a renamed concept: update [docs/glossary.md](docs/glossary.md).
- Moving where a kind of change belongs: update
  [docs/where-to-change-what.md](docs/where-to-change-what.md).

## Need Help?

- Check the [docs/](docs/) folder for detailed documentation
- Look at existing code for patterns
- Open an issue for questions

## Documentation Map

- [Architecture Overview](docs/architecture.md)
- [Glossary](docs/glossary.md)
- [HTTP API](docs/http-api.md)
- [Data Model](docs/data-model.md)
- [Where to Change What](docs/where-to-change-what.md)
- [Backend Architecture](docs/backend.md)
- [Frontend Architecture](docs/frontend.md)
- [Control Tower](docs/tower.md)
- [Responsive Design Patterns](docs/responsive-design.md)
- [SDK](docs/sdk.md)
- [Code Quality Plan](docs/code-quality-plan.md)
- [Roadmap](docs/roadmap.md)
