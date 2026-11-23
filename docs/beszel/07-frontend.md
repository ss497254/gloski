# 07 — Frontend

Beszel's SPA at this commit is **React 19.1 + TypeScript + Vite 7 + Tailwind v4
+ shadcn/Radix** — nearly the same stack as Gloski's. This is the most directly
portable part of the codebase: most components need dependency swaps, not a
framework rewrite. All paths are under `internal/site/` unless stated.

## Stack comparison

| Concern | Beszel | Gloski | Work to port |
|---|---|---|---|
| Framework | React 19.1.2 | React 19 | none |
| Build | Vite 7.3.5 + `@vitejs/plugin-react-swc` | Vite 7 | none |
| CSS | Tailwind v4.1 + `@tailwindcss/vite` + `tw-animate-css` | Tailwind v4 | none |
| UI primitives | shadcn (`components.json`), 15 Radix packages | shadcn/Radix | none/trivial |
| Icons | `lucide-react` + 19 hand-made SVGs | lucide-react | drop-in |
| Charts | **Recharts 2.15.4** + `d3-time` | **Chart.js** | render layer only; math is portable |
| Tables | **TanStack Table 8.21** | own | pattern or full port |
| State | **nanostores** + `@nanostores/router` | Zustand + React Router | small rewrite |
| i18n | Lingui 5.4 + 34 `.po` catalogues + Crowdin | none | skip for now |
| Validation | `valibot` 1.4 | — | portable |
| Real-time | PocketBase SSE subscriptions | gRPC/WS via master | swap transport, keep store patterns |
| Syntax highlight | `shiki` 3.13 | CodeMirror 6 | choose one |
| Tests | `bun:test`, 8 files | `bun test` | none |
| Lint/format | Biome 2.3.15 (tabs, 120 cols, double quotes, no semicolons) | eslint/prettier | copy config if wanted |

Vite config uses `base: "./"` — useful if Gloski is served under a proxy path.

## Pure TypeScript to lift verbatim (A)

These have no framework coupling beyond `clsx`, local enums/types or Lingui
macros that can be shimmed. Test files come with several of them.

| File | LOC | Functions | Notes |
|---|---:|---|---|
| `src/lib/utils.ts` | 538 | `toFixedFloat`, `formatNumber`, `formatMicroseconds`, `formatTemperature`, `formatBytes` (bytes/bits + `/s`), `formatDuration`, `parseSemVer`/`compareSemVer`, `debounce`, `debounceByKey`, `runOnce` (WeakMap), `parseFilterGroups`/`matchesFilterGroups`, `secondsToUptimeString`, `generateToken`, `getHubURL`, cached `Intl.NumberFormat` | **Highest-value single file.** Replace the Lingui locale reference and drop the `plural` call from `secondsToString`. Test: `tests/utils.test.ts` (87). |
| `src/components/routes/system/chart-data.ts` | 99 | `appendData` — inserts gap markers when a delta exceeds 1.5× the expected interval, converts `created` strings to ms in place, caps array length in one slice; `chartTimeData`, `makeContainerData/Point`, `dockerOrPodman` | Directly fixes Chart.js line-continuity problems. |
| `src/components/ui/chart.tsx` | 493 | `fixedDomainTicks` (four equal ticks so the axis does not stop one short), `xAxisCache`/`xAxis()` domain math, tooltip Total row, item sorter/truncation, `roundFloat` | Port the math, rewrite the JSX for Chart.js. |
| `src/components/charts/hooks.ts` | 176 | `hasChartValues`, HSL hue distribution for N series, `useYAxisWidth` off-screen probe, filter fading of series | Generated colours solve many-series legibility. |
| `src/components/charts/line-chart.tsx` | 236 | `isolatedDot` — draw only points with no neighbours so data gaps are visible | Pure predicate plus a visual idea. |
| `src/lib/network-monitor-utils.ts` | 157 | `withMonitorGaps`, `mergeSameTimestamps`, `getCertDaysLeft`, `getCertExpiryLevel`, `parseMonitorFields` (quoted CSV) | Test: `tests/network-monitor-utils.test.ts` (171). |
| `src/lib/package-updates.ts` | 85 | `classifyVersionChange`, `splitVersion` across apt/dnf/zypper/pacman/apk/xbps, epochs, calendar versions | Test: `src/lib/package-updates.test.ts` (63). Useful for formatting check output. |
| `src/lib/monitor-compare.ts` | 149 | Target formatting, identity keys, compare selection/caps | Deliberately UI- and store-free so it can be unit-tested. |
| `src/lib/wifi.ts` | 49 | `connectedWiFi`, `primaryWiFi`, `wifiSignalState`, `wifiColor` | Test: `wifi.test.ts` (56). |
| `src/lib/use-intersection-observer.ts` | 160 | `useIntersectionObserver` with threshold/root/rootMargin/freeze/onChange | Drop-in React hook; drives lazy mounting. |
| `src/components/ui/icons.tsx` | 198 | 19 SVG icons (Tux, Windows, Apple, FreeBSD, Docker, GPU, thermometer, Ethernet, WebSocket, battery states…) | Drop-in; keep licence comments. |
| `src/lib/languages.ts`, `enums.ts` | 36 / 96 | 35 locale/flag tuples; `Unit`, `MeterState`, `SystemStatus`, `HourFormat`, `ConnectionType` | Data only; maps to Gloski concepts. |
| `src/lib/i18n.ts` | 75 | Locale detection, zh-variant mapping, RTL flags, persistence | Port the logic if Gloski adds i18n; swap Lingui. |
| `src/components/install-dropdowns.tsx` | 99 | Copy docker-compose / docker run / Linux / Homebrew / Windows commands | Borrow the stable-token + copy UX only (Gloski has no manual install path). |

## Charts and live data flow

`src/components/routes/system/use-system-data.ts` (390) is the blueprint:

- Subscribe to realtime only for the `1m` range; the first message clears data
  from other periods.
- Cap 60 live points and 100 history points.
- Stale-cache check: `now - last < interval * 0.9`.
- Request-ID guard so a late response from a previous range cannot overwrite
  the current one.
- localStorage-first settings, then server hydration.
- Keyboard ←/→ (or h/l) to cycle systems; Shift changes the split in tabs mode.
- Hide the `1m` option when the agent version does not support it.

Other chart pieces:

- `chart-time-select.tsx` (48): range selector that **hides ranges whose
  `minVersion` exceeds the connected agent's version** — a nice extension of
  Gloski's protocol gate.
- `chart-card.tsx` (133): title/description/corner slot, default and grid spans,
  debounced filter bar, Average vs "Max 1 min".
- Chart catalogue (all adapt to Chart.js): `cpu-charts.tsx` (96),
  `memory-charts.tsx` (178), `network-charts.tsx` (181), `disk-charts.tsx` (295),
  `gpu-charts.tsx` (244), `sensor-charts.tsx` (344),
  `storage-pool-charts.tsx` (138), `monitors-charts.tsx` (284),
  `load-average-chart.tsx` (55), `wifi-chart.tsx` (49). `syncId: "io"` syncs the
  cursor across disk charts.
- `charts/*-sheet.tsx` (~720): fullscreen drill-down for CPU, disk I/O and network.
- `chart-no-values.tsx` (16): "Waiting for enough records to display" spinner.

## Tables and views

| Item | Source | Class | Detail |
|---|---|---|---|
| Systems table | `systems-table/systems-table.tsx` (586) + `systems-table-columns.tsx` (696) | B | Table **and grid** views; one "View" DropdownMenu with four panes (Layout, Status with counts, Sort By, Visible Fields); filter with clear; fixed row heights; sticky header; hover-preloads the detail route. Flagship pattern for Gloski's devices list. |
| Cell patterns | same | A/B | `TableCellWithMeter` (thin threshold-coloured bar + `tabular-nums`), multiple-disk dots with per-FS tooltip, version cell coloured against the hub version, security-update ranking. |
| Pagination | `ui/data-table-pagination.tsx` (168) | B | Page size per device in localStorage, `autoResetPageIndex:false` so realtime updates do not jump pages, reset on filter/sort, container-query responsive controls, hides itself when all rows fit. |
| Containers | `containers-table/*` (782) | B | Stale-snapshot merge by `updated` within 70 s, auto-hide empty Ports column, Shiki-highlighted logs, fullscreen, timestamp toggle. |
| systemd | `systemd-table/*` (971) | B | Failed-count chip, `logsStatus` union, request-ID guard, auto-scroll, fullscreen. |
| Network monitors | `network-monitors-table/*` (1,276) + `monitor-dialog.tsx` (1,074) | B | Bulk writes in chunks of 20, confirm dialogs, quoted CSV bulk add, valibot validation, compare sheet. |
| SMART / ZFS / updates | `smart-table.tsx` (812), `storage-pools-table.tsx` (701), `package-updates-table.tsx` (339) | B | Only if those features land. Note dynamic column pruning: hide columns with no data. |
| Alert history | `alerts-history-data-table.tsx` (304) | B | CSV download, delete confirmation — skip with alerts. |

## UX polish worth copying

| Detail | Source | Class |
|---|---|---|
| Pre-paint theme script (no flash of wrong theme) | `index.html` (32) | B |
| Runtime config injected from Go into `index.html` | `internal/hub/server.go` | B |
| Cmd/Ctrl+K command palette with aliases and shortcut labels | `command-palette.tsx` (267) | B |
| Copy with fallback dialog (auto-selected textarea when clipboard needs a secure context) | `utils.ts`, `copy-to-clipboard.tsx` | A/B |
| Masked token with reveal + copy | `token-value.tsx` (50) | B — useful for the join token |
| Fullscreen log view, auto-scroll, timestamp toggle | `logs-display.tsx` (131) | B — mirrors Gloski's terminal/editor |
| Favicon badge with active count | `utils.ts` `updateFavicon` | B |
| Connection-type icon in the status tooltip | `info-bar.tsx`, `ui/icons.tsx` | B — unix socket vs Tailnet TCP |
| Session-scoped banner dismissal by id that forgets resolved items | `active-alerts.tsx` (143) | C — reuse for dashboard issues, not alerts |
| Mobile defaults to grid under 200 systems; Select instead of sidebar | `systems-table.tsx`, `sidebar-nav.tsx` | B |
| Hover-preloads route chunks | `navbar.tsx`, `systems-table.tsx` | B |
| Generated HSL series colours: `hsl(i*360/n, var(--chart-saturation), var(--chart-lightness))` | `index.css` (219) | B |
| Sticky-table patch, `scrollbar-gutter: stable`, Radix scroll-lock fix | `index.css` | A/B |
| Toast viewport: top on mobile, bottom-right on desktop, swipe to dismiss | `ui/toast.tsx`, `use-toast.ts` | B |
| Three-state theme cycle with tooltip/animation | `mode-toggle.tsx` (40) | B |
| `input-tags.tsx` (77), `input-copy.tsx` (36), `cmdk`-based `command.tsx` (131) | `ui/` | A/B |

## Page catalogue

Routes use `@nanostores/router` with a runtime-injected base path
(`components/router.tsx`, 58). Pages:

| Route | Files | LOC | Relevance |
|---|---|---:|---|
| `/` dashboard | `routes/home.tsx`, `systems-table.tsx` | 24 + 586 | Direct analogue of Gloski's device list |
| `/system/:id` | `routes/system.tsx`, `info-bar.tsx`, `use-system-data.ts` | 337 + 255 + 390 | Device page blueprint |
| `/containers` | `routes/containers.tsx`, `containers-table.tsx` | 22 + 519 | Roadmap step 4 |
| `/smart` | `routes/smart.tsx`, `smart-table.tsx` | 16 + 812 | Only with SMART collectors |
| `/monitors` | `routes/monitors.tsx`, `network-monitors-table.tsx` | 29 + 836 | With checks |
| `/settings/:name?` | `routes/settings/*` (general, notifications, tokens/fingerprints, API keys, alert history, heartbeat, YAML config, quiet hours) | ~2,100 | Mostly skip; layout/nav pattern is useful |
| `/login`, `/forgot-password`, `/request-otp` | `login/*` | ~720 | Skip (no multi-user) |

`src/main.tsx` (151) shows the shell: lazy pages + Suspense, provider order,
subscription bootstrap and version/API-key fetch.

## Styling and assets

- `index.css` (219): Tailwind v4 tokens for light/dark, `--radius`, `--chart-1..5`,
  chart saturation/lightness, custom `xs` breakpoint, Twemoji country-flag font
  restricted to regional-indicator codepoints, InterVariable, `@utility` helpers.
- `public/static/`: `icon.svg`, `icon.png`, `InterVariable.woff2`,
  `manifest.json` (PWA, `display: standalone`) — trivial to add to Gloski.
- `types.d.ts` (752): complete domain types with compact keys; use as a schema
  design reference, not as code.

## Frontend tests

8 files, ~958 LOC: `utils.test.ts` (87), `chart-time.test.ts` (41),
`network-monitor-utils.test.ts` (171), `monitor-compare.test.ts` (204),
`package-updates.test.ts` (63), `wifi.test.ts` (56), `token-value.test.tsx` (24),
`load-login.test.ts` (202).

Style to copy: `mock.module()` for Lingui and browser-only modules, dynamic
`await import()` after mocks, `renderToStaticMarkup` for accessibility
assertions, `test.each` tables for formatters.

## Skip

Lingui catalogues/Crowdin, nanostores and router code, PocketBase subscription
hooks, Recharts JSX (keep the math), all login/OAuth/OTP/alert pages, SMART/ZFS/
package tables until those features exist, `src/lib/time.ts` (empty), lockfiles,
`.prettierrc`, and the beszel logo/branding.
