import type { ProcessInfo, SystemStats } from '@/shared/api'
import { cn, formatBytes, formatUptime } from '@/shared/lib/utils'
import type { FleetDevice } from '@/shared/tower'
import { Activity, Cpu, HardDrive, MemoryStick, Network } from 'lucide-react'

interface BtopViewProps {
  device: FleetDevice
  stats: SystemStats | null
  processes: ProcessInfo[]
  connected: boolean
}

function Stat({
  icon: Icon,
  label,
  value,
  detail,
  tone,
}: {
  icon: typeof Cpu
  label: string
  value: string
  detail: string
  tone?: string
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Icon className={cn('h-4 w-4', tone)} />
        {label}
      </div>
      <p className={cn('mt-2 text-2xl font-semibold tabular-nums', tone ?? 'text-foreground')}>{value}</p>
      <p className="mt-1 truncate text-[11px] text-muted-foreground">{detail}</p>
    </div>
  )
}

function usageTone(percent: number): string {
  if (percent > 85) return 'text-destructive'
  if (percent > 65) return 'text-warning'
  return 'text-success'
}

export function BtopView({ device, stats, processes, connected }: BtopViewProps) {
  if (!stats) {
    return (
      <div className="flex h-64 items-center justify-center rounded-lg border border-dashed border-border text-sm text-muted-foreground">
        {device.status === 'online' ? 'Waiting for device statistics…' : 'Device is offline'}
      </div>
    )
  }

  const cpu = stats.cpu.usage_percent
  const memory = stats.memory.used_percent

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Activity className={cn('h-3.5 w-3.5', connected ? 'text-success' : 'text-muted-foreground')} />
        <span>{connected ? 'Live metrics' : 'Reconnecting…'}</span>
        <span className="text-muted-foreground/50">·</span>
        <span>{stats.hostname}</span>
        <span className="text-muted-foreground/50">·</span>
        <span>uptime {formatUptime(stats.uptime)}</span>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Stat
          icon={Cpu}
          label="CPU"
          value={`${cpu.toFixed(0)}%`}
          detail={`${stats.cpu.cores} cores · load ${stats.load_avg.load1.toFixed(2)}`}
          tone={usageTone(cpu)}
        />
        <Stat
          icon={MemoryStick}
          label="Memory"
          value={`${memory.toFixed(0)}%`}
          detail={`${formatBytes(stats.memory.used)} / ${formatBytes(stats.memory.total)}`}
          tone={usageTone(memory)}
        />
        <Stat
          icon={Network}
          label="Network"
          value={`↓ ${formatBytes(stats.network.rx_rate)}/s`}
          detail={`↑ ${formatBytes(stats.network.tx_rate)}/s`}
          tone="text-info"
        />
        <Stat
          icon={HardDrive}
          label="Disks"
          value={`${stats.disks.length}`}
          detail={`${stats.processes} running processes`}
        />
      </div>

      <section className="overflow-hidden rounded-lg border border-border bg-card">
        <header className="border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold text-foreground">Top processes</h2>
        </header>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="border-b border-border text-[10px] uppercase tracking-wider text-muted-foreground">
              <tr>
                <th className="px-4 py-2 font-medium">pid</th>
                <th className="px-4 py-2 font-medium">user</th>
                <th className="px-4 py-2 text-right font-medium">cpu%</th>
                <th className="px-4 py-2 text-right font-medium">mem%</th>
                <th className="px-4 py-2 text-right font-medium">rss</th>
                <th className="px-4 py-2 font-medium">command</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {processes.slice(0, 10).map((process) => (
                <tr key={process.pid} className="hover:bg-muted/40">
                  <td className="px-4 py-2 font-mono text-muted-foreground">{process.pid}</td>
                  <td className="px-4 py-2 text-foreground/80">{process.user || '?'}</td>
                  <td className={cn('px-4 py-2 text-right font-mono tabular-nums', usageTone(process.cpu))}>
                    {process.cpu.toFixed(1)}
                  </td>
                  <td className="px-4 py-2 text-right font-mono tabular-nums text-info">{process.memory.toFixed(1)}</td>
                  <td className="px-4 py-2 text-right font-mono text-muted-foreground">{formatBytes(process.rss)}</td>
                  <td className="max-w-0 truncate px-4 py-2 text-foreground/90">{process.command || process.name}</td>
                </tr>
              ))}
              {processes.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-4 text-center text-muted-foreground">
                    No process data yet
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section className="overflow-hidden rounded-lg border border-border bg-card">
        <header className="border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold text-foreground">Disks</h2>
        </header>
        <div className="divide-y divide-border/60">
          {stats.disks.slice(0, 6).map((disk) => (
            <div key={disk.mount_point} className="flex items-center gap-4 px-4 py-2.5 text-xs">
              <span className="w-32 shrink-0 truncate font-mono text-foreground/80">{disk.mount_point}</span>
              <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                <div
                  className={cn('h-full rounded-full', disk.used_percent > 85 ? 'bg-destructive' : 'bg-success')}
                  style={{ width: `${Math.min(100, disk.used_percent)}%` }}
                />
              </div>
              <span className="w-10 shrink-0 text-right font-mono tabular-nums text-muted-foreground">
                {disk.used_percent.toFixed(0)}%
              </span>
              <span className="w-24 shrink-0 text-right font-mono text-muted-foreground">
                {formatBytes(disk.free)} free
              </span>
            </div>
          ))}
          {stats.disks.length === 0 && <p className="px-4 py-4 text-xs text-muted-foreground">No disks reported</p>}
        </div>
      </section>
    </div>
  )
}
