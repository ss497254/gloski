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

The job tables are **not** migrations. `agents.NewJobStore` creates
`agent_jobs` and `agent_job_logs` with `CREATE TABLE IF NOT EXISTS` and adds
`agent_jobs.last_seq` with a runtime `ALTER TABLE`. A schema change to jobs must
be added there as well as to the startup path; a new migration alone will not
create the tables on an empty database.

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

### `agent_job_logs`

One row per streamed output chunk.

| Column | Meaning |
|---|---|
| `id` | Autoincrement, orders chunks |
| `job_id` | Owning job |
| `chunk` | Raw bytes, stdout or stderr |
| `stderr` | `1` for stderr and for the synthetic gap notice |
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
Finished jobs are kept until deleted; there is no age or size limit yet.

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
