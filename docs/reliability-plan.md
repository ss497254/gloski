# Reliability Plan

Goal: when you want to do something on a device, it works, and when it cannot,
you are told why. Security hardening is out of scope for now.

**Keep it simple.** The agent is stateless, so there are only two repairs:
restart it, or reinstall it. Every failure is handled by one of those two, done
automatically where that is cheap and by one button where it is not. Do not add
a separate mechanism per failure.

## Design

1. **Each device keeps its own agent alive.** A supervisor inside the agent
   binary restarts it after a crash. A crontab line starts it every 5 minutes if
   it is not running, and reinstalls it from the master if its files are gone.
2. **The master is the source of truth.** It serves one install script, one
   uninstall script and the agent builds (embedded in the master binary) on
   port 9090. An agent whose binary differs from the master's replaces itself.
3. **Every action ends in a definite outcome.** It runs, or it fails with a
   reason a person can act on.

## What handles what

| Problem | Handled by |
|---|---|
| Agent crashes, or is killed for using too much memory | Supervisor restarts it (done) |
| Device reboots, or `/tmp` is wiped | Crontab line reinstalls it (done) |
| Two agents on one device | Lock; the second exits (done) |
| Wrong architecture, or `noexec` directory | Install fails with a clear message (done) |
| Agent log fills the disk | Supervisor caps it (done) |
| Master down, however long | Agents keep retrying and reconnect (done) |
| Master restarts mid-job | Job keeps running, output is replayed (done) |
| Agent is frozen or otherwise misbehaving | **Restart** button; if it ignores that, **Reinstall**, which force-kills |
| Agent is older than the master | Runs the install script again by itself, after running jobs finish |
| Job sent to an offline device | Fails at once with "offline since …", no queue |
| Job logs piling up | Deleted after 30 days; 10 MB kept per job |
| "Offline" with no explanation | Device shows why: never connected, wrong token, last seen, last restart reason |

Deliberately left out, because each adds a mechanism for a rare case: master-side
SSH revive loop, agent self-removal timers, remembering deleted devices, memory
and heartbeat watchdogs, job queueing, per-job timeouts. If one of these cases
actually bites, add it then.

A device without `crontab` restarts after a crash but not after a reboot; the
fix is to press Reinstall or paste the install command again.

## Build order

See [roadmap.md](roadmap.md), step 1. Phase 1 (self-healing install) is done.

## Status

- Phase 1: done. Verified live: install by curl, agent crash, supervisor killed,
  double start, repeated install, wiped directory, stop and remove from the API,
  wrong-architecture message. Not yet exercised: the SSH install path and a real
  cron daemon (the crontab line was run by hand).
- Next: roadmap step 1.

## Testing notes

The development machine has `crontab` but no cron daemon and no SSH server. Run
the crontab line by hand to test self-heal. Test the SSH path against an
in-process SSH server written with `golang.org/x/crypto/ssh`.
