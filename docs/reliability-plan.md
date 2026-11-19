# Reliability Plan

Goal: when you want to do something on a device, it works, and when it cannot,
you are told exactly why. Security hardening is out of scope for now.

## The design in three rules

The first version of this plan had the master chase dead agents over SSH. That
made the master's SSH access a second thing that can break (changed host keys,
no ssh-agent, devices installed by hand that the master cannot reach). This
version turns it around.

**1. Each device keeps its own agent alive.**

- The agent binary supervises itself: a small parent process starts the real
  agent, restarts it within seconds if it dies, caps the log size, and holds a
  lock so two agents can never run on one device.
- Install adds one crontab line. Every 5 minutes, and at boot, it starts the
  agent if it is not running, and if the files are gone (reboot wiped `/tmp`) it
  downloads the install script from the master and runs it again.
- So a device heals itself after a crash, a reboot or a wiped `/tmp`, and it
  does not matter whether it was installed from the dashboard or by `curl`.
- The master's SSH is a convenience and a backstop: it runs the same install
  script for the first install, and re-runs it for a device that has saved SSH
  details and has been offline for 10 minutes (this covers devices without
  cron).

**2. The master is the only source of truth, and agents converge on it.**

- One install script and one uninstall script, served by the master on port
  9090 next to gRPC. Install is safe to run any number of times.
- Agent builds for amd64, arm64 and arm are embedded in the master binary, so
  the master always serves agents that match itself.
- At connect, the agent reports the hash of its binary. If it differs from the
  master's build, the agent downloads the new one, verifies it and restarts,
  waiting for running jobs to finish first.
- An agent never stops just because the master is unreachable; it keeps
  retrying, which costs nothing. It removes itself completely (process, files,
  crontab line) in two cases: the master says the device was deleted, or the
  master has been unreachable for 7 days in a row.

**3. Every action ends in a definite outcome.**

- An action sent while a device is reconnecting waits a few seconds for it.
- A job sent to an offline device is queued, runs when the device returns, and
  expires after 1 hour. It can be cancelled while queued.
- Every failure carries a reason a person can act on: "offline since 14:02,
  last error: join token rejected", not "offline".
- Finished jobs are deleted after 30 days. Each job keeps at most its most
  recent 10 MB of output.

### What changed from the earlier decisions, and why

- **"Stop after 30 minutes" became "remove itself after 7 days".** A stopped
  agent would be restarted by its own crontab line 5 minutes later, so stopping
  has no meaning any more. The purpose of that rule was to avoid leftover agents
  when a master is gone for good; a full self-removal after 7 days does that,
  and a short master outage no longer takes the fleet down with it.
- **The revive loop moved from the master to the device.** The master-side
  version stays only as a backstop.
- Unchanged: no SSH fallback for actions, agents stay in `/tmp`, no offline
  alerts, systemd later.

## Failure scenarios

| # | Scenario | What handles it |
|---|---|---|
| A1 | Agent process crashes | Supervisor restarts it in seconds |
| A2 | Device reboots, `/tmp` is wiped | Crontab line reinstalls from the master |
| A3 | A `/tmp` cleaner deletes the binary | Same as A2 within 5 minutes |
| A4 | `/tmp` is `noexec`, or wrong architecture | Install runs the binary once, fails with a clear message, accepts another directory |
| A5 | Two agents on one device | Lock held by the supervisor; a second start exits quietly |
| A6 | Agent alive but its connection is stuck | Keepalive drops the stream and the agent reconnects |
| A7 | A heavy job gets the agent killed | Supervisor restarts it and reports why the last run ended; the job is marked abandoned with that reason |
| A8 | Agent log fills the disk | Supervisor caps the log |
| A9 | Master down for hours or days | Agents keep retrying and reconnect when it returns |
| N1 | Master restarts | Agents reconnect in seconds; actions wait briefly |
| N2 | Master's name does not resolve on the device | Agent is given the name and the IPs and tries each |
| N3 | Tailscale comes up after the master | Already handled: the master retries binding |
| N4 | Half-dead connection | Already handled: keepalives |
| N5 | Join token changed | Master records the rejection; the device shows "rejected: bad token" |
| N6 | Old agent, new master | Hash check at connect, agent updates itself |
| R1 | Device without cron | Install reports it; the master's SSH backstop covers it if SSH details are saved |
| R2 | SSH backstop cannot log in (changed host key, no key) | Only matters for R1 devices; the device page shows the SSH error |
| R3 | You stopped an agent on purpose | Stop removes the crontab line and is recorded, so nothing restarts it |
| R4 | Device deleted in the dashboard while offline | When it next connects it is told to remove itself |
| T1 | Job sent to an offline device | Queued, expires after 1 hour |
| T3 | Huge output, or logs piling up for months | 10 MB per job, 30 days |
| T6 | Upload interrupted | Stale partial files cleaned up |
| T7 | Permission denied | Error names the user the agent runs as |
| U2 | Update download cut off or corrupt | Hash verified before the swap; the old binary keeps running |
| U3 | Update needed while jobs run | Waits for jobs to finish |
| U4 | Update loop | Impossible: binaries are embedded in the master, hashes match after one update |

Later, on request: resumable terminals, per-job timeouts, editor "changed on
disk" check, one command on many devices with per-device results, dashboard
reconnect banner.

## Build order

Each phase is committed to `main` and tested against a live master and agents.

1. **Self-supervising agent and install scripts.** Supervisor with lock and log
   cap; `install-agent.sh` and `uninstall-agent.sh`; builds for three
   architectures embedded in the master and served on 9090; crontab self-heal;
   dashboard deploy uses the same script; everything under one directory
   (`/tmp/gloski`). Covers A1 to A5, A8, A9, R1, R3.
2. **Convergence.** Hash-based self-update, removal on device delete, 7-day
   self-removal, master SSH backstop. Covers N6, R4, U2 to U4.
3. **Definite outcomes.** Short wait for reconnecting devices, queued jobs,
   failure reasons, log cap and pruning, stale upload cleanup. Covers N1, N5,
   T1, T3, T6, T7.
4. **Device health view.** Per device: why offline and since when, agent user,
   version, architecture, how it self-heals (cron, SSH backstop or none), why
   the agent last restarted, and the tail of its log.
5. **Break-it test.** A script that kills the master, the agent and the
   connection mid-job, wipes the install directory, and checks recovery.

## Status

- Phase 1: done. Verified live: install by curl, agent crash, supervisor killed,
  double start, repeated install, wiped directory, stop and remove from the
  API, wrong-architecture message. Not yet exercised: the SSH path (waits for
  the in-process SSH test server in phase 2) and a real cron daemon (the
  crontab line was run by hand).
- Phase 2: next.

## Testing notes

The development machine has cron, so the self-heal path can be tested for real.
It has no SSH server; the SSH backstop is tested against an in-process SSH
server written with `golang.org/x/crypto/ssh`, which the project already uses.
