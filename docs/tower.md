# Control Tower

Gloski has two topologies:

- **Standalone** — one `gloski` binary serves the dashboard and local API. This is the default.
- **Control tower** — one `gloski` master serves the dashboard, discovers devices on the tailnet, and runs `gloski-agent` slave services on those devices.

The browser only talks to the master. Device tokens stay in the master database and are injected into proxied requests.

## How deployment works

The master does not shell out to `tailscale` or `ssh` commands. It:

1. Reads tailnet peers from **tailscaled's LocalAPI** over its Unix socket (no CLI).
2. Opens an SSH session in-process with `golang.org/x/crypto/ssh`.
3. Copies the `gloski-agent` binary to `/tmp/gloski-agent` on the device.
4. Writes a `0600` environment file and starts the agent with `nohup`.
5. Tracks the remote PID so the tower can stop or restart the agent later.

SSH credentials come from `SSH_AUTH_SOCK` (ssh-agent) or `~/.ssh/id_ed25519`,
`id_ecdsa`, `id_rsa`, or `GLOSKI_MASTER_SSH_KEY_FILE`. New host keys are trusted
on first use and stored in `<data_dir>/known_hosts`; later key changes are
rejected.

## Quick start

```bash
make build
```

This produces `bin/gloski`, `bin/gloski-agent`, and `web/dist`. The master
serves `web/dist` itself — there is no separate static-host deployment.

Start the master on the device that will be the tower:

```bash
GLOSKI_MODE=master \
GLOSKI_HOST=127.0.0.1 \
GLOSKI_PORT=8080 \
GLOSKI_DATA_DIR="$HOME/.gloski/master" \
GLOSKI_MASTER_WEB_DIR="$PWD/web/dist" \
bin/gloski
```

Open the dashboard. In **Fleet**, click **Deploy agent**:

- Pick an online tailnet peer (or enter a host manually).
- Set the SSH user and ports.
- Check the **Tower URL** — the agent must be able to reach it. If the dashboard
  is served through `tailscale serve`, use its HTTPS URL.

The dialog follows the deployment until the device appears in the fleet.

For a browser outside the tailnet, put the master behind `tailscale serve`:

```bash
tailscale serve --bg http://127.0.0.1:8080
```

## Fleet UI

- **Fleet** — compact table of devices with status, CPU, memory, network,
  uptime, and last heartbeat. Deploy, start, stop, and remove actions live here.
- **Device** — live btop-style detail (per-core CPU, memory, network, disks,
  processes), one-click command runs, and links to Files, Terminal, Jobs,
  Downloads, and Search. Those tools are proxied through the master, so device
  tokens are never exposed to the browser.
- Live fleet updates arrive over `/api/fleet/ws`.
- Standalone mode keeps the original server list; in master mode the sidebar
  only shows Fleet and Settings to keep one clear workflow.

## Configuration

### Master environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_MODE` | `standalone` | Set to `master` |
| `GLOSKI_HOST` | `127.0.0.1` | Bind address |
| `GLOSKI_PORT` | `8080` | Bind port |
| `GLOSKI_DATA_DIR` | `~/.gloski/data` | Database, join token, and known_hosts |
| `GLOSKI_MASTER_WEB_DIR` | `web/dist` | Built SPA directory |
| `GLOSKI_MASTER_JOIN_TOKEN` | generated | Enrollment token |
| `GLOSKI_MASTER_AGENT_BINARY` | next to master | Path to `gloski-agent` for deployments |
| `GLOSKI_MASTER_SSH_KEY_FILE` | (none) | Explicit SSH private key |
| `GLOSKI_MASTER_DEPLOY_TIMEOUT` | `120` | Max seconds for an SSH deployment |
| `GLOSKI_TAILSCALE_SOCKET` | auto | tailscaled Unix socket path |
| `GLOSKI_API_KEY` | (none) | Optional user auth for fleet/proxy routes |

### Agent environment variables

Agents are normally configured by the master, so these are only needed for a
manual launch:

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_AGENT_MASTER_URL` | required | Master base URL |
| `GLOSKI_AGENT_JOIN_TOKEN` | (none) | Enrollment token |
| `GLOSKI_AGENT_NAME` | hostname | Display name |
| `GLOSKI_AGENT_DEPLOY_ID` | (none) | Links the agent to its master deployment |
| `GLOSKI_HOST` / `GLOSKI_PORT` | `127.0.0.1:8080` | Agent API bind address |
| `GLOSKI_DATA_DIR` | `~/.gloski/data` | Agent state |

The agent asks tailscaled's LocalAPI for its own tailnet address when it binds
to `0.0.0.0`; if that API is unavailable it falls back to interface scanning.

## Control-plane API

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/tower/info` | Detection, counts, deployment/tailnet availability |
| `GET` | `/api/tailnet/peers` | Tailnet peers from LocalAPI |
| `POST` | `/api/devices/deploy` | Start an SSH deployment |
| `GET` | `/api/deployments/{id}` | Poll deployment status |
| `POST` | `/api/devices/{id}/start` | Start a managed agent |
| `POST` | `/api/devices/{id}/stop` | Stop a managed agent |
| `GET` | `/api/devices` | List devices and cached stats |
| `DELETE` | `/api/devices/{id}` | Remove a device |
| `*` | `/api/devices/{id}/...` | Proxy to the agent with device token injected |
| `GET` | `/api/fleet/ws` | Live device snapshots |

## Security notes

- The master is unauthenticated unless `GLOSKI_API_KEY` is set. Keep it on the
  tailnet and prefer `tailscale serve` or ACLs.
- The join token enrolls devices. Rotate it by changing
  `GLOSKI_MASTER_JOIN_TOKEN` and re-deploying agents.
- Host keys are trusted on first use. Protect `<data_dir>/known_hosts`.
- Removing a device from the fleet does not stop a running agent. Use **Stop
  agent** first if you want to terminate the process.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| No peers in Deploy dialog | `GLOSKI_TAILSCALE_SOCKET`; tailscaled must be running and accessible |
| "No ssh credentials found" | ssh-agent or a default key on the master |
| Deployment fails to connect | SSH user, port, and that the device accepts SSH on the tailnet |
| Agent appears offline after start | Master URL reachability and device heartbeat |
| Device online but stats empty | Agent `/api/system/stats` through `/api/devices/<id>/api` |
| Proxy returns 502 | Re-deploy to refresh the advertised agent URL |
| SPA shows 404 | Build `web/dist` and check `GLOSKI_MASTER_WEB_DIR` |
