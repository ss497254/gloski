# Reliability Plan

Goal: when you want to do something on a device, it works, and when it cannot,
you are told exactly why. Security hardening is out of scope for now.

Decisions already made:

- No SSH fallback for actions. If the agent is down, the master revives it; it
  does not run your task over SSH instead.
- One install script and one uninstall script. Install handles architecture.
  Uninstall removes the binary and every leftover.
- Scripts and agent binaries are served on the agent port (9090), next to gRPC.
- An agent that cannot reach the master for 30 minutes stops itself once no job
  is running. Files stay in place.
- An agent whose build differs from the master's updates itself.
- Agents stay in `/tmp` for now; systemd comes later. No offline alerts.

## Failure scenarios and intended behaviour

### The agent is not running

| # | Scenario | Behaviour |
|---|---|---|
| A1 | Agent process crashes | Restart loop on the device restarts it within seconds |
| A2 | Device reboots, `/tmp` is wiped | Master sees it offline, reinstalls over SSH and starts it |
| A3 | A `/tmp` cleaner deletes the binary while it runs | Same as A2 at the next restart |
| A4 | `/tmp` is `noexec`, or the binary is the wrong architecture | Install script runs the binary once, fails with a clear message, and accepts another directory |
| A5 | Two agents on one device (reinstall without stop, orphan after a master crash) | Install stops the old one first; an agent refuses to start if another holds the lock; the master's local agent dies with its parent |
| A6 | Agent is alive but stuck | Keepalive drops the stream; an internal watchdog exits non-zero so the loop restarts it |
| A7 | A heavy job gets the agent OOM-killed | Loop restarts it; the lost job is marked abandoned with that reason |
| A8 | Agent log fills the disk | Log is size-capped |
| A9 | Master was down longer than 30 minutes, agents stopped themselves | SSH-managed agents are revived when the master returns. **Open: curl-installed agents have nobody to revive them (see Open questions)** |

### The agent runs but cannot connect

| # | Scenario | Behaviour |
|---|---|---|
| N1 | Master restarts | Agents reconnect within seconds; actions sent in that window wait briefly for the device instead of failing |
| N2 | Master's Tailscale name does not resolve on the device | Agent is given the name and the IPs, and tries each |
| N3 | Tailscale comes up after the master starts | Already handled: the master retries binding |
| N4 | Half-dead connection | Already handled: keepalives |
| N5 | Join token changed (data directory lost or master moved) | Agent logs the exact reason; the health panel shows "rejected: bad token" instead of plain offline |
| N6 | Master and agent speak different protocol versions | Checked at connect; the agent updates itself, or reports the mismatch if it cannot |

### The master cannot revive the agent

| # | Scenario | Behaviour |
|---|---|---|
| R1 | SSH host key changed after a reinstall | Revive stops, health panel says so and offers "trust new key" |
| R2 | Master has no SSH key (no ssh-agent under systemd) | Reported at master start and in the doctor check |
| R3 | You stopped the agent on purpose | Recorded as desired state; the loop leaves it alone |
| R4 | Device is gone for days | Attempts back off to a slow steady rate and never block other devices |
| R5 | Master just started and every agent is briefly offline | Grace period before the first revive attempt |
| R6 | Device was installed with curl, so the master has no SSH record | Shown as "unmanaged"; only the on-device restart loop protects it |

### The action itself goes wrong

| # | Scenario | Behaviour |
|---|---|---|
| T1 | Job sent to an offline device | Queued, runs when the device returns, expires after 1 hour |
| T2 | Job hangs forever | Optional per-job timeout; long-running jobs are flagged |
| T3 | Job output is enormous, or logs pile up for months | Per-job cap and automatic pruning so the master's disk cannot fill |
| T4 | Terminal drops on a network blip or tab sleep and the running command is lost | Session stays alive on the device for a few minutes and the browser reattaches with recent output replayed |
| T5 | File saved from the editor after it changed on disk | Save is refused with "changed since you opened it" unless forced |
| T6 | Upload interrupted | Stale partial files are cleaned up |
| T7 | Permission denied | Error names the user the agent runs as; the health panel shows it |
| T8 | Command run on many devices, some fail | Per-device result, never one combined pass or fail |
| T9 | Dashboard shows stale state after its own connection drops | Fleet socket reconnects and refetches; a banner shows while disconnected |

### Updating

| # | Scenario | Behaviour |
|---|---|---|
| U1 | Master rebuilt, agents are old | Agents compare binary hashes at connect and update themselves |
| U2 | Download is cut off or corrupt | Hash is verified before the swap; the old binary keeps running |
| U3 | Update needed while jobs run | Waits for jobs to finish |
| U4 | Master's agent binaries are stale relative to the master | Hash comparison means at most one update per build, never a loop |

## Build order

Each phase is committed to `main` and tested against a live master and agents.

1. **Install and uninstall scripts, per-architecture binaries.**
   `make build` produces agents for amd64, arm64 and arm. The master serves
   `install-agent.sh`, `uninstall-agent.sh` and the binaries on port 9090. The
   dashboard's SSH deploy uses the same script. New layout: everything under one
   directory (`/tmp/gloski`) so uninstall is a single removal. Covers A1, A4, A5,
   A8.
2. **Revive loop.** Master restarts or reinstalls SSH-managed agents that stay
   offline. Covers A2, A3, A9 (managed), R3, R4, R5.
3. **Self-stop and self-update.** 30-minute rule, hash-based update, protocol
   version check, local agent dies with the master. Covers A9, N6, U1 to U4.
4. **Definite outcomes.** Queued jobs with expiry, short wait for a
   reconnecting device, clear failure reasons, log cap and pruning, stale upload
   cleanup. Covers N1, T1, T3, T6, T7.
5. **Health panel and doctor.** Per-device: why offline, since when, reconnect
   count, agent user and version, last revive error, agent log tail, and an
   on-demand check (SSH, architecture, directory executable, disk, master
   reachable). Covers N5, R1, R2, R6.
6. **Break-it test.** A script that kills the master, the agent and the
   connection mid-job and checks recovery. Run before each release.

Later, on request: resumable terminals (T4), job timeouts (T2), editor conflict
check (T5), multi-device runs (T8), dashboard reconnect banner (T9), agent
watchdog (A6), multiple master addresses (N2).

## Open questions

1. **Curl-installed agents and the 30-minute rule.** If they stop themselves,
   nothing brings them back. Options: the rule applies only to SSH-managed
   agents; or curl-installed agents never stop; or they stop and you reinstall.
2. **Job log pruning.** Automatic deletion needs a number: suggested 30 days or
   10 MB per job, whichever comes first.
3. **Testing the revive loop.** This machine has no SSH server, so phase 2 can
   only be unit-tested here unless one is installed locally for the test.
