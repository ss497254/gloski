# 09 — Testing and quality

Beszel's test suites are unusually thorough: 66 `*_test.go` files under `agent/`
(~22.5k LOC) plus frontend tests and large hub test files
(`agent_connect_test.go` 55 KB, `api_test.go` 57 KB, `client_test.go` 25 KB).
The *patterns* are framework-independent and are the most transferable part of
the Go side. Class **A** throughout unless noted.

## Go test patterns worth adopting

### 1. Build tag keeps internals private

`agent/agent_test_helpers.go` (13 LOC) carries `//go:build testing` and exposes
`GetConnectionManager` / `GetState` only to tests. `health/health_test.go` (140)
does the same. Production builds contain no test hooks.

Gloski currently runs `go test -tags=testing ./...` in places; standardise this
pattern where a test needs access to unexported state.

### 2. Variable-as-test-seam

External dependencies are package variables overridden in tests:
`uptimeFilePath`, `cgroupRoot`, `emmcSysfsRoot`, `mdraidSysfsRoot`,
`scanSysfsDevices`, etc. Fixtures live in `agent/test-data/` (33 files, ~110 KB):

- `smart/*.json` — nvme, sata, scsi, sat, apple_nvme, permission-denied
- `package_updates/*.txt` — apt ubuntu/debian, dnf4/5, zypper, xbps, pacman, apk
- `zfs/*.txt`, `container*.json`, `nvtop.json`, `amdgpu.ids` (23 KB PCI table)
- `system_info.json`

This avoids introducing an interface solely for testing. Use it for Gloski's
`/proc`, sysfs and command-reading code.

### 3. Self-linking test binary as a fake external command

`agent/gpu_fixture_test.go` (89 LOC) is the cleverest trick in the repo.
`TestMain` re-execs the test binary under the names `nvidia-smi`, `rocm-smi`,
`tegrastats`, `nvtop` and `intel_gpu_top`. A fixture named `<exe>.stdout`
supplies output, `<exe>.args` captures arguments, and `<exe>.fail` makes it exit
nonzero. No shell scripts, no PATH manipulation in the test body, no external
tools required.

Apply this wherever Gloski parses command output (`tailscale`, `smartctl`,
`zpool`, `journalctl`, `docker`).

### 4. `testing/synctest` for time-dependent behaviour

`health/health_test.go:37-62` uses `testing/synctest` instead of real sleeps to
test the 90-second mtime threshold. Go 1.24+ supports it; use it for heartbeat,
backoff and retention tests in Gloski.

### 5. Generic helpers tested directly

`readWithTimeout` with an `atomic.Bool` inflight guard is tested in
`fans_test.go` (249) and `sensors_test.go` (983), including the case where a
blocked read must not stack goroutines. `deltatracker_test.go` (217) covers
first-sample-zero, cycles and concurrency.

### 6. Table-driven tests everywhere

63 of 66 agent test files use at least one of `testify/require`, `t.TempDir()`
and `t.Setenv()`, almost always table-driven. Representative sizes:
`gpu_test.go` 2,256, `docker_test.go` 2,183, `smart_test.go` 1,466,
`disk_test.go` 1,402, `sensors_test.go` 983, `client_test.go` 872,
`network_monitor_test.go` 745, `package_updates_test.go` 582,
`cpu_linux_test.go` 520.

## Frontend tests

See [07](07-frontend.md) for the file list. Patterns to copy: `mock.module()`
for Lingui macros and browser-only modules, dynamic `await import()` after
mocks, `renderToStaticMarkup` for accessibility assertions, `test.each` tables
for formatter cases, and SSR markup assertions for masked tokens.

## Code-quality observations

- Beszel's `agent.go` itself carries `// TODO: this is confusing, make common
  package with common/types common/helpers etc`. It is a god-struct with many
  managers. Do not replicate it; Gloski's `docs/code-quality-plan.md` already
  identifies the same problem in `Tower` and `FilesContext`.
- `internal/site/src/lib/time.ts` is empty; `index.css` etc. coexist with a
  leftover `.prettierrc` alongside Biome. Every mature codebase has dead corners;
  the lesson is to keep cleaning them, not to import them.
- Beszel has **no general Go test CI**. Gloski does. Keep that advantage.
- The frontend has no test script in `package.json`; `bun test` works by
  convention. Gloski's explicit scripts are better.

## Suggested adoption in Gloski

| Pattern | Where | Priority |
|---|---|---|
| `//go:build testing` helper files | `server/internal/**` | low |
| Variable-as-seam for sysfs/`/proc`/command paths | `internal/system`, `internal/provisioning` | medium |
| Self-reexec fake commands | tests for `tailscaled` LocalAPI, `smartctl`, `zpool`, `journalctl` parsing | medium |
| `testing/synctest` for backoff/heartbeat/retention | `controltower`, `fleet` | medium |
| Fixture corpus with real captured output | `server/internal/**/test-data/` | medium |
| `test.each` formatter tests | `web/src/shared/` | high (ships with the utility port) |
| Keep `go test ./...`, `go test -race`, `bun test`, `bun run type-check` in CI | already present | — |
