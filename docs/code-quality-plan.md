# Code readability and documentation plan

Status: in progress, 2026-11-19. Batches 1 and 2 are done.

## Why

The system works, but reading it is harder than it should be. The causes are
vocabulary drift, a few very large units, and documentation that stops at the
system diagram. The fix is incremental and behavior-preserving: move code,
rename things, write down what exists. No rewrite.

## Rules for every batch

1. No behavior changes. If a bug surfaces while moving code, note it and fix it
   in a separate commit.
2. One word per concept. Canonical: **master**, **agent**, **device**, **job**,
   **file**, **dashboard**. "Tower" stays a product metaphor in prose, not a
   second name for the master in code.
3. A file does one job and stays readable. Target: under about 300 lines of
   hand-written code, except tests and generated code.
4. Comments explain why, constraints and invariants, not what the next line
   does.
5. Every Go package has a package comment, every exported Go symbol has a doc
   comment, and every HTTP route appears in `docs/http-api.md`.
6. Docs change in the same commit as the code they describe.
7. Each batch is shippable on `main` on its own. `go test ./...`,
   `bun run type-check`, `bun test`, and a live smoke test pass before it lands.

## What is wrong today

| Symptom | Where | Why it hurts |
|---|---|---|
| One process, four names: master, tower, control tower, server | `internal/controltower`, `Tower`, docs | New code and readers cannot tell whether these mean the same thing |
| `Tower` is a 24-field god object | `internal/controltower/server.go` | DB, HTTP, gRPC, SPA, lifecycle and the local child all hang off one type, so every handler reaches through it |
| One file owns path policy, listing, read/write, delete/rename, upload and search | `internal/files/service.go` (751 lines) | Changes to unrelated operations collide; policy is duplicated |
| Delete and Rename each carry their own dangerous-path list, and the lists disagree | `internal/files/service.go:294`, `:317` | Duplicated policy has already drifted: `/dev`, `/proc`, `/sys`, `/run`, `/root` are protected from delete but not rename |
| Job execution, file operations, process listing and JSON building in one file | `internal/worker/agent_jobs.go` (419 lines) | A reader looking for file ops does not find them in `agent_files.go` |
| An 825-line context exposing 77 fields and handlers | `web/src/features/files/context/FilesContext.tsx` | Navigation, preview, editing, upload, dialogs, selection, sorting and focus all change the same file |
| Only about half the Go packages have a package comment; no HTTP API, data-model or "where do I add X" reference | `server/internal/*`, `docs/` | Docs explain the system but not the code; `routes.go` is the only route list |
| `CONTRIBUTING.md` requires feature branches and PRs | `CONTRIBUTING.md` | Contradicts the actual rule: commit straight to `main` |
| Bare string errors and a stringly-typed file operation | `internal/files`, `fileRequest(path, "list")` | Call sites are opaque and expected failures cannot be handled without string matching |
| No Go linter | repository | Nothing enforces package docs, exported docs or file size; the frontend has ESLint |

## Batches

### 1. Make the map true (docs only)

- `docs/glossary.md`: one canonical word per concept, and the words to avoid.
- Package comments for the roughly dozen packages missing one; doc comments for
  exported symbols that lack them.
- `docs/http-api.md`: method, path, body, response and errors for every route.
- `docs/data-model.md`: tables, columns, and the job status lifecycle.
- `docs/where-to-change-what.md`: "add an endpoint", "add an agent command",
  "add a page", "change the schema", with the exact files.
- Fix `CONTRIBUTING.md` (main-only workflow), and stale claims in README and
  `docs/backend.md`.

Risk: none. Verify: `go doc` output, doc links, unchanged build.

### 2. Split the shared file code

- `internal/files/service.go` becomes `path.go` (validation and protected-path
  policy), `list.go`, `read.go`, `write.go`, `delete.go`, `rename.go`,
  `upload.go`, `search.go`.
- One protected-path list used by delete and rename, with one scope each:
  Delete refuses the whole tree (except under `/tmp`), Rename refuses only the
  roots, so moving a file inside a system directory stays legal. Write its tests
  before moving the code.
- Move file operations out of `internal/worker/agent_jobs.go` into
  `agent_files.go`; leave jobs and processes where they are.

Risk: low. Verify: `go test ./...` and a live file list/read/write/rename/delete.

### 3. Shrink the master's god object

- `Tower` keeps construction, ownership and shutdown only.
- Each handler group gets a small struct with the dependencies it actually uses
  (files, jobs, terminal, deployment); `routes.go` wires them.
- Move the master-side `JobStore` out of `controltower/agents` into a jobs
  package, so the `agents` package is only the stream.

Risk: medium. Verify: existing tests plus a live job, terminal and deploy.

### 4. Break up the files feature

- Sort, filter, path and selection logic move to `lib/` with tests.
- State and actions split into hooks: directory, preview, actions, upload,
  dialogs, selection. The provider composes them; components do not change at
  first.
- Only after that, narrow the context value if components allow it.

Risk: medium. Verify: type-check, `bun test`, manual list/preview/save/upload.

### 5. Typed errors and honest names

- Sentinel errors for expected failures, wrapping with context, one mapping to
  HTTP status and message.
- Replace `fileRequest(path, "list")` with explicit constructors.
- Rename packages to the glossary: `controltower` to `master`, `worker` to
  `agent`, `controltower/agents` to `master/agentstream`. Mechanical and last,
  so it does not hide real changes.

Risk: medium diff, no behavior change. Verify: full test suite and smoke test.

### 6. Keep it readable

- `golangci-lint` with a small set: `govet`, `errcheck`, `staticcheck`,
  `revive` exported-doc rule, `gofmt`. `make lint`.
- CONTRIBUTING checklist: package doc, exported doc, route doc, file size,
  docs touched.

Risk: low.

## Order

Batches 1 and 2 first: docs settle the vocabulary, splitting files gives
immediate relief. Batches 3 to 5 are separate commits. Renaming is last.

## What "done" looks like

- A newcomer can open `docs/where-to-change-what.md` and add an endpoint or an
  agent command without reading the whole tree.
- No hand-written file over about 400 lines except tests and generated code.
- Every package and exported symbol has a doc comment; every route is in the API
  reference.
- `Tower` no longer carries feature logic.
- The files feature can be changed one concern at a time.
- Tests and the live smoke test are green after every batch.

## Not doing

- No rewrite, no new architecture, no dependency-injection framework.
- No interfaces with one implementation, no generic repositories.
- No behavior changes hidden inside refactors.
- No CI or pre-commit hook unless asked.

## Progress

- [x] Batch 1: docs map (`19f3769`)
- [x] Batch 2: files split (`8fc397f`)
- [ ] Batch 3: shrink `Tower`
- [ ] Batch 4: files feature split
- [ ] Batch 5: errors and renames
- [ ] Batch 6: lint gate
