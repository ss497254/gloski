# 02 — Agent and metrics

The `agent/` directory is 199 files: about 15k non-test Go LOC, 22.5k test LOC
and 33 fixture files. It is where Beszel has spent most of its effort. Almost
every collector is written against Beszel's `system.Stats` entity and CBOR tags,
so **A — copy** still usually means "swap the struct, drop the tags". The one new
dependency that unlocks most of this code is `github.com/shirou/gopsutil/v4`.

## Agent structure and lifecycle

| Item | Source | Class | Note |
|---|---|---|---|
| `Agent` god-struct, `gatherStats`, cache-time enrichment, `hubIdleFor` | `agent/agent.go` (304) | C | One cache entry per polling interval; `detailsDirty` re-send flag; wall-clock idle detection counts suspend toward idle. Do not copy the struct. |
| WebSocket client | `agent/client.go` (501) | D / C | Gloski has gRPC. Borrow: signed auth challenge, dialer that ties socket lifetime to the agent context, rotating User-Agent, "close on write failure to trigger reconnect". |
| WS↔SSH state machine | `agent/connection_manager.go` (429) | D / C | Gloski has no SSH fallback. Borrow the race guards: ignore stale callbacks, track SSH per TCP connection, keep the reconnect ticker armed after an async handshake failure. |
| SSH fallback server | `agent/server.go` (312) | D | Opposite of Gloski's design. |
| Data directory probing | `agent/data_dir.go` (119) | B | env → platform dirs → first existing-and-writable, else first creatable; write/remove `.write-test`. |
| Per-interval cache | `agent/agent_cache.go` (55) | **A** | Fresh if age < half the requested cache interval. |
| Fingerprint | `agent/fingerprint.go` (90) | C | Persisted SHA-256 of host ID, known-bad UUID guard, hostname+CPU fallback. See [10](10-do-not-borrow.md) and the identity gap. |
| Delta tracker | `agent/deltatracker/deltatracker.go` (100) + tests (217) | **A** | Generic `DeltaTracker[K,V]`: `Set`, `Deltas`, `Previous`, `Delta`, `Cycle` (previous = current). Thread-safe. Only external dep is `x/exp/constraints`; Go 1.24+ has `constraints` built in. Ideal for network/disk rates. |

## CPU, memory, load and uptime

| Item | Source | Class | Algorithm / edge case |
|---|---|---|---|
| Cross-platform CPU sampling | `agent/cpu.go` (154) | C | `lastCpuTimes` and `lastPerCoreCpuTimes` maps **keyed by cache-time interval**; `clampPercent`; Linux excludes Guest/GuestNice from the total; counter rollback resets the baseline. |
| LXC-aware cgroup CPU | `agent/cpu_linux.go` (337) | **A** | Detects lxcfs-mounted `/proc/stat`, uses `cpu.stat`/`cpuacct.usage` instead; `cpuCgroupCores` = min(affinity, cpuset, quota); parses `cpu.list` "0-3,5"; v1 jiffies → µs. |
| cgroup memory | `agent/memory_linux.go` (89) | **A** | v1 "unlimited" sentinel, subtract shmem from cache, container-scoped accounting only when explicitly enabled. |
| Memory totals | `agent/memory.go` (77) | B | `hostSwap` fallback on non-Linux, ZFS ARC subtraction, saturating unsigned math, byte arrays alongside deprecated GB floats. |
| Host details | `agent/system.go` (406) | B | Static info from gopsutil; MIPS `cpu_model`/`system type` parsing from `/proc/cpuinfo`; socket count; defensive `/proc/meminfo` read; `saturatingSub`; immutable-OS root detection. |
| Uptime | `agent/uptime_linux.go` (44) | **A** | Reads `/proc/uptime` rather than calling `sysinfo(2)` because lxcfs cannot virtualise the syscall; rejects NaN/Inf/negative; falls back to `host.Uptime()`. |

## Disks and I/O

`agent/disk.go` (900 LOC) plus platform stubs is the richest subsystem.

**A-grade (port the math):**

- Per-cache-time `prevDisk` snapshots plus a shared `diskBaseline`.
- The first sample must span at least half the collection interval, otherwise
  rates are nonsense.
- Values above 50,000 MB/s are rejected as bad reads.
- `ioTimeDelta` handles the Linux 32-bit `/proc/diskstats` counter wrap at 2³².
- Derived metrics: read/write time %, utilisation %, `r_await`, `w_await`,
  weighted I/O %, all as a fixed `[6]float64`.

**B-grade (adapt):**

- Device → I/O-counter resolution: LVM symlinks (`/dev/vg/lv` → `dm-N`), exact /
  prefix / busiest-candidate fallback in `findIoDevice`.
- Root-device cascade: `FILESYSTEM` env → partition heuristics → most-active
  device last resort, with 16 MiB tolerance for duplicate-root pruning.
- `FILESYSTEM=device__name` custom naming; `/extra-filesystems` mounts.
- ZFS dataset mounts use `zfs list` rather than statfs.

**C-grade (ideas):**

- `DISK_USAGE_CACHE` avoids waking sleeping disks.
- Immutable OS roots (Fedora Silverblue, Atomic, Flatcar) map to `/sysroot`.
- `agent/disk_darwin.go` (60): `diskutil info -plist` with a 15 s timeout to map
  an APFS volume to its physical disk.

**Network counters:**

- `agent/network.go` (260): `NICS` allow/deny with `*`, per-NIC delta tracker,
  **sum of deltas** rather than difference of totals (survives interface resets),
  invalid >10 GB/s resets the baseline, skips `lo`/`docker`/`veth`/`bond`.
- `agent/network_counters_linux.go` (71): Jetson `nvethernet` MMC octets from
  ethtool; split low/high registers are combined by **addition, not OR**.

## GPU (~3,500 LOC)

| Item | Source | Class | Note |
|---|---|---|---|
| Collector manager | `agent/gpu.go` (888) | B | Streaming parse, exponential backoff 5 s → 5 min, restart-forever, one collector per vendor group, per-interval usage/power/engine snapshots. |
| AMD sysfs | `agent/gpu_amd_linux.go` (487) | **A** | APU vs discrete from `gpu_metrics` header revision (1=dGPU, 2/3=APU), GTT on APUs, backoff for EOPNOTSUPP/slow reads, `amdgpu.ids` PCI-ID cache. |
| NVML FFI | `agent/gpu_nvml.go` (309) | C | `purego` into `libnvidia-ml.so.1`; D3cold avoidance (skip memory poll at 0% util); power-sample fallback. Heavy — idea only. |
| nvtop | `agent/gpu_nvtop.go` (218) | B | Device IDs reorder across boots; match by name. |
| Intel `intel_gpu_top -J` | `agent/gpu_intel.go` (296) | **A** | Tolerates **missing commas between JSON objects** (Arrow Lake), skips the first bad sample, rejects >5,000 W from unsigned wrap. |
| Intel sysfs energy | `agent/gpu_intel_sysfs_linux.go` (280) | **A** | `energy1_input` µJ → power via delta/time with reset detection; multiple VRAM path variants. |
| macOS | `agent/gpu_darwin.go` (257) | B | powermetrics idle residency → utilisation, macmon JSON lines. |
| Platform stubs | `gpu_*_unsupported.go` | C | `//go:build` tags plus tiny stubs so one binary builds everywhere. Adopt this pattern. |

## SMART and disk health

| Item | Source | Class | Note |
|---|---|---|---|
| Collection orchestration | `agent/smart.go` (574) | B | **Wake a disk at most once per run** and reuse last good data; 30-min scan cache; `Complete` flag for hub-side pruning; ATA→SCSI attempt ordering; embedded `smartctl.exe` on Windows. |
| JSON parsing | `agent/smart_parse.go` (342) | **A** | Protocol detection (nvme/scsi/ata), reject virtual devices (QEMU/VBOX/VMWARE/MSFT/IET), unified attributes, `"7344 (253d 8h)"` and `"0h+0m+0.000s"` formats. |
| Entity parsers | `internal/entities/smart/smart.go` (308) | **A** | `NVMeCounter` treats 128-bit overflow as a value, not a decode failure. |
| mdraid | `agent/mdraid_linux.go` (277) | A/C | Sparse-slot (QNAP) and redundant-mirror (Synology) heuristics to downgrade misleading "degraded". |
| eMMC | `agent/emmc_{common,linux}.go` (95+185) | **A** | `pre_eol_info`, JEDEC life-time A/B → %, both `life_time` layouts. |

## Storage pools (ZFS and btrfs)

| Item | Source | Class | Note |
|---|---|---|---|
| ZFS commands | `agent/zfs/zfs.go` (222) | **A** | Root-dataset capacity instead of parity-inclusive figures; refquota roots skipped; `LC_ALL=C`; 10 s timeout. |
| ZFS kernel counters | `agent/zfs/zfs_linux.go` (302) | **A** | Tiered sources `iostats` → `io` → sum `objset-*`; **never switch source after a usable read** (prevents false spikes); QNAP `/proc/lpl/...`; `/dev/zfs` fast-fail in containers. |
| btrfs | `agent/btrfs/btrfs_{,linux}.go` (26+395) | B | ioctl fallback for pre-5.13 kernels (Synology); filesystem UUID as stable ID across subvolumes/bind mounts; missing/replaced/sparse devices; DEGRADED health. |
| Manager | `agent/storage_pool.go` (514) | **B (strong)** | Auxiliary utility calls have their own cache and background refresh so a hung `zpool list` never delays a stats response; `CompleteBackends` records partial success; duplicate pool charts are hidden. |

## Sensors, fans, battery, Wi-Fi

| Item | Source | Class | Note |
|---|---|---|---|
| Sensor reads | `agent/sensors.go` (423) | **A** (helpers) | Generic `readWithTimeout` with `atomic.Bool` inflight guard so a kernel-blocked read cannot stack goroutines; panic recovery for gopsutil bugs; shadow sysfs avoids waking GPUs; duplicate names disambiguated before filtering. |
| Fans | `agent/fans.go` (182) | **A** | `sync.OnceValues` discovery cache; legacy `applesmc` layout; keeps genuine 0 RPM. |
| Battery | `agent/battery/*.go` (~540) | **A** | Deterministic primary selection (full-charge capacity → system scope → name), strips invalid UTF-8, disambiguates duplicates, skips ghost batteries on macOS. |
| Wi-Fi | `agent/wifi/*.go` (~400) | B | Association proved from BSS status, never a cached scan result; nl80211 gated on `/sys/class/ieee80211` to avoid modprobe; `iwinfo` for OpenWrt APs; drops non-UTF-8 SSIDs. |
| Windows LHM | `agent/sensors_windows.go` (377) + `agent/lhm/*.cs` | D | .NET helper; Windows only. |

## Docker / Podman

`agent/docker.go` (1,109), `docker_registry.go` (224), `docker_image_updates.go`
(108), `docker_names.go` (123), `internal/entities/container/container.go` (205).

**Take (A-grade ideas):**

- Counter-rollback guards in `CalculateCpuPercentLinux/Windows/Podman`.
- Bounded log decoding: 5 MB total / 1 MB frame caps, multiplexed-stream
  sniffing, ANSI stripping, Env removal from inspect output.
- Auxiliary network work (registry digest checks) runs **outside** the metrics
  WaitGroup so a slow registry cannot delay stats.
- Semaphore-limited per-container stats requests.

**Adapt (B):** the collector itself — Podman/Windows detection from the `Server`
header, Docker <25 one-shot retry, per-interval CPU/network deltas, memory =
usage − inactive_file.

**Skip (D):** image-update checks and registry lookups, unless update checks
become a saved check.

## Package updates, health file, identity

| Item | Source | Class | Note |
|---|---|---|---|
| Package update parsers | `agent/package_updates.go` (644) | B | Pure-Go parsers for apt, dnf, zypper, pacman, apk, xbps; hourly background cache in a private dir. Useful as ready-made **shell check definitions** for roadmap step 3, not as a feature. |
| Health file | `agent/health/health.go` (88) + tests (140) | A/C | Touch a file every 90 s, prefer `/dev/shm` over `/tmp`, read-only check picks the newest mtime, idempotent cleanup. Gloski's live gRPC stream may already cover liveness; use this only as a cheap local probe in the diagnose chain if needed. |
| Fingerprint | `agent/fingerprint.go` (90) | C | `sha256(host.HostID())[:24]` hex; rejects the known non-unique product UUID `03000200-0400-0500-0006-000700080009`; falls back to hostname + CPU model; persists so hostname changes do not create a new identity. Gloski stores `machine_id` but still keys devices by name. |
| CLI | `internal/cmd/agent/agent.go` (249) | C | `fingerprint view|reset` and `health` subcommands; key from flag/`KEY`/`KEY_FILE`. |

## What to actually take from this area

1. **Now, with no new feature surface:** `deltatracker/`, `uptime_linux.go`,
   `cpu_linux.go`, `memory_linux.go`, the sensor timeout/inflight helpers, and
   the test patterns in [09](09-testing-and-quality.md).
2. **With checks (roadmap step 3):** probe files from
   [06](06-checks-alerts-and-monitors.md); package-update parsers as command
   definitions; SMART/eMMC/mdraid parsers for check output.
3. **Only on demonstrated need:** GPU, ZFS/btrfs, Docker and Wi-Fi collectors.
   Each is a large, platform-specific surface. A shell command in a saved check
   is usually the cheaper answer.
4. **Never:** the WebSocket client, connection manager, SSH server or agent
   self-updater. Gloski's gRPC lifecycle and master-owned repairs supersede all four.
