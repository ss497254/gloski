# Glossary

Gloski has one word per concept. Use these terms in code, commits, docs and UI
copy.

| Term | Meaning |
|---|---|
| **Gloski** | The project and the product. |
| **master** | The control process: dashboard, browser API, SQLite database, and agent lifecycle. |
| **agent** | The device-side process (`gloski-agent`). One per device, including the master host. |
| **device** | A machine enrolled in the fleet. |
| **master device** | The device record representing the master host itself; its agent connects over a Unix socket. |
| **local agent** | The agent on the master host, started and supervised as a child process by the master. |
| **remote agent** | An agent on another device, reached over h2c on the Tailnet. |
| **deployment** | The SSH record of how an agent was installed on a device. |
| **install phase** | The named stage in an install error (`platform`, `prepare`, `binary`, `config`, `replace`, or `start`), carried in the error text. |
| **job** | A command the master owns and one agent executes. |
| **session** | One live bidirectional gRPC stream for a device. |
| **diagnosis** | An ordered chain of connection checks from the master outwards, with the first broken link as the headline. |
| **self-test** | A read-only checklist of the master's own health, shown under Settings → Master health. |
| **device action** | A master-owned change to a device, recorded so it can be explained and, when the previous wanted state is known, undone. |
| **undo** | Restoring the wanted state recorded in a reversible device action. |
| **check result** | One run of one check or native probe on one device: verdict, summary, details, timing. |
| **native probe** | A TCP, HTTP, DNS or TLS check run by the agent itself, requested over the existing gRPC stream. |
| **stable check ID** | A deterministic hash of a probe's kind and target (plus accepted statuses), so schedule changes do not split history. |
| **idle suppression** | Native probes pause after three minutes with no dashboard or device-stats request; heartbeats and metric sampling continue. |
| **metric sample** | One metric value recorded by the master from a heartbeat, at most once every five minutes per device and metric. |
| **check group** | A fixed column of the fleet matrix (`Connectivity`, `Storage`, `Services`, `Updates`, `Backups`, plus `Other`), derived from check ID prefixes. |
| **fleet check matrix** | Devices as rows and check groups as columns, one coloured block per check, ordered red first. Grey means skipped, stale or never run. |
| **stale** | A check result older than twice its schedule interval. Recomputed when read. |
| **skipped** | A check that does not apply to a device. Not a failure. |
| **unknown** | A check that has not run, or ran without a usable result. Not a failure. |
| **dashboard** | The React single-page app. |
| **file transfer** | A download or upload of file contents over HTTP on the agent port. |
| **transfer id** | A one-time ID that lets one agent HTTP request reach its transfer. |
| **terminal** | A PTY session opened from the browser. |

## Words to avoid

- **server**: only for an actual HTTP or gRPC server. Do not use it for the master.
- **tower / control tower**: product prose only, not a separate code concept from master.
- **worker**: replaced by agent; do not use it in code, prose, or UI.
- **node / host**: use device for a fleet member. Host is fine for the SSH address field inside a deployment.
- **standalone**: removed. There is no standalone mode.
- **upload session / chunked upload**: legacy code removed; real uploads use HTTP file transfers.
