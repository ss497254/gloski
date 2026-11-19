# Reliability Plan

Goal: when you want to do something on a device, it works, and when it cannot,
you are told why. Security hardening is out of scope for now.

**Keep it simple.** The agent is stateless, so there is one repair: run the
install script again. It is safe to repeat and fixes whatever is wrong. Do not
add a separate mechanism per failure.

## Design

1. **The master owns the agent lifecycle.** Devices differ too much to rely on
   systemd or cron being there. The master installs agents over SSH and a loop
   reinstalls any deployed device whose agent is disconnected or is not the
   master's build. The agent does not restart, update or reinstall itself.
2. **The master is the source of truth.** Agent builds for amd64, arm64 and
   32-bit ARM are embedded in the master, so they always match it. An agent the
   master refuses as incompatible exits and gets replaced.
3. **Every action ends in a definite outcome.** It runs, or it fails with a
   reason a person can act on.

## What handles what

| Problem | Handled by |
|---|---|
| Agent crashes or is killed | Lifecycle loop reinstalls it (done) |
| Device reboots, or `/tmp` is wiped | Lifecycle loop reinstalls it (done) |
| Agent is older than the master | Loop replaces it once it has no running jobs (done) |
| Agent is incompatible with the master | Master refuses it, agent exits, loop replaces it (done) |
| Agent is frozen or misbehaving | **Restart** button runs the install script with a forced restart (done) |
| You stopped an agent on purpose | Marked stopped; the loop leaves it alone (done) |
| Wrong architecture, or `noexec` directory | Install fails with a clear message (done) |
| Agent log fills the disk | Rolled over at 5 MB on each start (done) |
| Master down, however long | Agents keep retrying and reconnect (done) |
| Master restarts mid-job | Job keeps running, output is replayed (done) |
| SSH to the device fails | Reason shown on the device; changed host key has a "Trust new key" button (done) |
| Job sent to an offline device | Fails at once with the reason, no queue |
| A big download or upload delays commands | Contents use one HTTP request per transfer on the agent port, not the gRPC stream (done) |
| Job logs piling up | Deleted after 30 days; 10 MB kept per job |

Recovery takes about a minute: 45 seconds of grace so reconnecting agents are
left alone, plus up to 15 seconds for the next look.

Left out on purpose: anything on the device that manages the agent (supervisor,
cron, systemd), installing by hand, job queueing, per-job timeouts, watchdogs.
If one of these cases actually bites, add it then.

## Status

- Master-owned lifecycle: done and verified live against a local test SSH
  server (`server/internal/provisioning/sshtest`): deploy, agent killed and
  directory wiped (reinstalled in 76 s), master rebuilt with a new agent build
  (agent replaced automatically), stop (left alone), start, remove, unreachable
  device (clear error). Changed host key and repeat installs are covered by
  tests. Not yet exercised: an incompatible agent being refused (the check is
  in place but no old-protocol agent was run against it), a real device, and a
  32-bit ARM board.
- File transfers: done and verified live on a unix socket and over TCP:
  whole-file download, a byte-range download, and upload. A 200 MB download
  throttled to 1 MB/s did not delay a job, which finished in 10 ms.
- Next: see [roadmap.md](roadmap.md).

## Testing notes

The development machine has no SSH server. For a live test run
`go run ./internal/provisioning/sshtest/sshd` from `server/` (it listens on
127.0.0.1:2222 and runs commands locally), generate a key with `ssh-keygen`,
start the master with `GLOSKI_MASTER_SSH_KEY_FILE` pointing at it, and deploy
to host `127.0.0.1`, SSH port `2222`. The agent lands in `/tmp/gloski`.
