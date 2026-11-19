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
