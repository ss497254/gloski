# HTTP API

The browser talks only to the master. Device calls
(`/api/devices/{id}/api/*`) are translated into gRPC commands on that device's
agent stream. Web routes are in [frontend.md](frontend.md); agent transport is
described in [backend.md](backend.md).

Base URL: the dashboard origin, for example `http://127.0.0.1:8080`. All routes
are registered in `server/internal/controltower/routes.go`.

## Authentication

Auth is off unless `GLOSKI_API_KEY` or a JWT public key is configured. When it
is on, every route marked **auth** below accepts either:

- `X-API-Key: <key>`, or `?api_key=<key>`
- `Authorization: Bearer <jwt>`, or `?token=<jwt>`

`GET /api/health`, `GET /api/tower/info` and the SPA are always public.
Authentication failures are rate limited to 10 per minute per client IP and
return `429` with `Retry-After: 60`.

Known gap: the dashboard client does not send the API key yet (see
`AGENTS.md`).

## Response shapes

The API uses three shapes, sometimes in the same feature. The endpoint tables
say which one applies.

| Name | Body |
|---|---|
| **raw** | The payload itself, for example `{"jobs":[...]}` |
| **wrapped** | `{"success":true,"data":<payload>}` |
| **error** | `{"success":false,"error":"plain message"}` |

JSON request bodies must use `Content-Type: application/json`; anything else
returns `415`. Writes are same-origin only (see `httpapi.SameOrigin`); set
`GLOSKI_ALLOWED_ORIGINS=*` to disable that check behind a proxy that rewrites
`Host`.

## Common errors

| Status | Meaning |
|---|---|
| `400` | Bad body, failed validation, or the operation is not allowed |
| `401` | Auth is configured and missing or invalid |
| `403` | Cross-origin write or WebSocket handshake rejected |
| `404` | Device, deployment, job or path not found |
| `413` | Upload exceeds 8 GiB |
| `415` | JSON body without `application/json` |
| `429` | Too many failed auth attempts |
| `500` | Master-side failure |
| `502` | The device's agent is not connected or did not answer |

Errors are always the **error** shape.

## Public endpoints

| Method | Path | Response |
|---|---|---|
| `GET` | `/api/health` | raw `{"status":"ok"}` |
| `GET` | `/api/tower/info` | wrapped object: `mode`, `name`, `version`, `started_at`, `devices`, `online`, `heartbeat_interval`, `port`, `deploy_available`, `agent_builds`, `tailscale_available`, `tailscale_socket` |

## Fleet and deployments

All routes are **auth**.

| Method | Path | Body / query | Response |
|---|---|---|---|
| `GET` | `/api/dashboard` | — | wrapped `devices`, `online`, `offline`, `issues`, `recent_jobs`, `generated_at` |
| `GET` | `/api/fleet/ws` | — | WebSocket, see below |
| `GET` | `/api/devices` | — | wrapped `{"devices":[Device]}` |
| `POST` | `/api/devices/deploy` | `name`, `host`, `user`, `ssh_port`, `agent_port` | wrapped `Deployment`; `400` when the master has no builds or validation fails |
| `GET` | `/api/devices/{id}` | — | wrapped `Device` |
| `DELETE` | `/api/devices/{id}` | — | wrapped `{"removed":true}`; `400` for the master device while its local agent runs |
| `POST` | `/api/devices/{id}/start` | — | wrapped `{"pid":"...","status":"running"}`; installs over SSH now |
| `POST` | `/api/devices/{id}/stop` | — | wrapped `{"stopped":<bool>,"status":"stopped"}`; `400` when offline and not deployed over SSH |
| `POST` | `/api/devices/{id}/trust-host` | — | forgets the stored SSH host key, then starts; same response as start |
| `GET` | `/api/deployments` | — | wrapped `{"deployments":[Deployment]}` |
| `GET` | `/api/deployments/{id}` | — | wrapped `Deployment` |
| `GET` | `/api/tailnet/peers` | — | wrapped `available`, `backend_state`, `self`, `peers`; `available:false` plus `error` when tailscaled is unreachable |

`Device` and `Deployment` fields are defined in `server/internal/fleet/types.go`
and `server/internal/fleet/deployment.go`. Tokens and machine IDs are never
returned (`json:"-"`).

## Device endpoints

All routes are **auth**. `{id}` is the device ID. An offline device returns
`502` for commands that need the stream; stored data such as job lists still
works.

### Stats and processes

| Method | Path | Query | Response |
|---|---|---|---|
| `GET` | `/api/devices/{id}/api/system/stats` | — | raw `Stats` from the last heartbeat |
| `GET` | `/api/devices/{id}/api/system/stats/ws` | — | WebSocket: one `Stats` JSON object every 2 seconds |
| `GET` | `/api/devices/{id}/api/system/processes` | `limit` (default 20), `sort` = `cpu` (default), `memory`, `pid` | raw `[Process]` from the agent |

### Jobs

| Method | Path | Body | Response |
|---|---|---|---|
| `GET` | `/api/devices/{id}/api/jobs` | — | raw `{"jobs":[Job]}`; newest 200 |
| `POST` | `/api/devices/{id}/api/jobs` | `command` (required), `cwd` | raw `Job`; `400` when `command` is blank |
| `GET` | `/api/devices/{id}/api/jobs/{jobId}` | — | raw `Job` |
| `GET` | `/api/devices/{id}/api/jobs/{jobId}/logs` | — | raw `{"logs":["line", ...]}` |
| `POST` | `/api/devices/{id}/api/jobs/{jobId}/stop` | — | wrapped `null` |
| `DELETE` | `/api/devices/{id}/api/jobs/{jobId}` | — | wrapped `null` |

A job ID from another device returns `404`. Job fields and statuses are in
[data-model.md](data-model.md).

### Files

| Method | Path | Body / query | Response |
|---|---|---|---|
| `GET` | `/api/devices/{id}/api/files` | `path` (default `~`) | raw `{"path":"...","entries":[FileEntry]}` |
| `DELETE` | `/api/devices/{id}/api/files` | `path` | wrapped `null` |
| `GET` | `/api/devices/{id}/api/files/read` | `path` | raw `{"content":"...","path":"..."}`; text only, 10 MB limit |
| `GET` | `/api/devices/{id}/api/files/download` | `path` | binary; supports `Range`; media is inline, everything else attachment |
| `POST` | `/api/devices/{id}/api/files/write` | `path`, `content` | wrapped `null`; JSON body limited to 32 MiB |
| `POST` | `/api/devices/{id}/api/files/mkdir` | `path` | wrapped `null`; creates parents |
| `POST` | `/api/devices/{id}/api/files/rename` | `old_path`, `new_path` | wrapped `null`; destination must not exist |
| `POST` | `/api/devices/{id}/api/files/upload` | multipart: `path` part, then `file` part | raw `{"filename":"...","size":<bytes>}`; up to 8 GiB |

`FileEntry` has `name`, `path` (tilde form under the home directory), `type`
(`file` or `directory`), `size`, `modified`, `permissions`.

Uploads stream in 1 MB `WRITE_AT` pieces and replace the target only when the
last piece lands. Only images, audio, video and PDF are served inline; SVG and
HTML are downloads on purpose.

### Terminal

`GET /api/devices/{id}/api/terminal?cwd=...&cols=80&rows=24` upgrades to a
WebSocket.

- Client to master: text frames are keystrokes. A 5-byte binary frame is a
  resize: `[0x01, cols_hi, cols_lo, rows_hi, rows_lo]`.
- Master to client: binary frames are PTY output.
- The socket fails with `502` before the upgrade when the agent is offline, and
  closes when the shell exits or the agent disappears.

## WebSockets

All WebSocket handshakes go through the same origin check as writes.

**Fleet** (`/api/fleet/ws`) sends text JSON:

1. `{"type":"snapshot","devices":[Device]}` immediately on connect.
2. `{"type":"devices","devices":[Device]}` whenever the fleet changes.

A slow client is dropped rather than stalling heartbeats. The server pings
every 25 seconds and expects a pong within 60 seconds.

**Stats** (`.../stats/ws`) sends the device's cached `Stats` object every 2
seconds while the socket is open.

## Agent transport port

The agent TCP port (`9090` by default, Tailscale addresses only unless
`GLOSKI_MASTER_AGENT_GRPC_HOST` is set) carries two protocols on one listener:
gRPC h2c for agents, and plain HTTP for `GET /healthz` → `ok`. The master's own
agent uses the Unix socket `<data_dir>/agent.sock` instead. This port is not
part of the browser API.
