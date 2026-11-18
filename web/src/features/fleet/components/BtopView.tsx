import type { ProcessInfo, SystemStats } from '@/shared/api'
import { formatBytes, formatUptime, formatRelativeTime } from '@/shared/lib/utils'
import type { FleetDevice } from '@/shared/tower'
import { cn } from '@/shared/lib/utils'
import { useEffect, useState, type ReactNode } from 'react'
import { Sparkline } from './Sparkline'

interface BtopViewProps {
  device: FleetDevice
  stats: SystemStats | null
  processes: ProcessInfo[]
  connected: boolean
}

function rate(value = 0): string {
  return `${formatBytes(value)}/s`
}

function Panel({
  title,
  children,
  className,
  accent = 'text-emerald-400',
}: {
  title: string
  children: ReactNode
  className?: string
  accent?: string
}) {
  return (
    <section className={cn('overflow-hidden rounded-lg border border-white/10 bg-[#0b0f14]', className)}>
      <header className="flex items-center justify-between border-b border-white/[0.06] px-3 py-1.5">
        <h2 className={cn('font-mono text-[10px] uppercase tracking-[0.2em]', accent)}>{title}</h2>
      </header>
      <div className="p-3">{children}</div>
    </section>
  )
}

function Bar({ value, className }: { value: number; className?: string }) {
  return (
    <div className="h-2 overflow-hidden rounded-sm bg-white/10">
      <div
        className={cn('h-full transition-[width] duration-500', className ?? 'bg-emerald-400')}
        style={{ width: `${Math.min(100, Math.max(0, value))}%` }}
      />
    </div>
  )
}

export function BtopView({ device, stats, processes, connected }: BtopViewProps) {
  const [cpuHistory, setCpuHistory] = useState<number[]>([])
  const [netHistory, setNetHistory] = useState<number[]>([])
  const [memHistory, setMemHistory] = useState<number[]>([])

  /* eslint-disable react-hooks/set-state-in-effect -- roll the history forward per received stats frame */
  useEffect(() => {
    if (!stats) return
    setCpuHistory((history) => [...history.slice(-89), stats.cpu.usage_percent])
    setNetHistory((history) => [...history.slice(-89), (stats.network.rx_rate ?? 0) + (stats.network.tx_rate ?? 0)])
    setMemHistory((history) => [...history.slice(-89), stats.memory.used_percent])
  }, [stats])
  /* eslint-enable react-hooks/set-state-in-effect */

  const online = device.status === 'online'
  const cpu = stats?.cpu.usage_percent ?? 0
  const mem = stats?.memory.used_percent ?? 0

  return (
    <div className="min-h-full bg-[#070a0f] p-4 font-mono text-[#c9d1d9]">
      <header className="mb-3 flex flex-wrap items-center gap-x-5 gap-y-1 text-xs">
        <span
          className={cn(
            'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-wider',
            online ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-300' : 'border-white/10 text-white/40'
          )}
        >
          <span className={cn('h-1.5 w-1.5 rounded-full', online ? 'bg-emerald-400' : 'bg-white/30')} />
          {device.status}
        </span>
        <span className="text-white/40">
          ws <span className={connected ? 'text-emerald-400' : 'text-red-400'}>{connected ? 'live' : 'off'}</span>
        </span>
        {stats && (
          <>
            <span className="text-white/40">
              up <span className="text-white/80">{formatUptime(stats.uptime)}</span>
            </span>
            <span className="text-white/40">
              load{' '}
              <span className="text-white/80">
                {stats.load_avg.load1.toFixed(2)} {stats.load_avg.load5.toFixed(2)} {stats.load_avg.load15.toFixed(2)}
              </span>
            </span>
            <span className="text-white/40">
              proc <span className="text-white/80">{stats.processes}</span>
            </span>
          </>
        )}
        <span className="text-white/30">
          {device.platform} · agent {device.version || '?'} · seen {formatRelativeTime(device.last_seen)}
        </span>
      </header>

      {!stats ? (
        <div className="flex h-72 items-center justify-center rounded-lg border border-dashed border-white/10 text-sm text-white/35">
          {online ? 'waiting for the first stats frame…' : 'device is offline'}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
          <Panel title="cpu" className="lg:col-span-2">
            <div className="mb-2 flex items-end justify-between">
              <div className="flex items-baseline gap-2">
                <span
                  className={cn('text-4xl font-semibold leading-none', cpu > 85 ? 'text-red-400' : 'text-emerald-400')}
                >
                  {cpu.toFixed(0)}
                </span>
                <span className="text-sm text-white/35">%</span>
              </div>
              <div className="text-right text-[10px] leading-relaxed text-white/40">
                <div>
                  user <span className="text-white/70">{stats.cpu.user.toFixed(1)}%</span> · sys{' '}
                  <span className="text-white/70">{stats.cpu.system.toFixed(1)}%</span>
                </div>
                <div>
                  idle <span className="text-white/70">{stats.cpu.idle.toFixed(1)}%</span> · io{' '}
                  <span className="text-white/70">{stats.cpu.iowait.toFixed(1)}%</span>
                </div>
              </div>
            </div>
            <Sparkline values={cpuHistory} max={100} height={34} />
            <div className="mt-2 grid grid-cols-8 gap-x-2 gap-y-1 sm:grid-cols-12 lg:grid-cols-[repeat(16,minmax(0,1fr))]">
              {(stats.cpu.per_core ?? []).map((core, index) => (
                <div key={index} className="flex items-center gap-1">
                  <span className="w-4 shrink-0 text-right text-[9px] text-white/25">{index}</span>
                  <div className="h-1.5 flex-1 overflow-hidden rounded-sm bg-white/10">
                    <div
                      className={cn(
                        'h-full',
                        core > 85 ? 'bg-red-400' : core > 60 ? 'bg-amber-400' : 'bg-emerald-400/80'
                      )}
                      style={{ width: `${Math.min(100, core)}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </Panel>

          <Panel title="memory" accent="text-cyan-300">
            <div className="mb-2 flex items-baseline justify-between">
              <span className={cn('text-2xl font-semibold', mem > 85 ? 'text-red-400' : 'text-cyan-300')}>
                {mem.toFixed(0)}%
              </span>
              <span className="text-[10px] text-white/40">
                {formatBytes(stats.memory.used)} / {formatBytes(stats.memory.total)}
              </span>
            </div>
            <Bar value={mem} className="bg-cyan-400" />
            <Sparkline values={memHistory} max={100} height={26} color="#22d3ee" />
            <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-[10px]">
              {[
                ['available', formatBytes(stats.memory.available)],
                ['free', formatBytes(stats.memory.free)],
                ['cached', formatBytes(stats.memory.cached)],
                ['buffers', formatBytes(stats.memory.buffers)],
              ].map(([label, value]) => (
                <div key={label} className="flex justify-between">
                  <dt className="text-white/35">{label}</dt>
                  <dd className="text-white/70">{value}</dd>
                </div>
              ))}
            </dl>
            <div className="mt-2 border-t border-white/[0.06] pt-2">
              <div className="flex justify-between text-[10px]">
                <span className="text-white/35">swap</span>
                <span className="text-white/70">
                  {formatBytes(stats.swap.used)} / {formatBytes(stats.swap.total)}
                </span>
              </div>
              <Bar value={stats.swap.used_percent} className="mt-1 bg-fuchsia-400" />
            </div>
          </Panel>

          <Panel title="network" accent="text-sky-300" className="lg:col-span-2">
            <div className="mb-2 flex items-baseline justify-between">
              <div className="flex gap-4 text-sm">
                <span className="text-emerald-300">↓ {rate(stats.network.rx_rate)}</span>
                <span className="text-sky-300">↑ {rate(stats.network.tx_rate)}</span>
              </div>
              <span className="text-[10px] text-white/35">
                total {formatBytes(stats.network.total_rx)} / {formatBytes(stats.network.total_tx)}
              </span>
            </div>
            <Sparkline values={netHistory} height={30} color="#38bdf8" />
            <div className="mt-2 grid grid-cols-1 gap-x-4 gap-y-0.5 text-[10px] sm:grid-cols-2">
              {(stats.network.interfaces ?? []).slice(0, 6).map((iface) => (
                <div key={iface.name} className="flex items-center justify-between gap-2">
                  <span className={cn('truncate', iface.up ? 'text-white/70' : 'text-white/25')}>{iface.name}</span>
                  <span className="shrink-0 text-white/40">
                    ↓{rate(iface.rx_rate)} ↑{rate(iface.tx_rate)}
                  </span>
                </div>
              ))}
            </div>
          </Panel>

          <Panel title="disks" accent="text-amber-300">
            <div className="space-y-2">
              {stats.disks.slice(0, 8).map((disk) => (
                <div key={disk.mount_point}>
                  <div className="flex items-baseline justify-between text-[10px]">
                    <span className="truncate text-white/70">{disk.mount_point}</span>
                    <span className={disk.used_percent > 90 ? 'text-red-400' : 'text-white/40'}>
                      {disk.used_percent.toFixed(0)}% · {formatBytes(disk.free)} free
                    </span>
                  </div>
                  <Bar
                    value={disk.used_percent}
                    className={
                      disk.used_percent > 90
                        ? 'bg-red-400'
                        : disk.used_percent > 75
                          ? 'bg-amber-400'
                          : 'bg-amber-300/80'
                    }
                  />
                </div>
              ))}
              {stats.disks.length === 0 && <p className="text-[10px] text-white/30">no physical mounts</p>}
            </div>
          </Panel>

          <Panel title="processes" className="lg:col-span-3">
            <div className="overflow-x-auto">
              <table className="w-full border-collapse text-left text-[11px]">
                <thead className="text-[9px] uppercase tracking-wider text-white/30">
                  <tr className="border-b border-white/[0.06]">
                    <th className="py-1 pr-3 font-normal">pid</th>
                    <th className="py-1 pr-3 font-normal">user</th>
                    <th className="py-1 pr-3 text-right font-normal">cpu%</th>
                    <th className="py-1 pr-3 text-right font-normal">mem%</th>
                    <th className="py-1 pr-3 text-right font-normal">rss</th>
                    <th className="py-1 font-normal">command</th>
                  </tr>
                </thead>
                <tbody>
                  {processes.map((proc) => (
                    <tr key={proc.pid} className="border-b border-white/[0.03] hover:bg-white/[0.03]">
                      <td className="py-0.5 pr-3 text-white/40">{proc.pid}</td>
                      <td className="py-0.5 pr-3 text-white/60">{proc.user || '?'}</td>
                      <td
                        className={cn(
                          'py-0.5 pr-3 text-right tabular-nums',
                          proc.cpu > 50 ? 'text-red-400' : proc.cpu > 10 ? 'text-amber-300' : 'text-emerald-300'
                        )}
                      >
                        {proc.cpu.toFixed(1)}
                      </td>
                      <td className="py-0.5 pr-3 text-right tabular-nums text-cyan-300/80">{proc.memory.toFixed(1)}</td>
                      <td className="py-0.5 pr-3 text-right tabular-nums text-white/40">{formatBytes(proc.rss)}</td>
                      <td className="max-w-0 truncate py-0.5 text-white/70">{proc.command || proc.name}</td>
                    </tr>
                  ))}
                  {processes.length === 0 && (
                    <tr>
                      <td colSpan={6} className="py-3 text-center text-white/25">
                        waiting for process table…
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </Panel>
        </div>
      )}
    </div>
  )
}
