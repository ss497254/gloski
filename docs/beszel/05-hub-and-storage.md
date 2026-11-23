# 05 — Hub and storage

Almost everything in `internal/hub/` is welded to PocketBase (`core.App` is
passed into every manager, rules are PB filter strings, retention runs via
`app.Cron()`, data lives in PB collections). Treat this document as a catalogue
of **algorithms and operational shapes**, not a code donor. The portable items
are marked A/B.

## Startup, configuration and CLI

| Item | Source | Class | Note |
|---|---|---|---|
| Hub CLI | `internal/cmd/hub/hub.go` (121) | C | Cobra. An early `health` shortcut runs before bootstrapping; `update` is detected via `rootCmd.Find` and run **before** PocketBase bootstrap so it does not create a data dir or open the DB as root. Adopt this rule in Gloski's CLI. |
| Hub struct and lifecycle | `internal/hub/hub.go` (229) | C | Uses an `onAfterBootstrapAndMigrations` workaround because PB's `OnBootstrap` runs before migrations. Wires sync, routes, cron, server, system manager and heartbeat in one `OnServe`. Lazy Ed25519 key generation under mutex. |
| Feature version gates | `beszel.go` (27) | C | Named per-feature minimum versions. See [04](04-protocol-and-transport.md). |
| Batch settings | `internal/hub/hub.go` | C | Enables PB batch requests (max 100, 1 MiB) specifically so alert writes can be batched. |
| YAML config reconciliation | `internal/hub/config/config.go` (287) | C | Keys records by `name+host+port`, converges create/update/delete, only updates tokens when supplied, converts emails to user IDs, can export DB back to YAML. Good pattern if Gloski gains a declarative config. |
| Env prefix fallback | `internal/hub/utils/utils.go` (39) | A | `GetEnv(key)` checks `BESZEL_HUB_`+key first, then bare key — same idea as `GLOSKI_` overrides. |
| Build-tag feature switch | `internal/ghupdate/selfupdate{,_disabled}.go` (15) | A | `no_self_update` compiles self-update out with a clear error. |

## Collections — what a mature fleet tool stores

Schema snapshot: `internal/migrations/0_collections_snapshot_0_20_0.go` (2,315
LOC), imported via PB's `ImportCollectionsByMarshaledJSON`.

| Collection | Key fields | Note | Class |
|---|---|---|---|
| `systems` | id, name, status (up/down/paused/pending), host, port, info JSON, users relation, created/updated | One row per device | C |
| `users` | id, password, tokenKey, email, role (admin/user/readonly) | PB auth collection | D |
| `fingerprints` | id, system, token, fingerprint, updated | Token + fingerprint pair | C |
| `system_stats` / `container_stats` | system, stats JSON, type (1m/10m/20m/120m/480m), created/updated | Tiered time series | B (algorithm only) |
| `network_monitor_stats` | system, monitor, total/success counts, res sum/min/max, type, created as Unix ms | Aggregated probe tiers | B |
| `network_monitors` | stable id, system, target, protocol (icmp/tcp/http/dns), port, interval, result cache, cert info, status codes pattern | Monitor config | B |
| `alerts` | user, system, name (17 types), threshold, duration minutes, triggered, hidden state JSON | Alert definition | C |
| `alerts_history` | user, system, alert, name, value, created, resolved | Incident log | C |
| `quiet_hours` | user, system (empty = global), daily/one-time, start/end | Notification silencing | D |
| `user_settings` | user, settings JSON (chart time, units, colours, hour format, layout, sort) | Preferences | C (shape) |
| `containers` | name, system, state, cpu/peaks, memory/peaks, updated as Unix ms | Current container state; stale rows deleted after 10 min | C |
| `systemd_services` | user, system, type, start, end | Failed-service events; stale after 20 min | C |
| `smart_devices` | system, name, model, state, capacity, temp, firmware, serial, hours, cycles, attributes JSON | Disk health | C |
| `zfs_pools` | system, name, health, size, alloc, free, scrub/vdevs/datasets JSON | Pool cache | C |
| `system_details` | hostname, OS, kernel, CPU, arch, cores, memory, podman, sockets | Static once-per-connection info | C |
| `universal_tokens` | user, token, created | Persistent bootstrap tokens | C |
| `_superusers`, `_externalAuths` | PB-managed | OAuth/admin | D |

Delta migrations are tiny (24–72 LOC each), one numbered file per additive
change, each with an inverse (`down`). Mirror that discipline with plain SQL.

## Rollups and retention — the best portable algorithm here

Source: `internal/records/records.go` (764 LOC) and `records_deletion.go` (131
LOC). Classes **B** (logic is pure Go once PB calls become SQL).

**Tier construction**, run in a transaction:

| Step | Rule |
|---|---|
| 1m → 10m | at least 9 samples |
| 10m → 20m | at least 2 samples |
| 20m → 120m | at least 6 samples |
| 120m → 480m | at least 4 samples |
| Cutoff | `now - period`, with a **+5 min fudge** for non-1m tiers so jobs firing just after the hour do not fall on the boundary |
| Skip | if a longer row already exists for the target period |
| Save | `SaveNoValidate` |

**Averaging `Stats`** (~370 LOC): numeric fields averaged; **peaks preserved**
for CPU, memory, network, bandwidth and disk I/O; per-key maps (WiFi RSSI,
batteries, temperatures, fans, ZFS pools, GPUs, extra filesystems, per-core CPU)
use **per-entry counters**, so a sensor absent from some samples is not averaged
as zero; legacy GB → bytes conversion; two-decimal rounding.

**Averaging containers** (~50 LOC): merge by name, average CPU/mem/bandwidth,
handle legacy MB fields.

**Network-monitor aggregation:** SQL `SUM(total_count) / SUM(success_count) /
SUM(res_sum) / MIN(res_min) / MAX(res_max)`, with derived
`res_avg = res_sum / success_count` and `loss = (total-success)/total*100`.

**Retention:**

| Tier | Keep |
|---|---|
| 1m | 1 hour |
| 10m | 12 hours |
| 20m | 24 hours |
| 120m | 7 days |
| 480m | 30 days |
| Alert history | newest 200 per user once count > 250, via `NOT IN (SELECT … LIMIT 200)` |
| Containers / systemd rows | 10 / 20 minutes |

**Apply to Gloski:** extend the existing daily maintenance pass
(`internal/controltower/maintenance.go`, already responsible for 30-day job
retention) with a `metric_samples(device_id, metric, ts, value)` table using this
tier scheme. Do not add a second scheduler. Combine with the 5-minute sampling
and 30-day-reference ideas in `docs/ideas-from-smdwin.md` §5.

## Self-update pipeline — strong code reuse

`internal/ghupdate/`: `ghupdate.go` (405), `checksum.go` (43), `extract.go`
(148), `release.go` (37), `selinux.go` (66), plus 15 LOC of build-tag switch.
Mostly stdlib; only external dep is `blang/semver`.

The flow:

1. Fetch `/repos/{owner}/{repo}/releases/latest` (or a mirror).
2. Compare semver; exit if not newer.
3. Pick the asset by OS/GOARCH/GOARM/glibc-aware suffix (e.g.
   `beszel-agent_linux_armv7.tar.gz`, `..._glibc.tar.gz`).
4. `os.MkdirTemp(dataDir, ".beszel_update-")`.
5. Download with context and retry.
6. **Streaming SHA-256 verification** against the GitHub asset `digest` field
   (`sha256:<hex>`), with strict algorithm and hex-length validation.
7. Extract tar.gz or zip; `filepath.IsLocal` rejects path traversal; symlinks,
   pipes and sockets are skipped entirely.
8. `os.Rename(oldExec, oldExec+".old")`, then rename the new binary into place.
9. On any failure, revert via a deferred closure; cross-device `EXDEV` falls
   back to a copy preserving mode; deferred cleanup removes the temp dir.

`install-agent.sh` adds a checksums-manifest lookup by filename, `tar -tzf`
pre-flight and a non-empty `-s` check. See [08](08-packaging-and-release.md).

**Apply to Gloski (A/B):** your installer already verifies SHA-256 against
embedded builds. Borrow the *staging on the destination filesystem*, `.bak`
plus revert closure, path containment, and explicit failure phases. There is no
need to fetch from GitHub — the master is the source of truth.

## SPA serving and proxies

| Item | Source | Class | Note |
|---|---|---|---|
| Embedded FS | `internal/site/embed.go` (13) | A | `//go:embed all:dist` + `fs.Sub` — same pattern as Gloski. |
| Runtime info injection | `internal/hub/server.go` (74) | **B** | Marshals `PublicAppInfo` to JSON and string-replaces a `"{info}"` placeholder in `index.html`; rewrites `"./"` for a base path. |
| Production static handler | `internal/hub/server_production.go` (49) | **B** | Caches `/static/` and `/assets/` for 30 days; `CSP` env replaces the default `X-Frame-Options`; **returns 404 status for unknown paths** while still serving the SPA's 404 page, so scanners see the miss. |
| Base-path awareness | `server.go` `isAppRoute()` | C | Keep the frontend route list in sync with the router, with an optional base-path prefix for proxies that may or may not strip it. Gloski's `GLOSKI_ALLOWED_ORIGINS` notes have a similar proxy caveat. |
| Trusted proxy CIDRs | `internal/hub/api.go` (~60) | **B** | `net/netip` IPs/CIDRs, IPv4-mapped normalization (`Unmap`, require ≥/96 for mapped prefixes), `SplitHostPort` then strip zone; **invalid entries are skipped so the allowlist narrows, never widens**. |
| Trusted auth header | `internal/hub/api.go` (~105) | B/C | `AUTO_LOGIN=<email>` or `TRUSTED_AUTH_HEADER`, optionally limited by `TRUSTED_PROXY_IPS`; warns loudly when unrestricted. Gloski's reverse proxy uses basic auth, so this is only a warning-pattern idea for now. |
| Opaque 404 | `internal/hub/api.go` | C | Inaccessible resources return 404 rather than 403, revealing nothing. |

## API, keys and MCP

`internal/hub/api.go` (598) registers roughly 25 routes under `/api/beszel`:
info, update check, notification test, heartbeat status/test, config YAML,
agent connect, universal tokens, API keys, user alerts, SMART/ZFS refresh,
systemd info/logs, package updates, container logs/info, first-run user creation
and MCP. Full route list is in the source; most are irrelevant to Gloski.

Portable ideas (C):

- Static API keys (`api_keys.go`, 102): verify the **presented token**, not the
  resolved identity, for privileged operations, because identity may come from
  auto-login; rotating `tokenKey` revokes all tokens at once.
- MCP server (`internal/hub/mcp/mcp.go` 185, `tools.go` 1,599): stateless
  Streamable-HTTP handler (no sessions to leak), bearer API-key auth with RFC
  6750 challenge, 10 read-only tools, caps of 400 samples / 200 records, and
  tool descriptions that teach the model the retention semantics. Park this for
  Gloski: the diagnose chain already returns structured state.

## Backups — no code to reuse

The README advertises "automatic backups to disk or S3". There is **no beszel
backup code**: it is PocketBase's admin-UI backup feature (PB PRO), linked from
the navbar as `/_/#/settings/backups`. `config-yaml.tsx` only warns "Please make
regular backups". Implement Gloski's nightly DB + join-token copy yourself;
`docs/ideas-from-smdwin.md` §4 has a concrete self-test/backup design.

## Storage shape recommendation for Gloski

Keep all durable state in the existing SQLite database. Add only:

- `metric_samples(device_id, metric, ts, value)` with the retention tiers above.
- `check_results(check_id, device_id, status, summary, details, ran_at, stale)`
  from the smdwin note.
- Optional `device_actions` log if the action log from the smdwin note is built.

Do not introduce PocketBase, a second database, or a separate timeseries store.
