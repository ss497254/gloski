# Glossary

Gloski has one word per concept. Use these terms in code, commits, docs and UI
copy. The right-hand column says where the concept lives today; the planned
renames are in [code-quality-plan.md](code-quality-plan.md).

| Term | Meaning | Where |
|---|---|---|
| **Gloski** | The project and the product. | repository |
| **master** | The control process: dashboard, browser API, SQLite, agent lifecycle. | `cmd/gloski`, `internal/controltower` |
| **agent** | The device-side process. One per device, including the master host. | `cmd/gloski-agent`, `internal/worker` |
| **device** | A machine enrolled in the fleet. | `fleet.Device` |
| **master device** | The device row with `master = 1`. Its agent reaches the master over a Unix socket. | `Device.Master` |
| **local agent** | The agent on the master host, started and supervised by the master. | `controltower/local_agent.go` |
| **remote agent** | An agent on another device, reached over h2c on the Tailnet. | `controltower/agent_transport.go` |
| **deployment** | The SSH record of how an agent was installed on a device. | `fleet.Deployment` |
| **job** | A command the master owns and one agent executes. | `agents.Job` |
| **session** | One live `Agent.Connect` gRPC stream for a device. | `agents.connection` |
| **diagnosis** | An ordered chain of facts from the master outwards, with the first broken link as the headline. | `controltower/diagnose.go` |
| **check** | A saved command whose exit code is a verdict, run on a schedule yet to be built. | `internal/checks` (result model) |
| **check result** | One run of one check on one device: verdict, summary, details, timing. | `checks.Result`, `check_results` |
| **stale** | A result older than twice its schedule. Recomputed when read, never stored as a final answer. | `checks.Result.StaleAt` |
| **skipped** | A check that does not apply to a device. Not a failure. | `checks.StatusSkipped` |
| **unknown** | A check that has not run, or ran without a usable result. Not a failure. | `checks.StatusUnknown` |
| **dashboard** | The React single-page app. | `web/` |
| **file transfer** | A download or upload of file contents over HTTP on the agent port. | `controltower/file_transfer.go` |
| **transfer id** | A one-time id that lets one agent HTTP request reach its transfer. | `agentv1.FileTransfer` |
| **terminal** | A PTY session opened from the browser. | `worker/agent_terminal.go` |

## Words to avoid

- **server**: only for an actual HTTP or gRPC server (`http.Server`,
  `grpc.Server`). Do not use it for the master.
- **tower / control tower**: product prose only. It is not a second name for
  the master in code, and it is not a package name after batch 5 of the
  readability plan.
- **worker**: only the Go package name `internal/worker`. In prose and new
  code, say agent.
- **node / host**: use device for a fleet member. Host is fine for the SSH
  address field inside a deployment.
- **standalone**: removed. There is no standalone mode.
- **upload session / chunked upload**: the chunked upload API in
  `internal/files` is unused legacy code, not the upload path. Real uploads use
  a `FileTransfer` over HTTP.
