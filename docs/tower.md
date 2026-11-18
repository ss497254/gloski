# Control Tower

Gloski can run in two topologies:

- **Standalone** — one `gloski` binary serves both the dashboard and the local API. This is the default and is unchanged.
- **Control tower** — one `gloski` master serves the dashboard and registry, while a `gloski-agent` on each device exposes that device's local API. The master proxies requests to agents over your tailnet.

```
Browser ──HTTPS/WSS──▶ gloski master (tailscale serve)
                              │
                    ┌─────────┼─────────┐
                    ▼         ▼         ▼
              gloski-agent gloski-agent gloski-agent
                (laptop)     (server)     (vm)
```

The browser only talks to the master. Device tokens stay in the master's database and are injected into proxied requests, so the browser never holds an agent credential.

## Modes

| Mode | Binary | Purpose |
|------|--------|---------|
| `standalone` | `gloski` | Original single-node dashboard and API |
| `master` | `gloski` | Serves the SPA, enrolls agents, proxies device requests |
| `agent` | `gloski-agent` | Reports device stats and serves its local API to the master |

`gloski-agent` always runs in agent mode. `gloski` accepts `--mode master`; without it the binary stays standalone.

## Quick Start

### 1. Build

```bash
make build
```

This produces `bin/gloski`, `bin/gloski-agent`, and `web/dist`. The master serves
`web/dist` itself; there is no separate static-host or Netlify deployment step.

### 2. Start the master

On the machine that will be the control tower:

```bash
GLOSKI_MODE=master \
GLOSKI_HOST=127.0.0.1 \
GLOSKI_PORT=8080 \
GLOSKI_DATA_DIR="$HOME/.gloski/master" \
GLOSKI_MASTER_WEB_DIR=/path/to/gloski/web/dist \
bin/gloski
```

The master creates `<data_dir>/master.db` and, on first start, an enrollment
token at `<data_dir>/master-join-token`:

```bash
cat ~/.gloski/master/master-join-token
```

Set `GLOSKI_MASTER_JOIN_TOKEN` (or `master.join_token`) to choose your own token
instead. Treat it as a secret: anyone who has it can enroll a device.

### 3. Enroll each device

Copy `gloski-agent` to the device, then run:

```bash
GLOSKI_AGENT_MASTER_URL=http://<master-tailnet-host>:8080 \
GLOSKI_AGENT_JOIN_TOKEN=<token-from-master> \
GLOSKI_AGENT_NAME=my-device \
GLOSKI_HOST=0.0.0.0 \
GLOSKI_PORT=8080 \
bin/gloski-agent
```

The agent:

1. Registers with the master and receives a per-device token.
2. Saves its credentials to `<data_dir>/agent.json` (mode `0600`).
3. Starts its local API and sends a heartbeat every 10 seconds.

When the agent binds to `0.0.0.0` or `::`, it advertises its Tailscale address
(preferred) or another interface address automatically. This is required when
the master sits behind `tailscale serve`, because the master otherwise sees the
registration coming from loopback. If the agent has several interfaces, set
`GLOSKI_AGENT_ADVERTISE_URL` explicitly to the URL the master should use:

```bash
GLOSKI_AGENT_ADVERTISE_URL="http://$(tailscale ip -4):8080"
```

The agent API requires the issued device token even when it listens on
`0.0.0.0`. Prefer binding to the tailnet address when possible.

### 4. Put the master behind Tailscale

Run the master on loopback and let `tailscale serve` terminate HTTPS:

```bash
tailscale up
tailscale serve --bg http://127.0.0.1:8080
tailscale serve status
```

Open the HTTPS URL printed by `tailscale serve`. The SPA detects the master and
shows **Fleet** in the sidebar. Agents use the same URL as
`GLOSKI_AGENT_MASTER_URL`. MagicDNS names are usually simplest:

```bash
GLOSKI_AGENT_MASTER_URL=https://gloski.tailnet-name.ts.net
```

If you terminate TLS elsewhere, point the agent at that URL instead. Plain HTTP
over the tailnet also works for a private setup.

Example user systemd units live in `deploy/gloski-master.service` and
`deploy/gloski-agent.service`; update the paths and user, then install them with
`systemctl --user enable --now <unit>`.

## What the Fleet UI Shows

- One btop-inspired card per device with CPU, memory, network, load, and
  uptime, updated from the agent's stats WebSocket through the master.
- Device detail with per-core CPU, memory breakdown, network rates, disks, and
  the top processes polled every 2.5 seconds.
- **Files & tools** and **Jobs** links open the existing feature pages against
  that device. The master rewrites those API calls through
  `/api/devices/<id>/...`, so the normal file, terminal, and job workflows keep
  working.
- Live fleet updates over `/api/fleet/ws`: enrolments, heartbeats, and removals
  appear without refreshing.

Standalone behavior is unchanged. When the app does not detect a master, the
Fleet page explains how to start one and the original server list remains the
primary workflow.

## Configuration

### Master environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_MODE` | `standalone` | Set to `master` |
| `GLOSKI_HOST` | `127.0.0.1` | Bind address |
| `GLOSKI_PORT` | `8080` | Bind port |
| `GLOSKI_DATA_DIR` | `~/.gloski/data` | Database and join token directory |
| `GLOSKI_MASTER_WEB_DIR` | `web/dist` | Built SPA directory |
| `GLOSKI_MASTER_JOIN_TOKEN` | generated | Enrollment token |
| `GLOSKI_MASTER_DEVICE_TIMEOUT` | `45` | Seconds without heartbeat before a device is offline |
| `GLOSKI_MASTER_HEARTBEAT_INTERVAL` | `10` | Heartbeat interval advertised to agents |
| `GLOSKI_API_KEY` | (none) | Optional user auth for `/api/devices` and the proxy |

### Agent environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_AGENT_MASTER_URL` | required | Master base URL |
| `GLOSKI_AGENT_JOIN_TOKEN` | (none) | Enrollment token; only needed before first enrollment |
| `GLOSKI_AGENT_NAME` | hostname | Display name and stable registry key |
| `GLOSKI_AGENT_ADVERTISE_URL` | derived | URL the master should use to reach this agent |
| `GLOSKI_AGENT_HEARTBEAT_INTERVAL` | `10` | Heartbeat interval in seconds |
| `GLOSKI_AGENT_STATE_FILE` | `<data_dir>/agent.json` | Credential file |
| `GLOSKI_HOST` | `127.0.0.1` | Agent API bind address; use the tailnet address or `0.0.0.0` so the master can reach it |
| `GLOSKI_PORT` | `8080` | Agent API port |
| `GLOSKI_DATA_DIR` | `~/.gloski/data` | Data directory |
| `GLOSKI_ALLOWED_PATHS` | all | Filesystem roots the master may browse through this agent |
| `GLOSKI_SHELL` | `$SHELL` | Shell used by the terminal |

### JSON configuration

The same settings can be placed in a config file and passed with `-c`:

```json
{
  "mode": "master",
  "host": "127.0.0.1",
  "port": 8080,
  "data_dir": "/home/me/.gloski/master",
  "master": {
    "web_dir": "/opt/gloski/web/dist",
    "join_token": "",
    "device_timeout": 45,
    "heartbeat_interval": 10
  }
}
```

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "data_dir": "/home/me/.gloski/agent",
  "agent": {
    "master_url": "https://gloski.tailnet-name.ts.net",
    "join_token": "",
    "name": "media-server",
    "advertise_url": "",
    "heartbeat_interval": 10
  }
}
```

For an agent, `mode` is forced to `agent` by the `gloski-agent` binary.

## Re-enrollment and removal

- Re-enrolling a device with the same `name` rotates its device token in place.
- If the master database is recreated, delete the agent's `agent.json` and
  restart it with the new join token to enroll again.
- Removing a device with the trash button deletes its registry entry. The agent
  keeps running and will need to re-enroll to appear again.

## Control-plane API

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `GET` | `/api/tower/info` | none | Detection and fleet summary |
| `GET` | `/api/devices` | optional user auth | List devices and cached stats |
| `GET` | `/api/devices/{id}` | optional user auth | One device |
| `DELETE` | `/api/devices/{id}` | optional user auth | Remove a device |
| `*` | `/api/devices/{id}/...` | optional user auth | Proxy to the agent, with the device token injected |
| `GET` | `/api/fleet/ws` | none | Live device snapshots |
| `POST` | `/api/agents/register` | join token | Agent enrollment |
| `POST` | `/api/agents/heartbeat` | device token | Liveness and stats |

`/api/tower/info` returns the master name, version, device counts, and heartbeat
interval, which is how the SPA decides whether to show fleet features.

## Security notes

- Keep the master on loopback and expose it through `tailscale serve`, or use
  Tailscale ACLs to limit who can reach it.
- The join token enrolls devices. Rotate it by changing
  `GLOSKI_MASTER_JOIN_TOKEN` and removing stale devices.
- Device tokens are private to the master. The proxy removes dashboard
  credentials before forwarding and injects the appropriate device token.
- The agent API is not intended to be exposed to the public internet. Its
  device token is strong, but the master is the only intended client.
- Setting `GLOSKI_API_KEY` protects the fleet and proxy routes. The SPA does not
  currently provide a login screen, so a master that uses user auth should also
  be protected by the tailnet or fronted by an auth proxy.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| Device never appears | Agent logs, `GLOSKI_AGENT_MASTER_URL`, join token, and port reachability from the master |
| Device shows offline | Agent process, heartbeat logs, and `GLOSKI_MASTER_DEVICE_TIMEOUT` |
| Device online but stats empty | Agent `/api/health`, then `curl -H "X-API-Key: <device-token>" http://<agent>:8080/api/system/stats` from the master |
| Proxy returns 502 | The stored device URL is stale; re-enroll or set `GLOSKI_AGENT_ADVERTISE_URL` |
| SPA shows 404 | Build the web app and set `GLOSKI_MASTER_WEB_DIR` to the directory containing `index.html` |
| Heartbeats fail after recreating master | Remove the agent state file and restart with a valid join token |

Logs from both sides use the standard Gloski logger; set `GLOSKI_LOG_LEVEL=debug`
for more detail.
