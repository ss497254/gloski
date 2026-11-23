# 03 — Stats and data model

Beszel's data model is the most thought-through part of the codebase: compact
integer keys on the wire, short keys in storage, explicit deprecated slots and
"do not reuse" markers. Gloski should borrow the **versioning discipline**, not
the fields — it already has its own `Stats` with CPU, memory, swap, disks,
network, load and processes.

All definitions live in `internal/entities/`. The main file is
`internal/entities/system/system.go` (246 LOC).

## `Stats` — the main payload

Tags are `json / cbor`. `Max*` fields are hub-side peaks and are never sent
(`cbor:"-"`). Fields marked deprecated are retained for older hubs/integrations.

| Field | json / cbor | Meaning |
|---|---|---|
| `Cpu` | `cpu` / 0 | CPU busy % |
| `MaxCpu` | `cpum` / – | hub-side peak |
| `Mem`, `MemUsed`, `MemBuffCache`, `MemZfsArc`, `MaxMem` | `m`,`mu`,`mb`,`mz`,`mm` / 2,3,5,6,– | Deprecated GB values; use `MemBytes` |
| `MemPct` | `mp` / 4 | used % |
| `Swap`, `SwapUsed` | `s`,`su` / 7,8 | Deprecated GB; use `SwapBytes` |
| `DiskTotal`, `DiskUsed`, `DiskPct` | `d`,`du`,`dp` / 9,10,11 | root filesystem GiB/% |
| `DiskReadPs`, `DiskWritePs` | `dr`,`dw` / 12,13 | Deprecated; agents <0.13; hub migrates to `DiskIO` |
| `NetworkSent`, `NetworkRecv` | `ns`,`nr` / 16,17 | legacy totals |
| `MaxNetworkSent`, `MaxNetworkRecv` | `nsm`,`nrm` / – | hub-side peaks |
| `Temperatures` | `t` / 20 | sensor name → °C |
| `ExtraFs` | `efs` / 21 | mount name → `FsStats` |
| `GPUData` | `g` / 22 | GPU name → `GPUData` |
| — | cbor 23–25 | **removed LoadAvg1/5/15 — "do not reuse"** |
| `Bandwidth` | `b` / 26 | `[sent B/s, recv B/s]` |
| `MaxBandwidth` | `bm` / – | hub-side peak |
| `LoadAvg` | `la` / 28 | `[1m, 5m, 15m]` |
| `Battery` | `bat` / 29 | `[percent, charge state]` |
| `NetworkInterfaces` | `ni` / 31 | interface → `[up/s, down/s, total up, total down]` |
| `DiskIO` | `dio` / 32 | `[read B/s, write B/s]` |
| `MaxDiskIO` | `diom` / – | hub-side peak |
| `CpuBreakdown` | `cpub` / 33 | `[user,system,iowait,steal,idle,irq,softirq,nice]` |
| `CpuCoresUsage` | `cpus` / 34 | per-core busy % (`Uint8Slice`) |
| `DiskIoStats` | `dios` / 35 | `[read time %, write time %, util %, r_await ms, w_await ms, weighted %]` |
| `MaxDiskIoStats` | `diosm` / – | hub-side peaks |
| `Fans` | `f` / 36 | sensor → RPM |
| `Batteries` | `bats` / 37 | name → percent |
| `DiskIOTotal` | `diot` / 38 | cumulative `[read, write]` device bytes |
| `ZfsPools` | `z` / 39 | pool name → `ZfsPool` |
| `WiFi` | `wf` / 40 | interface → RSSI dBm (`int8`); unavailable omitted |
| `MemBytes` | `mem` / 41 | `[total, used, buff/cache, zfs arc]` bytes |
| `MaxMemBytes` | `memm` / – | hub-side peak |
| `SwapBytes` | `swap` / 42 | `[total, used]` bytes |

## Supporting types

**`ZfsPool`** (cbor 0–8): `Total`/`Used` GiB, `ReadBytes`/`WriteBytes` B/s,
`Health` (ONLINE, DEGRADED, FAULTED…), `Raw`, `DisplayName`, and `HideUsage` /
`HideIO` flags that suppress charts duplicating filesystem metrics.

**`GPUData`**: `Name`, `MemoryUsed`, `MemoryTotal`, `Usage`, `Power`, `Engines`
(map), `PowerPkg`. `Temperature` and `Count` are hub-side only.

**`FsStats`**: `DiskTotal`/`DiskUsed` GiB, `TotalRead`/`TotalWrite` cumulative
bytes, `DiskReadBytes`/`DiskWriteBytes` B/s, `DiskIoStats[6]`, plus deprecated
`DiskReadPs`/`DiskWritePs`. `Root`, `Mountpoint` and `Name` are json/cbor omitted.

**`WiFi`** (used in `Info`, not `Stats`): `SSID`, `Signal *float64` in native dBm
(nil means association confirmed but RSSI unavailable — **never convert quality
percent**), `AP` flag, Windows `Profile`.

**`Battery`** `[2]uint8` and **`Uint8Slice`**: custom `MarshalJSON` emits arrays
of numbers instead of base64, with `UnmarshalJSON` so `encoding/json/v2` (used by
PocketBase for stored JSON) accepts them. CBOR keeps the compact byte string.
This dual-encoding trick is a neat solution to a real compatibility problem.

## `Info` — what the dashboard table needs

`internal/entities/system/system.go:187-216`, cbor keys 0–27. Uptime, CPU %,
memory %, disk %, agent version, GPU %, dashboard temperature, bandwidth bytes,
load average, connection type, extra filesystem % map, `[total, failed]`
services, battery, root disk name, `[total, security]` package updates, Wi-Fi
map and a `SystemdLogs` capability flag. Keys 15–17 are retired load-average
slots ("do not reuse"). Several old fields (hostname, kernel, cores, CPU model,
OS, Podman) are marked deprecated and moved to `Details`.

## `Details` — static per-connection data

cbor 0–12: hostname, kernel, cores, threads, CPU model, OS/OS name, arch, Podman
flag, total memory, SMART interval, ZFS interval, physical CPU sockets. Sent
once per connection rather than on every poll.

## `CombinedData` — one response

cbor 0–6: `Stats`, `Info`, `Containers`, `SystemdServices`, `Details`,
`SystemdServicesUpdated` (distinguishes a genuinely empty snapshot from a short
cache request that omitted systemd data) and `Monitors`.

## Other entity packages

| Package | LOC | Notable |
|---|---|---|
| `internal/entities/container/container.go` | 205 | Docker API shapes; CPU-percent functions with counter-rollback guards. |
| `internal/entities/monitor/monitor.go` | 151 | `Config`, `Result`, `CertInfo`, sync request/response; sample counts, 1h windows, loss %. |
| `internal/entities/smart/smart.go` | 308 | Full smartctl JSON model; `NVMeCounter` overflow handling. |
| `internal/entities/systemd/systemd.go` | 127 | Service state + `UpdateCPUPercent` rollback guard. |
| `internal/entities/zfs/zfs.go` | 70 | Pool capacities, health, scrub, datasets. |
| `internal/entities/system/package_updates.go` | 20 | Update counts. |

## Versioning rules worth adopting (C)

These are the real lesson of this file:

1. **Integer keys on the wire**, short keys in storage, both tags on every field.
2. **Reserved slots with comments.** "cbor keys 23–25 were LoadAvg1/5/15 — do not
   reuse." This prevents a future field from being decoded as an ancient one.
3. **Deprecated fields stay for one release cycle**, with a `Deprecated:` comment
   naming the replacement. The hub migrates old values on read.
4. **`omitzero` everywhere**, so unchanged/empty fields cost nothing on the wire.
5. **Fixed tuples instead of nested structs** for related scalars (`[2]uint64`,
   `[3]float64`, `[6]float64`) with a comment describing element order.
6. **Capability flags** (`SystemdLogs`, `Complete`, `SystemdServicesUpdated`) so
   absence of data is not confused with failure to collect it.
7. **Hub-only peaks marked `cbor:"-"`**, keeping device→hub payloads small while
   still supporting dashboard "max" charts.

Gloski's protobuf gives it field numbers for free, but it does not currently
have reserved fields or a documented deprecation policy. Add both when the
stats message next changes, and consider a `capabilities` field for "this agent
can/cannot read SMART / journal / Docker" instead of inferring from empty data.
