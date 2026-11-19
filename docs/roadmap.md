# Roadmap

What to build next, in order. Reliability details are in
[reliability-plan.md](reliability-plan.md). The rule for everything here: keep
it simple, and prefer one general building block over many special features.

## Step 1: finish reliability

1. ~~Master-owned agent lifecycle~~ (done, see reliability-plan.md).
2. **File transfers on their own connection** (design B below).
3. **Smarter heartbeats** (design C below).
4. **Clear reasons.** An action on an offline device fails at once and says
   why. Each device shows: connected or not, since when, and the reason.
5. **Job log limits.** Finished jobs deleted after 30 days; 10 MB kept per job.
6. **Protect the master.** It is the single point of failure.
   - Document running it under systemd with automatic restart (the unit file is
     in `deploy/`).
   - A nightly copy of the database and join token into `<data_dir>/backups`,
     keeping the last 7. Losing the join token locks out every agent.
7. **Break-it test.** One script: kill the master and the agent mid-job, wipe
   the install directory, check everything recovers.
8. **Try it on real devices**, including a Raspberry Pi. Fix what turns up
   before building further.

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

## Design direction (owner's decision, 2026-11-19)

A is built. B and C are next.

### A. The master owns the agent's lifecycle (built)

The agent exists only to serve the master, so the master is responsible for it.
Devices differ too much (no systemd, no cron) to rely on anything on them.

- The agent is a plain process started with `nohup`. It has no supervisor, no
  crontab line and no update code. If the master says it is incompatible, or
  anything else is badly wrong, it exits.
- The master keeps the wanted state per device (running or stopped) and runs
  one loop: for every device that should be running but is not connected, log
  in over SSH and run the install script. The script is safe to repeat: it
  replaces the binary if its hash differs from the master's and starts the
  agent if it is not running. That single action covers a crash, a reboot, a
  wiped directory, an old version and an incompatible agent.
- Rules for the loop: wait about a minute after the master starts and after a
  device drops, so reconnecting agents are left alone; back off for devices
  that stay unreachable; never touch a device the owner stopped; replace an
  outdated agent only when it has no running jobs.
- Everything depends on SSH from the master working, so each device shows the
  last SSH error in plain words (no key accepted, host key changed with a
  one-click "trust it", unreachable).
- Agents are never installed by hand; every device is deployed from the
  dashboard, so the master always has its SSH details.
- Removed: `internal/supervisor`, the `start` and `supervise` subcommands, and
  the crontab line in the install script.

### B. File transfers get their own connection

Reads and writes of file contents leave the gRPC stream. The master asks on the
stream ("send me this file", "fetch this upload") and the agent makes a plain
HTTP request to the master's agent port, which already serves HTTP, one request
per transfer with a one-time id. The bytes are streamed straight through, so
there is no piece protocol, a big transfer cannot delay terminals or commands,
and a failed transfer affects nothing else. The stream keeps commands, job
output, terminals, and small file operations (list, stat, rename, delete).

### C. Heartbeats

- A live stream means the device is online. gRPC keepalive pings already
  detect a dead connection, so no separate liveness message is needed.
- Any message from the agent counts as "seen".
- Stats are sent only while someone is looking at that device in the
  dashboard, and otherwise once a minute for the overview.
