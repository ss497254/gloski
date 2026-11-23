# 10 — Do not borrow

A consolidated exclusion list, with reasons. This is the counterweight to the
rest of the folder: Beszel is a large, mature codebase and most of it should stay
there.

| Area | Decision | Reason |
|---|---|---|
| PocketBase as the hub | Skip | Replaces Gloski's API, DB, auth and realtime layer; a giant dependency; conflicts with "keep it simple". Almost every `internal/hub` file takes `core.App`. |
| Multi-user, OAuth/OIDC, roles, sharing, collection rules | Skip | Explicitly out of scope. |
| Alerts, quiet hours, shoutrrr delivery, email templates, SSRF destination blocking | Skip | No alerts wanted. The SSRF guard would also block legitimate internal destinations such as a local ntfy. |
| "Automatic backups" | Skip | Not beszel code — it is PocketBase's admin-UI feature. Build Gloski's nightly copy using `docs/ideas-from-smdwin.md` §4 instead. |
| WebSocket/CBOR transport, request manager, SSH fallback server | Skip | Gloski has a gRPC bidi stream and an owner decision of no SSH fallback for actions. Only the deadline rules are portable (see [04](04-protocol-and-transport.md)). |
| Universal tokens, auto-registration from token | Skip | Gloski's join-token and SSH deployment model differs. Keep the fingerprint mismatch rule only. |
| Agent self-update, systemd/OpenRC/procd/rc.d units, manual install/uninstall | Skip | The master owns the lifecycle; `/tmp` installs are deliberate; agents are never installed by hand. |
| Windows/macOS/FreeBSD/OpenWrt/NAS install matrix; Homebrew, Scoop, WinGet, Debian packages | Skip | Gloski targets Linux devices on a Tailnet, deployed from the dashboard. |
| Six container image variants, Helm charts, 30+ release targets | Skip | Matrix sprawl; Gloski cross-builds three architectures and embeds them. |
| `install-hub.sh`'s floating `latest` with no checksum | Skip | Keep Gloski's SHA-256 pinning discipline. |
| Remote script fetched from `raw.githubusercontent.com` and executed | Skip | Trust model differs; the master is the source of truth. |
| GPU collectors, Windows LHM/.NET helper, Wi-Fi nl80211, nvtop | Skip for now | Large, platform-specific surfaces. Revisit only if a real device needs a metric a shell command cannot answer. |
| SMART/ZFS/btrfs/Docker collector trees as features | Skip for now | Same reason; use saved checks first. Keep the parsers in the reference folder for later. |
| Smartmontools code (GPL-2.0) and LibreHardwareMonitor (MPL-2.0) | Skip | Licence obligations; neither is needed. Do not copy their source. |
| A combined health score | Skip | Beszel's own changelog shows how misleading it gets (a clean run scoring 43/100). Roadmap says never a combined pass/fail. |
| Notification centre / alert banner as a feature | Skip | Reuse only the *dismissal semantics* for dashboard issues. |
| Beszel branding, logo, screenshots, Lingui catalogues, Crowdin | Skip | Irrelevant or premature. |
| Recharts JSX, nanostores, router, PocketBase subscriptions | Skip | Same stack differences as above; port the math and patterns, not the bindings. |
| Beszel's missing Go test CI | Do not copy | It is a gap; Gloski's test + race CI is stronger. |

## Already in Gloski — do not reimplement

- gRPC bidirectional stream with exponential backoff, job replay and protocol
  version gate (`docs/architecture.md`).
- Master-owned agent lifecycle with one general repair: SSH install with SHA-256
  verification, idempotent script, backoff and plain-language SSH errors
  (`docs/reliability-plan.md`).
- Connection diagnosis chain and dashboard issues with advice, route and
  error-first ordering (`docs/ideas-from-smdwin.md`).
- Job logs capped at 10 MB and 30-day retention.
- per-device WebSocket stats fanout, process listing, file transfers on a
  separate HTTP connection, PTY terminal, file editor.
- Chart.js charts and a btop-style device overview — this was borrowed and the
  history gap it identified is now filled by `metric_samples` and the device
  charts.

## Licensing reminders

Beszel is MIT, © 2024 henrygd. The project is credited in this repository's
README; there is no separate notices file or per-file header (owner decision,
2026-11-23). Revisit if Gloski is ever distributed publicly. Two vendored
licences are not MIT:
`supplemental/licenses/smartmontools/LICENSE` (GPL-2.0) and
`supplemental/licenses/LibreHardwareMonitor/LICENSE` (MPL-2.0). The UI icon
file carries per-icon comments; preserve them. Container images install distro
packages (smartmontools, zfsutils, nvtop, igt-gpu-tools) whose licences are not
inventoried — if those Dockerfiles were ever borrowed, a licence pass would be
needed.
