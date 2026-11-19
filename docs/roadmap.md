# Roadmap

What to build next, in order. Reliability details are in
[reliability-plan.md](reliability-plan.md). The rule for everything here: keep
it simple, and prefer one general building block over many special features.

## Step 1: finish reliability

1. **Restart and Reinstall buttons** per device. Restart makes the agent exit so
   its supervisor starts it fresh. Reinstall runs the install script over SSH,
   or shows the one-line command when the master has no SSH details.
2. **Self-update by reinstall.** An agent whose binary differs from the master's
   fetches and runs the same install script once its jobs have finished. No
   separate update code.
3. **Clear reasons.** An action on an offline device fails at once and says why.
   Each device shows: connected or not, since when, the reason, agent version,
   and whether it survives a reboot (cron or not).
4. **Job log limits.** Finished jobs deleted after 30 days; 10 MB kept per job.
5. **Protect the master.** It is the single point of failure.
   - Document running it under systemd with automatic restart (the unit file is
     in `deploy/`).
   - A nightly copy of the database and join token into `<data_dir>/backups`,
     keeping the last 7. Losing the join token locks out every agent.
6. **Break-it test.** One script: kill the master and the agent mid-job, wipe
   the install directory, check everything recovers.
7. **Try it on real devices**, including a Raspberry Pi, with a real cron daemon
   and a real SSH install. Fix what turns up before building further.

## Step 2: three building blocks

Most homelab features are "run a command somewhere, on a schedule, and look at
the result". Build that once instead of building each feature separately.

1. **Saved commands.** Name a command and keep it: "update packages", "prune
   docker", "reboot". Run it with one click.
2. **Run on many devices.** Pick devices or a tag (`pi`, `docker`, `nas`); get
   one result per device, never a combined pass or fail.
3. **Schedules.** Run a saved command on chosen devices at a time or interval.
   The master keeps the schedule; a device that is offline at that moment is
   skipped and shown as skipped.

## Step 3: checks

A check is a scheduled saved command whose exit code means healthy or not, with
its last line shown as the detail. One feature, and it gives a green or red
light per device for anything a shell command can answer:

- disk nearly full
- updates pending, or a reboot required
- a systemd service failed
- a container stopped or restarting
- last backup too old
- SMART, ZFS or RAID reporting a problem
- a certificate about to expire
- a URL on the LAN not answering
- CPU temperature too high (Raspberry Pi)

Ship a handful of these ready-made. The dashboard's home page becomes: every
device, every check, red ones first. No alerts are sent (not wanted for now);
the page is where you look.

## Step 4: views worth their own screen

Only where a list of command output is not good enough:

- **Containers:** list, start, stop, restart, logs.
- **Services:** systemd units, failed ones first, restart, recent log lines.
- **Power:** reboot and shut down a device; wake a sleeping one with
  Wake-on-LAN sent from another device on the same network.
- **History:** keep metrics for days instead of minutes, so you can see when a
  disk started filling or a machine started swapping.

## Ideas parked

Copy a file from one device to another; notes per device (location, hardware,
what runs on it); installable phone app view; resumable terminals; alerts.
Add any of these when the need is real.

## Proposed design changes (not decided yet)

Two changes to how agents work, raised on 2026-11-19. If accepted they go into
step 1 ahead of everything else, because they remove code rather than add it.

### A. The agent stops managing its own lifecycle

Today the agent binary contains a supervisor (restart loop, lock, pid file, log
cap) and the install adds a crontab line. Proposal: hand all of that to
systemd, which already does each of those things, and make the agent a plain
process that connects, works, and exits non-zero whenever something is wrong.

- Install writes one systemd unit (a system unit as root, a user unit with
  lingering otherwise) with `Restart=always`. systemd then provides restart on
  crash, start at boot, one instance, log limits, a memory ceiling
  (`MemoryMax`) and a clean stop.
- The unit runs a short script before the agent starts: ask the master for the
  current build's hash and download it if it differs. So "update" is just
  "restart", the agent has no update code, and the dashboard's Restart button
  also updates.
- The agent moves out of `/tmp` to a directory that survives reboots. The
  uninstall script removes the unit and that directory.
- Devices without systemd (some containers and NAS boxes) get the same script
  run in a loop from one crontab line.
- Removed: `internal/supervisor`, the `start` and `supervise` subcommands, the
  lock, pid and log handling, and the reinstall-from-cron logic.

### B. More than one stream per connection

Today everything shares one gRPC stream: commands, heartbeats, job output,
terminal bytes and file pieces. It works, but a large file transfer or a flood
of job output sits in front of terminal keystrokes and heartbeats, and the code
has to do by hand what HTTP/2 already does (request ids, pending maps, bounded
buffers).

Proposal: keep the single connection, dialled by the agent as now, but open
separate streams on it:

- one **control** stream: register, heartbeats, small commands, job events;
- one stream **per terminal**;
- one stream **per file transfer**.

The master asks on the control stream ("open terminal 42") and the agent opens
the matching stream back to the master. Each stream has its own flow control,
so a slow download cannot delay a terminal, and backpressure comes for free.
Jobs stay on the control stream because their replay logic already works.

Not proposed: having the master dial agents. It would make the code plain
request and response, but every agent would need an open port, and the
agent-dials model already works everywhere.
