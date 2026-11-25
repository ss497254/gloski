# Control Tower

Gloski is master/agent only. One `gloski` master serves the dashboard and keeps
all state; `gloski-agent` runs on every device, including the master host.

```
Browser ──HTTP/WS──▶ Master (dashboard + state + proxy)
                         │ gRPC (one bidi stream per device)
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
   agent on A        agent on B        agent on C
   (master host,     (run by master)   (run by master)
    child process)
```

A, B, and C all appear in Fleet. The master device is badged **Master**. Every
dashboard action goes to the master, which forwards it to the right agent.

## Communication

- **Master ↔ agent:** gRPC only, one bidirectional stream.
  - Local agent: gRPC over `<data_dir>/agent.sock`.
  - Remote agents: gRPC h2c on port `9090`, bound to this machine's Tailscale
    addresses only. If Tailscale is not up yet the master retries every 15
    seconds.
  - The agent sends registration, heartbeats/stats, job output, file/process
    results and native check results; the master sends jobs, file requests,
    terminal I/O, probe requests and shutdown. Protocol version is 4.
- **Browser → master:** plain HTTP/WS. The master translates those calls into
  gRPC commands. No HTTP agent API exists anymore.

## Ephemeral agents

Agents keep nothing on disk. There is no local database and no job log files on
devices. Credentials and running jobs live in memory only. On restart the agent
re-registers with the master and receives a fresh device token.

Jobs belong to the agent process, not to the connection. If the stream drops or
the master restarts, jobs keep running and the agent buffers up to 4 MB of
output per job. On reconnect it reports the jobs it holds and resends whatever
the master is missing; if more than the buffer was produced, the log notes the
gap. A job is marked abandoned only when the agent comes back without it,
which means the agent process itself was restarted.

## Agent lifecycle

The master owns every remote agent's lifecycle. An agent does not restart,
update or reinstall itself; it is a plain process that connects, does work,
and exits if the master refuses it as incompatible.

**Deploy agent** in the dashboard records the device's SSH details and runs
the install script over SSH. The script is safe to repeat: it copies the right
build for the device's CPU (`amd64`, `arm64` or 32-bit `arm`, all embedded in
the master), preserves existing operator settings while updating master-owned
fields, verifies size and SHA-256, stages and swaps the binary atomically with
rollback on failure, and starts it if it is not running. Everything lives in
`/tmp/gloski`: the binary, `agent.json`, `agent.log` (rolled over at 5 MB) and
`agent.pid`.

From then on a loop in the master looks at every deployed device every 15
seconds and runs that same script when:

- the agent has been disconnected for 45 seconds. This covers a crash, a
  reboot, a wiped `/tmp` and an agent that exited as incompatible;
- the agent is connected but is not the master's build, and has no running
  jobs. This is how agents are updated after the master is rebuilt.

Failed attempts back off from 1 to 15 minutes. The reason for the last failure
is stored on the deployment and shown on the device, for example "cannot reach
host over SSH", "refused SSH login" or "SSH host key changed"; the last one has
a **Trust new key** button.

- **Stop** tells the agent to exit and marks the device stopped, so the loop
  leaves it alone until you start it.
- **Start** and **Restart** run the install script now and force a restart.
- **Remove device** makes a connected agent delete its own install; for an
  offline device the master removes it over SSH.

On a tailnet with Tailscale SSH enabled, the master needs **no SSH key**: the
SSH client offers "none" authentication first and Tailscale authenticates the
connection by tailnet identity. That is the rescue path when an agent is down
and the master has no key configured. If Tailscale SSH is off on a device, keys
are still supported: `GLOSKI_MASTER_SSH_KEY_FILE`, `~/.ssh/id_ed25519`,
`id_ecdsa`, `id_rsa`, or a running `ssh-agent`.

## Master-owned state

The master database stores devices, deployments, jobs, streamed job logs, check
results, metric samples, and device actions. Cached device stats arrive through
heartbeats. Job output is streamed from the agent over gRPC and persisted by
the master, so it survives agent restarts.

## Local agent lifecycle

Master starts `gloski-agent` as a child process (no shell). If it exits
unexpectedly, master logs it and restarts it with backoff; the dashboard stays
up. On master shutdown, the child is terminated first. Set
`GLOSKI_MASTER_LOCAL_AGENT=false` to disable it.

## Quick start

```bash
make build

GLOSKI_MODE=master \
GLOSKI_HOST=0.0.0.0 \
GLOSKI_PORT=8080 \
GLOSKI_MASTER_WEB_DIR="$PWD/web/dist" \
bin/gloski
```

Open the dashboard, choose **Deploy agent**, pick a tailnet peer, and provide
the SSH user. No key is needed when the peer offers Tailscale SSH; otherwise
keys come from `SSH_AUTH_SOCK` or `~/.ssh/id_*`.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GLOSKI_MODE` | `master` | `gloski` runs as master by default |
| `GLOSKI_HOST` / `GLOSKI_PORT` | `127.0.0.1:8080` | Dashboard HTTP listener |
| `GLOSKI_DATA_DIR` | `~/.gloski/data` | DB, tokens, agent socket, `known_hosts` |
| `GLOSKI_MASTER_WEB_DIR` | `web/dist` | Built SPA |
| `GLOSKI_MASTER_JOIN_TOKEN` | generated | Agent enrollment token |
| `GLOSKI_MASTER_AGENT_BINARY` | next to master | Agent binary used only when the master was built without embedded agents |
| `GLOSKI_MASTER_SSH_KEY_FILE` | (none) | Optional SSH key when Tailscale SSH is not enabled on a device |
| `GLOSKI_MASTER_AGENT_GRPC_HOST` | Tailscale IPs | Remote agent gRPC bind; set to override |
| `GLOSKI_MASTER_AGENT_GRPC_PORT` | `9090` | Remote agent gRPC port |
| `GLOSKI_MASTER_AGENT_GRPC_URL` | auto | Override the agent-facing gRPC URL |
| `GLOSKI_TAILSCALE_SOCKET` | auto | tailscaled LocalAPI socket |
| `GLOSKI_API_KEY` | (none) | Optional dashboard user auth |
| `GLOSKI_ALLOWED_ORIGINS` | (none) | Comma-separated extra origins allowed to call the API; `*` disables the check |
| `GLOSKI_PROBES` | (none) | JSON array of native probes: `kind` (`tcp`, `http`, `dns`, `tls`), `target`, optional `accepted`, `interval` (default `5m`), `timeout` (default `5s`, capped at `9s`), `skip_verify` |

## Logs and health

- Master log: wherever stdout of `gloski` is redirected. It records local agent
  start/stop, gRPC session state, health checks, and forced restarts.
- Local agent log: `<data_dir>/local-agent.log`.
- Remote agent log: `/tmp/gloski/agent.log` on the device.
- Master log lines starting `lifecycle:` say when and why the master installed
  an agent, and why an attempt failed.

The master checks the local agent's gRPC session every 5 seconds. If it is not
connected for 20 seconds, the master kills the child process and starts a new
one with backoff. Remote agents reconnect to the master on their own.

## Security

Keep the dashboard on the tailnet and use `tailscale serve` for HTTPS, or put it
behind a reverse proxy that enforces a login. The agent gRPC port is h2c and is
only bound to Tailscale addresses unless you override the host. Host keys are
trusted on first use and stored in `<data_dir>/known_hosts`.

The master rejects writes and WebSocket handshakes whose `Origin` does not match
the request, so another website cannot drive the dashboard through your
browser. Behind a reverse proxy, either forward the original `Host` header (or
set `X-Forwarded-Host`), or list the public origin in `GLOSKI_ALLOWED_ORIGINS`.
`nginx` does not forward `Host` by default: add `proxy_set_header Host $host;`.
JSON endpoints require `Content-Type: application/json`.
