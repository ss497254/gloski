# Data model

All durable state is in the master's SQLite database. Agents keep no durable
state: credentials, running jobs, terminals and buffered output live only in
memory.

## Where the data lives

| Path | Contents |
|---|---|
| `<data_dir>/master.db` | The database (`Config.MasterDBPath`) |
| `<data_dir>/master-join-token` | Enrollment token, generated on first start, mode `0600` |
| `<data_dir>/known_hosts` | SSH host keys trusted by the master |
| `<data_dir>/agent.sock` | Unix socket for the master's own agent |
| `<data_dir>/gloski-agent` | Local agent binary written from the embedded build |

The database opens with WAL, `busy_timeout=5000`, foreign keys on,
`synchronous=NORMAL`, and immediate transactions. Pragmas must be written as
`_pragma=name(value)` in the DSN; the older mattn-style parameters are silently
ignored.

## Migrations

`schema_migrations(version, applied_at)` records the highest applied version.
Migrations live in `server/internal/database/migrations.go`.

| Version | Change |
|---|---|
| 1 | Legacy `downloads`, `download_share_links` |
| 2 | Legacy `jobs` |
| 3 | Legacy `pinned_folders` |
| 4 | `devices` |
| 5 | `deployments`, `devices.deploy_id` |
| 7 | `devices.master` (version 6 was skipped and never existed) |
| 8 | `devices.machine_id` |
| 9 | `check_results` |
| 10 | `device_actions` |
| 11 | `metric_samples` |
| 12 | `agent_jobs`, `agent_job_logs` (and the `last_seq`, `log_bytes`, `log_truncated` columns) |

Migration 12 owns job storage on a fresh database and upgrades databases where
`agents.NewJobStore` used to create the tables at startup. It backfills
`log_bytes` from `agent_job_logs` when adding that column. A schema change to
jobs now belongs in a migration; `NewJobStore` is a plain read/write store.

## Tables in use

### `devices`

One row per enrolled device, keyed by name for identity. Re-registering a name
rotates its token and keeps its ID.

| Column | Meaning |
|---|---|
| `id` | UUID, primary key |
| `name` | Display name, unique; the current identity key |
| `url` | Display label (`grpc://agent`) |
| `token` | Device token, unique; rotated on every register |
| `hostname`, `platform`, `version` | Reported by the agent |
| `stats_json` | Last full `system.Stats` marshalled as JSON |
| `created_at`, `last_seen`, `updated_at` | Timestamps; `last_seen` moves on heartbeat |
| `deploy_id` | Link to `deployments.id`, empty for unmanaged devices |
| `master` | `1` for the master's own device |
| `machine_id` | Agent-reported machine identity; stored but not returned by the API |

`status` is not a column. It is computed: online while the agent stream is
connected, otherwise offline. Systems without a live connection check fall back
to `last_seen`.

### `deployments`

One row per SSH installation. Tokens are never stored here.

| Column | Meaning |
|---|---|
| `id` | UUID |
| `name`, `host`, `ssh_user`, `ssh_port`, `agent_port`, `master_url` | Install target |
| `status` | See below |
| `remote_pid` | PID reported by the install script |
| `device_id` | Set when the device enrolls |
| `error` | Last failure, in plain words |
| `created_at`, `updated_at` | Timestamps |

Statuses: `pending` (record created), `running` (install succeeded),
`stopped` (owner stopped it; the lifecycle loop leaves it alone), `failed`
(install or stop failed). `uploading` is declared but never set.

### `check_results`

One row per run of a saved check. There is one `check_id` per check definition
(the definitions themselves are not stored yet); a check that does not apply to
a device records `skipped`, and a check that has never run has no row at all.
Neither looks like a failure.

| Column | Meaning |
|---|---|
| `id` | Autoincrement; breaks ties within the same `ran_at` |
| `check_id`, `device_id` | The pair this result belongs to |
| `status` | `ok`, `warning`, `critical`, `skipped` or `unknown` |
| `summary` | One factual line from the command's first output line |
| `details_json` | Ordered key/value details from the remaining output |
| `ran_at` | When the check ran |
| `duration_ms` | How long it took |
| `interval_ns` | The check's schedule at run time; staleness is `now - ran_at > 2 × interval` |

Native probes write into this same table: one aggregated result per
(device, check, interval), computed from a bounded in-memory window of recent
runs. Probe definitions are not stored yet; they come from `GLOSKI_PROBES`. The
index is `(check_id, device_id, ran_at DESC, id DESC)` so the latest result and
a short history are cheap to read. `checks.Store.LatestForAllDevices`
returns one row per pair across the fleet for the dashboard matrix; a device
with no row has an `unknown` result, which is not a failure. `checks.Store.Prune`
keeps the newest 20 rows per pair by default (`HistoryLimit`); it is intended
for the daily maintenance pass.

### `metric_samples`

One row per metric value recorded from a device heartbeat. The master writes
one sample per (device, metric) at most every five minutes; the agent's in-memory ring is
separate and is not sent to the master.

| Column | Meaning |
|---|---|
| `device_id`, `metric` | The series this value belongs to |
| `ts` | When the master sampled it (UTC) |
| `day` | `ts` formatted as `YYYY-MM-DD`; stored explicitly because the SQLite driver's time format is not understood by `date()` |
| `value` | The metric value; percentages for CPU, memory, swap and disks, load1, and bytes/sec for network rates |

Metric names are stable: `cpu`, `memory`, `swap`, `load1`, `network_rx`,
`network_tx` and `disk:<mount_point>`. There are two indexes:
`(device_id, metric, ts DESC)` for chart reads and
`(device_id, ts DESC)` for the throttle's newest-timestamp lookup. The daily maintenance pass keeps full
resolution for 7 days, then one sample per (device, metric, day) up to 365 days,
and deletes older rows. It never touches job or check retention.

### `device_actions`

One master-owned change to a device. File deletions will be logged here later
but are not undoable; today only agent start/stop/restart are.

| Column | Meaning |
|---|---|
| `id` | UUID |
| `device_id` | Owning device |
| `kind` | `agent_start`, `agent_stop`, `agent_restart`, `trust_host`, `undo` |
| `target` | What was changed (`agent` today) |
| `from_value`, `to_value` | Wanted deployment state before and after |
| `result`, `error` | `ok` or `failed`, and why it failed |
| `undoable` | `1` when the previous wanted state is known |
| `undone_at` | Set when undo succeeded; an action cannot be undone twice |
| `created_at` | When the change was made |

The index is `(device_id, created_at DESC)`. The device page shows the newest
20 entries.

### `agent_jobs`

A command the master owns.

| Column | Meaning |
|---|---|
| `id` | UUID and gRPC request ID |
| `device_id` | Owning device |
| `command`, `cwd` | What to run |
| `status` | See below |
| `exit_code` | Agent-reported; `0` until a result arrives |
| `error` | Failure text, if any |
| `created_at`, `started_at`, `finished_at` | Timestamps |
| `last_seq` | Newest output sequence stored; used to drop replays and mark gaps |
| `log_bytes` | Bytes of log stored so far; the cap is 10 MB per job |
| `log_truncated` | `1` once the truncation notice has been written |

### `agent_job_logs`

One row per streamed output chunk.

| Column | Meaning |
|---|---|
| `id` | Autoincrement, orders chunks |
| `job_id` | Owning job |
| `chunk` | Raw bytes, stdout or stderr |
| `stderr` | `1` for stderr and for the synthetic gap and truncation notices |
| `created_at` | Timestamp |

Indexes: `agent_jobs(device_id)`, `agent_job_logs(job_id)`.

## Status lifecycles

**Job**

```
pending ──▶ running ──▶ finished
   │           ├──────▶ failed
   │           └──────▶ stopped
   └──────────────────▶ failed       (command could not be sent)
pending/running ──────▶ abandoned    (agent no longer holds it after reconnect)
```

`Finish` only moves a job out of `pending` or `running`, so a late result cannot
overwrite a terminal state. Deleting a device deletes its jobs and logs.
Each job keeps at most 10 MB of log; the chunk that crosses the limit is
replaced by a truncation notice. A daily maintenance pass deletes terminal jobs
(and their logs) finished more than 30 days ago.

**Deployment**

```
pending ──▶ running ──▶ stopped ──▶ running
   │           │
   └───────────┴──────▶ failed
```

The master's lifecycle loop only re-installs `running` deployments that are
disconnected or outdated. A `stopped` deployment is left alone until someone
presses start.

## Legacy tables

`downloads`, `download_share_links`, `jobs` and `pinned_folders` are from the
removed standalone mode. Nothing in the current code reads or writes them. They
are kept because dropping tables is a one-way change and costs nothing on a
small database.

`Config.DatabasePath()` still returns `<data_dir>/gloski.db`, but the master
opens `master.db`. Do not use the old path for new code.
