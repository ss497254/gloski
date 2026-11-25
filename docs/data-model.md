# Data model

All durable state is in the master's SQLite database. Agents keep no durable
state: credentials, running jobs, terminals and buffered output live only in
memory.

## Where the data lives

| Path | Contents |
|---|---|
| `<data_dir>/master.db` | The master SQLite database |
| `<data_dir>/master-join-token` | Enrollment token, generated on first start, mode `0600` |
| `<data_dir>/known_hosts` | SSH host keys trusted by the master |
| `<data_dir>/agent.sock` | Unix socket for the master's own agent |
| `<data_dir>/gloski-agent` | Local agent binary written from the embedded build |

The database opens with WAL, `busy_timeout=5000`, foreign keys on,
`synchronous=NORMAL`, and immediate transactions. Pragmas must be written as
`_pragma=name(value)` in the DSN; older parameter forms are silently ignored.

## Migrations

`schema_migrations(version, applied_at)` records the highest applied version.

| Version | Change |
|---|---|
| 1 | Legacy `downloads`, `download_share_links` |
| 2 | Legacy `jobs` |
| 3 | Legacy `pinned_folders` |
| 4 | `devices` |
| 5 | `deployments`, `devices.deploy_id` |
| 7 | `devices.master` (version 6 was skipped) |
| 8 | `devices.machine_id` |
| 9 | `check_results` |
| 10 | `device_actions` |
| 11 | `metric_samples` |
| 12 | `agent_jobs`, `agent_job_logs` (including `last_seq`, `log_bytes`, `log_truncated` and backfill) |
| 13 | Drop legacy standalone tables (`downloads`, `download_share_links`, `jobs`, `pinned_folders`) |

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
| `stats_json` | Last full device stats snapshot marshalled as JSON |
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
(install or stop failed).

### `check_results`

One row per run of a check or native probe. A check that does not apply to a
device records `skipped`, and a check that has never run has no row at all.
Neither looks like a failure.

| Column | Meaning |
|---|---|
| `id` | Autoincrement; breaks ties within the same `ran_at` |
| `check_id`, `device_id` | The pair this result belongs to |
| `status` | `ok`, `warning`, `critical`, `skipped` or `unknown` |
| `summary` | One factual summary line |
| `details_json` | Ordered key/value details |
| `ran_at` | When the check ran |
| `duration_ms` | How long it took |
| `interval_ns` | The check's schedule at run time; staleness is `now - ran_at > 2 × interval` |

Native probes (`GLOSKI_PROBES`) write into this table: one aggregated result per
(device, check, interval), computed from a bounded in-memory window of recent
runs. An index on `(check_id, device_id, ran_at DESC, id DESC)` keeps latest
result and short history reads fast. A daily maintenance pass prunes check
history to the newest 20 rows per `(check_id, device_id)` pair.

### `metric_samples`

One row per metric value recorded from a device heartbeat. The master writes at
most one sample per (device, metric) every five minutes.

| Column | Meaning |
|---|---|
| `device_id`, `metric` | The series this value belongs to |
| `ts` | When the master sampled it (UTC) |
| `day` | `ts` formatted as `YYYY-MM-DD` (stored explicitly for SQLite daily rollups) |
| `value` | Metric value (% for CPU, memory, swap, disks; load1; bytes/sec for network rates) |

Metric names are `cpu`, `memory`, `swap`, `load1`, `network_rx`, `network_tx`,
and `disk:<mount_point>`. The daily maintenance pass keeps full resolution for
7 days, then one sample per `(device, metric, day)` up to 365 days, and deletes
older rows.

### `device_actions`

One master-owned change to a device (currently agent start, stop, restart,
trust-host, and undo).

| Column | Meaning |
|---|---|
| `id` | UUID |
| `device_id` | Owning device |
| `kind` | `agent_start`, `agent_stop`, `agent_restart`, `trust_host`, `undo` |
| `target` | What was changed (`agent`) |
| `from_value`, `to_value` | Wanted deployment state before and after |
| `result`, `error` | `ok` or `failed`, and why it failed |
| `undoable` | `1` when the previous wanted state is known |
| `undone_at` | Set when undo succeeded; an action cannot be undone twice |
| `created_at` | When the change was made |

The device page displays the newest 20 entries.

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
| `log_bytes` | Bytes of log stored so far; capped at 10 MB per job |
| `log_truncated` | `1` once the truncation notice has been written |

### `agent_job_logs`

One row per streamed output chunk.

| Column | Meaning |
|---|---|
| `id` | Autoincrement, orders chunks |
| `job_id` | Owning job |
| `chunk` | Raw bytes, stdout or stderr |
| `stderr` | `1` for stderr and for synthetic gap/truncation notices |
| `created_at` | Timestamp |

## Status lifecycles

**Job**

```
pending ──▶ running ──▶ finished
   │           ├──────▶ failed
   │           └──────▶ stopped
   └──────────────────▶ failed       (command could not be sent)
pending/running ──────▶ abandoned    (agent no longer holds it after reconnect)
```

Terminal states are final so late results cannot overwrite them. Deleting a
device deletes its jobs and logs. Each job keeps at most 10 MB of log; the
chunk that crosses the limit is replaced by a truncation notice. A daily
maintenance pass deletes terminal jobs (and their logs) finished more than 30
days ago.

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

`downloads`, `download_share_links`, `jobs` and `pinned_folders` were from the
removed standalone mode and are dropped by migration 13.
