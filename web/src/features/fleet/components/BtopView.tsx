import type { ProcessInfo, SystemStats } from '@/shared/api'
import { cn, formatBytes, formatRelativeTime, formatUptime } from '@/shared/lib/utils'
import type { FleetDevice } from '@/shared/tower'
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
  accent = 'text-success',
}: {
  title: string
  children: ReactNode
  className?: string
  accent?: string
}) {
  return (
    <section className={cn('overflow-hidden rounded-lg border border-border bg-card text-card-foreground', className)}>
      <header className="flex items-center justify-between border-b border-border px-3 py-2">
        <h2 className={cn('text-[10px] font-semibold uppercase tracking-[0.18em]', accent)}>{title}</h2>
      </header>
      <div className="p-4">{children}</div>
    </section>
  )
}

function Bar({ value, className }: { value: number; className?: string }) {
  return (
    <div className="h-2 overflow-hidden rounded-sm bg-muted">
      <div
        className={cn('h-full transition-[width] duration-500', className ?? 'bg-success')}
        style={{ width: `${Math.min(100, Math.max(0, value))}%` }}
      />
    </div>
  )
}

function valueColor(percent: number): string {
  if (percent > 85) return 'text-destructive'
  if (percent > 65) return 'text-warning'
  return 'text-success'
}

function barColor(percent: number): string {
  if (percent > 85) return 'bg-destructive'
  if (percent > 65) return 'bg-warning'
  return 'bg-success'
}

export function BtopView({ device, stats, processes, connected }: BtopViewProps) {
  const [cpuHistory, setCpuHistory] = useState<number[]>([])
  const [netHistory, setNetHistory] = useState<number[]>([])
  const [memHistory, setMemHistory] = useState<number[]>([])

  /* eslint-disable react-hooks/set-state-in-effect -- roll history forward per stats frame */
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
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-x-5 gap-y-1 text-xs text-muted-foreground">
        <span className="inline-flex items-center gap-1.5 rounded-full border border-border px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider">
          <span className={cn('h-1.5 w-1.5 rounded-full', online ? 'bg-success' : 'bg-muted-foreground/50')} />
          <span className={online ? 'text-success' : undefined}>{device.status}</span>
        </span>
        <span>
          stream <span className={connected ? 'text-success' : 'text-destructive'}>{connected ? 'live' : 'off'}</span>
        </span>
        {stats && (
          <>
            <span>
              uptime <span className="text-foreground">{formatUptime(stats.uptime)}</span>
            </span>
            <span>
              load{' '}
              <span className="font-mono text-foreground">
                {stats.load_avg.load1.toFixed(2)} {stats.load_avg.load5.toFixed(2)} {stats.load_avg.load15.toFixed(2)}
              </span>
            </span>
            <span>
              processes <span className="font-mono text-foreground">{stats.processes}</span>
            </span>
          </>
        )}
        <span className="text-muted-foreground/70">
          {device.platform} · agent {device.version || '?'} · seen {formatRelativeTime(device.last_seen)}
        </span>
      </div>

      {!stats ? (
        <div className="flex h-72 items-center justify-center rounded-lg border border-dashed border-border text-sm text-muted-foreground">
          {online ? 'Waiting for the first stats frame…' : 'Device is offline'}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
          <Panel title="CPU" className="lg:col-span-2">
            <div className="mb-2 flex items-end justify-between">
              <div className="flex items-baseline gap-2">
                <span className={cn('font-mono text-4xl font-semibold leading-none tabular-nums', valueColor(cpu))}>
                  {cpu.toFixed(0)}
                </span>
                <span className="text-sm text-muted-foreground">%</span>
              </div>
              <div className="text-right text-[11px] leading-relaxed text-muted-foreground">
                <div>
                  user <span className="font-mono text-foreground">{stats.cpu.user.toFixed(1)}%</span> · sys{' '}
                  <span className="font-mono text-foreground">{stats.cpu.system.toFixed(1)}%</span>
                </div>
                <div>
                  idle <span className="font-mono text-foreground">{stats.cpu.idle.toFixed(1)}%</span> · io{' '}
                  <span className="font-mono text-foreground">{stats.cpu.iowait.toFixed(1)}%</span>
                </div>
              </div>
            </div>
            <Sparkline values={cpuHistory} max={100} height={34} className={valueColor(cpu)} />
            <div className="mt-3 grid grid-cols-8 gap-x-2 gap-y-1 sm:grid-cols-12 lg:grid-cols-[repeat(16,minmax(0,1fr))]">
              {(stats.cpu.per_core ?? []).map((core, index) => (
                <div key={index} className="flex items-center gap-1">
                  <span className="w-4 shrink-0 text-right font-mono text-[9px] text-muted-foreground/60">{index}</span>
                  <div className="h-1.5 flex-1 overflow-hidden rounded-sm bg-muted">
                    <div className={cn('h-full', barColor(core))} style={{ width: `${Math.min(100, core)}%` }} />
                  </div>
                </div>
              ))}
            </div>
          </Panel>

          <Panel title="Memory" accent="text-info">
            <div className="mb-2 flex items-baseline justify-between">
              <span
                className={cn(
                  'font-mono text-2xl font-semibold tabular-nums',
                  mem > 85 ? 'text-destructive' : 'text-info'
                )}
              >
                {mem.toFixed(0)}%
              </span>
              <span className="font-mono text-[11px] text-muted-foreground">
                {formatBytes(stats.memory.used)} / {formatBytes(stats.memory.total)}
              </span>
            </div>
            <Bar value={mem} className="bg-info" />
            <Sparkline values={memHistory} max={100} height={26} className="mt-1 text-info" />
            <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-[11px]">
              {[
                ['available', formatBytes(stats.memory.available)],
                ['free', formatBytes(stats.memory.free)],
                ['cached', formatBytes(stats.memory.cached)],
                ['buffers', formatBytes(stats.memory.buffers)],
              ].map(([label, value]) => (
                <div key={label} className="flex justify-between gap-2">
                  <dt className="text-muted-foreground">{label}</dt>
                  <dd className="font-mono text-foreground">{value}</dd>
                </div>
              ))}
            </dl>
            <div className="mt-3 border-t border-border pt-2">
              <div className="flex justify-between text-[11px]">
                <span className="text-muted-foreground">swap</span>
                <span className="font-mono text-foreground">
                  {formatBytes(stats.swap.used)} / {formatBytes(stats.swap.total)}
                </span>
              </div>
              <Bar value={stats.swap.used_percent} className="mt-1 bg-warning" />
            </div>
          </Panel>

          <Panel title="Network" accent="text-info" className="lg:col-span-2">
            <div className="mb-2 flex items-baseline justify-between">
              <div className="flex gap-4 font-mono text-sm tabular-nums">
                <span className="text-success">↓ {rate(stats.network.rx_rate)}</span>
                <span className="text-info">↑ {rate(stats.network.tx_rate)}</span>
              </div>
              <span className="font-mono text-[11px] text-muted-foreground">
                total {formatBytes(stats.network.total_rx)} / {formatBytes(stats.network.total_tx)}
              </span>
            </div>
            <Sparkline values={netHistory} height={30} className="text-info" />
            <div className="mt-3 grid grid-cols-1 gap-x-4 gap-y-0.5 text-[11px] sm:grid-cols-2">
              {(stats.network.interfaces ?? []).slice(0, 6).map((iface) => (
                <div key={iface.name} className="flex items-center justify-between gap-2">
                  <span className={cn('truncate', iface.up ? 'text-foreground' : 'text-muted-foreground/50')}>
                    {iface.name}
                  </span>
                  <span className="shrink-0 font-mono text-muted-foreground">
                    ↓{rate(iface.rx_rate)} ↑{rate(iface.tx_rate)}
                  </span>
                </div>
              ))}
            </div>
          </Panel>

          <Panel title="Disks" accent="text-warning">
            <div className="space-y-2">
              {stats.disks.slice(0, 8).map((disk) => (
                <div key={disk.mount_point}>
                  <div className="flex items-baseline justify-between text-[11px]">
                    <span className="truncate text-foreground">{disk.mount_point}</span>
                    <span
                      className={cn('font-mono', disk.used_percent > 90 ? 'text-destructive' : 'text-muted-foreground')}
                    >
                      {disk.used_percent.toFixed(0)}% · {formatBytes(disk.free)} free
                    </span>
                  </div>
                  <Bar value={disk.used_percent} className={barColor(disk.used_percent)} />
                </div>
              ))}
              {stats.disks.length === 0 && <p className="text-xs text-muted-foreground">No physical mounts</p>}
            </div>
          </Panel>

          <Panel title="Processes" className="lg:col-span-3">
            <div className="overflow-x-auto">
              <table className="w-full border-collapse text-left text-[11px]">
                <thead className="text-[10px] uppercase tracking-wider text-muted-foreground">
                  <tr className="border-b border-border">
                    <th className="py-1 pr-3 font-medium">pid</th>
                    <th className="py-1 pr-3 font-medium">user</th>
                    <th className="py-1 pr-3 text-right font-medium">cpu%</th>
                    <th className="py-1 pr-3 text-right font-medium">mem%</th>
                    <th className="py-1 pr-3 text-right font-medium">rss</th>
                    <th className="py-1 font-medium">command</th>
                  </tr>
                </thead>
                <tbody>
                  {processes.map((proc) => (
                    <tr key={proc.pid} className="border-b border-border/60 hover:bg-muted/40">
                      <td className="py-1 pr-3 font-mono text-muted-foreground">{proc.pid}</td>
                      <td className="py-1 pr-3 text-foreground/80">{proc.user || '?'}</td>
                      <td
                        className={cn(
                          'py-1 pr-3 text-right font-mono tabular-nums',
                          proc.cpu > 50 ? 'text-destructive' : proc.cpu > 10 ? 'text-warning' : 'text-success'
                        )}
                      >
                        {proc.cpu.toFixed(1)}
                      </td>
                      <td className="py-1 pr-3 text-right font-mono tabular-nums text-info">
                        {proc.memory.toFixed(1)}
                      </td>
                      <td className="py-1 pr-3 text-right font-mono text-muted-foreground">{formatBytes(proc.rss)}</td>
                      <td className="max-w-0 truncate py-1 text-foreground/90">{proc.command || proc.name}</td>
                    </tr>
                  ))}
                  {processes.length === 0 && (
                    <tr>
                      <td colSpan={6} className="py-3 text-center text-muted-foreground">
                        Waiting for the process table…
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
