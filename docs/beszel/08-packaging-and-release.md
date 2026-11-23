# 08 — Packaging, install and release engineering

Beszel's install scripts are far larger than Gloski's, and almost all of that
size is the *opposite* of Gloski's model: service managers, package managers,
self-update timers and manual uninstall. Port only the safety discipline.

## Script inventory

`supplemental/scripts/`:

| File | LOC | Description | Class |
|---|---:|---|---|
| `install-agent.sh` | 1,473 | POSIX-sh agent installer across Linux (systemd/OpenRC/procd), FreeBSD/OPNsense/pfSense; checksum verify, rollback | B, selective |
| `install-hub.sh` | 494 | Hub installer; systemd/FreeBSD rc.d; **no checksum and floating `latest`** | C / D |
| `install-agent-brew.sh` | 255 | macOS/Homebrew, tap migration, env-file normalisation | D |
| `install-agent.ps1` | 666 | Windows via Scoop/WinGet + NSSM + firewall | D |
| `upgrade-agent.ps1` / `upgrade-agent-wrapper.ps1` | 414 / 81 | Windows upgrade; wrapper fetches the latest script from raw.githubusercontent.com | D |
| `beszel-agent-launcher.sh` | 17 | Homebrew env-file launcher | D |

There is **no backup/restore script** anywhere in the repo. See
[05](05-hub-and-storage.md).

## `install-agent.sh` — what is worth reading

### Platform matrix (skip all of it)

| Platform | Service manager | Install path | Auto-update |
|---|---|---|---|
| Generic Linux | systemd | `/opt/beszel-agent/beszel-agent` | systemd timer |
| Alpine | OpenRC | same | root crontab `12 0 * * *` |
| OpenWrt | procd (`USE_PROCD=1`) | same | root crontab |
| FreeBSD | rc.d + `sysrc` | `/usr/local/sbin/beszel-agent` | cron.d |
| OPNsense | rc + `rc.syshook.d/start/99-…` | same | cron.d |
| pfSense | rc + boot script | same | cron.d |
| macOS / Windows | refused / separate `.ps1` | — | — |

Every one of those paths and service managers conflicts with Gloski's
`/tmp/gloski` install, master-owned lifecycle and "no systemd push" decision.
Do not port them.

Architecture detection (`:236-287`): `x86_64→amd64`, `armv5*→armv5`,
`armv6l→arm`, `armv7l→armv7`, `aarch64→arm64`, and **MIPS endianness from byte 6
of a system ELF binary** via `hexdump -n 1 -s 5`. glibc detection checks loader
paths then `ldd --version` to select the `_glibc` archive.

### Safety tricks to adapt

| Trick | Lines | Detail | Class |
|---|---:|---|---|
| Named install phases | `INSTALL_STEP`, `:455,461` | Every failure says which phase failed ("…while replacing the agent binary") | **A** |
| Trap-driven cleanup and rollback | `:453-492` | `trap cleanup 0`; HUP/INT/TERM → 129/130/143; on `UPGRADE_PENDING` remove staged file, restore `.bak`, re-apply SELinux context, restart service **only if registered** | B |
| Tri-state registration check | `:438-451` | systemd `LoadState`: `not-found` → 1 (definitely absent), empty → 2 (cannot tell), else 0 — so rollback can warn rather than guess | C (translate to PID file: absent / unreadable / present) |
| Staged atomic replace | `:995-1015` | `mktemp "$BIN_PATH.XXXXXX"` on the destination FS → copy/chown/chmod → `cp -p $BIN $BIN.bak` → stop process → `mv -f` | **B (strong)** |
| Reinstall preserves unmentioned config | `:516-521,1081-1089,1173-1188,1382-1390` | `KEY_PROVIDED`/`PORT_PROVIDED`/`TOKEN_PROVIDED`/`HUB_URL_PROVIDED`; `sed` only provided fields; OpenWrt parses the existing env line and merges missing values; warns on unparseable lines | **A (idea)** — critical for Gloski's reinstall loop |
| Checksum manifest lookup | `:951-981` | Download `*_checksums.txt`, `awk '$2 == FILE_NAME'`, validate 64 hex chars, `sha256sum` or FreeBSD `sha256 -q` | B |
| Archive pre-flight | `:970-993` | `tar -tzf` validity, extract only the agent binary, `-s` non-empty check | **A** |
| Version resolution | `:931-947` | `get.beszel.dev/latest-version` with GitHub API fallback; `-v` strips a leading `v` | B (Gloski pins by master build instead) |
| Retry / timeout / mirror | `:932-968,609-630` | `--retry 3 --retry-delay 2 --connect-timeout 10`; optional China mirror | C |
| Capability checks before changes | `:393-410,545-572` | Fail early with explicit reasons | C |
| SELinux fallback chain | `:42-86,1017-1018` | `semanage fcontext -a/-m` → `restorecon` → `chcon`; re-label on rollback because a temp inode loses its context; clean up on uninstall | C |
| Persistent machine-id | `:1024-1034` | Ensure `/etc/machine-id`; write UUID from `/proc/sys/kernel/random/uuid`, else `uuidgen`, else warn | C |
| Config-conflict failure | Helm `secret.yaml` | `{{- fail }}` when both `KEY` and `existingSecret` are supplied | A (idea) |
| Immutable release version | `.github/workflows/helm-charts.yml:96-105` | Check the version is unpublished before pushing | C |

`install-hub.sh` is a useful *negative* example: it downloads a floating
`latest` with no checksum and only checks `tar -tzf` and non-empty size. Keep
Gloski's SHA-256 pinning.

## Containers, Helm, Debian — skip

Six Dockerfiles (`internal/dockerfile_agent{, _alpine, _intel, _nvidia,
_nvidia_slim, _hub}`) show some clever tricks: `scratch` image with a CA bundle
and a copied `/tmp` directory; `ldd`-driven harvesting of dynamic libraries and
loader into a distroless rootfs; pinned smartmontools built from source for the
NVIDIA image. Gloski does not ship agent containers, so these are **C** at most.

Helm charts (`supplemental/helm/`) are a full DaemonSet/Deployment scaffold with
~1,500 LOC of values and READMEs. Debian packaging
(`supplemental/debian/`) is a service unit plus debconf maintainer scripts.
Both are **D**. The only reusable habits are failing on mutually exclusive
config and documenting each lintian override.

`agent/tools/fetchsmartctl/main.go` (130) is a build-time fetcher: pinned
SHA-256 in `smart.go:1-2`, stream through `io.MultiWriter(file, hasher)`, detect
hash length (SHA1 vs SHA256), delete on mismatch, chmod, atomic rename. Same
discipline as Gloski's embedded agent builds — **C**.

## Build, CI and release

| File | LOC | Note | Class |
|---|---:|---|---|
| `Makefile` | 141 | OS/ARCH overrides, NVML/glibc detection, Windows-only .NET and `go generate` steps, `-ldflags "-w -s"`, `go test -tags='testing no_ui' ./...` | B (conditional build steps) |
| `.goreleaser.yml` | 291 | Hub: 4 OS × 3 arches; agent: 5 OS × 8 arches × GOARM 5/6/7 × GOMIPS variants; deb/scoop/brew/winget; `release.draft: true`; GoReleaser's default checksums manifest | B (manifest + fork-safe `skip_upload` ideas) |
| `.github/workflows/docker-images.yml` | 145 | 6-image matrix, QEMU/Buildx, dual push, provenance + SBOM, tag-only | C |
| `.github/workflows/release.yml` | 54 | Tag-triggered; `IS_FORK` gating | C |
| `.github/workflows/helm-charts.yml` | 109 | Lint, template, version-unpublished check, OCI push | C |
| `.github/workflows/vulncheck.yml` | 29 | `govulncheck` on main pushes | C — consider for Gloski |
| `.github/dependabot.yml` | 12 | Weekly gomod + actions | B |

**Gap not to copy:** there is no general Go unit/race test CI workflow — only
vulncheck, release, docker, helm and repo maintenance. Gloski's
`go test ./...` + `go test -race ./internal/...` is stronger. Keep it.

## Operational documentation

`supplemental/guides/` contains only `systemd.md` (132 LOC, manual installs —
**D**). Reverse-proxy, OAuth, LXC and NAS guides live on beszel.dev, not in the
repo. `SECURITY.md` (9 LOC) asks users to open issues rather than advisories —
irrelevant.

`supplemental/CHANGELOG.md` (942 LOC, 36 releases) is worth skimming for the
failure modes that produced the reliability rules in
[04](04-protocol-and-transport.md): wrong listen address after upgrade,
environment variables lost on reinstall, health checks disagreeing with
connectivity, slow hubs dropping agents, connection timeouts during slow
collection, SELinux context after binary replacement, and update failures from
DB access or working directory.

## Conflict register

| Beszel behaviour | Gloski reality | Action |
|---|---|---|
| systemd/OpenRC/procd/rc.d units, enable/restart, sandboxing | Owner does not want systemd pushed for agents | Skip service-manager code |
| Install into `/opt`, `/usr/local/sbin`, Homebrew/Scoop/WinGet prefixes | `/tmp/gloski` is deliberate | Skip path/package-manager logic |
| `install-agent.sh -u`, `beszel-agent update`, cron/timer auto-update | Master owns lifecycle; agents never self-update | Skip |
| Manual local installs (`.ps1`, brew, guide) | Deployments only from the dashboard | Skip |
| Six container variants and Helm charts | Three embedded architectures, keep it simple | Skip |
| Floating `latest` with no checksum | Pin and verify | Do not copy this weakness |
| Remote script fetched and executed | Master is the source of truth | Skip |
