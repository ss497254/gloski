import type { ProcessInfo, SystemStats } from '@/shared/api'
import { ProgressBar, StatusDot, toneSurface } from '@/shared/components'
import { cn, formatBytes, formatUptime } from '@/shared/lib/utils'
import type { FleetDevice } from '@/shared/tower'
import { Badge } from '@/ui/badge'
import { Card } from '@/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/ui/table'
import type { StatusTone } from '@/shared/components'
import { Cpu, HardDrive, MemoryStick, Network } from 'lucide-react'

const toneText: Record<StatusTone, string> = {
  success: 'text-success',
  warning: 'text-warning',
  info: 'text-info',
  destructive: 'text-destructive',
  muted: 'text-muted-foreground',
}

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
  tone?: StatusTone
}) {
  return (
    <Card className="gap-0 p-4">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <span className={cn('flex size-7 items-center justify-center rounded-lg', tone ? toneSurface[tone] : 'bg-primary/10 text-primary')}>
          <Icon className="size-3.5" />
        </span>
        {label}
      </div>
      <p className={cn('mt-2 text-2xl font-semibold tabular-nums', tone ? toneText[tone] : 'text-foreground')}>{value}</p>
      <p className="mt-1 truncate text-[11px] text-muted-foreground">{detail}</p>
    </Card>
  )
}

function usageTone(percent: number): StatusTone {
  if (percent > 85) return 'destructive'
  if (percent > 65) return 'warning'
  return 'success'
}

export function BtopView({ device, stats, processes, connected }: BtopViewProps) {
  if (!stats) {
    return (
      <div className="flex h-64 items-center justify-center rounded-xl border border-dashed border-border text-sm text-muted-foreground">
        {device.status === 'online' ? 'Waiting for device statistics…' : 'Device is offline'}
      </div>
    )
  }

  const cpu = stats.cpu.usage_percent
  const memory = stats.memory.used_percent

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Badge variant={connected ? 'success' : 'secondary'} className="gap-1.5">
          <StatusDot status={connected ? 'online' : 'offline'} className="size-1.5" />
          {connected ? 'Live metrics' : 'Reconnecting…'}
        </Badge>
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
          tone="info"
        />
        <Stat
          icon={HardDrive}
          label="Disks"
          value={`${stats.disks.length}`}
          detail={`${stats.processes} running processes`}
        />
      </div>

      <Card className="gap-0 overflow-hidden py-0">
        <header className="border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold text-foreground">Top processes</h2>
        </header>
        <Table className="text-xs">
          <TableHeader className="bg-muted/40 hover:bg-muted/40">
            <TableRow>
              <TableHead className="px-4 py-2 text-[10px] uppercase tracking-wider text-muted-foreground">pid</TableHead>
              <TableHead className="px-4 py-2 text-[10px] uppercase tracking-wider text-muted-foreground">user</TableHead>
              <TableHead className="px-4 py-2 text-right text-[10px] uppercase tracking-wider text-muted-foreground">
                cpu%
              </TableHead>
              <TableHead className="px-4 py-2 text-right text-[10px] uppercase tracking-wider text-muted-foreground">
                mem%
              </TableHead>
              <TableHead className="px-4 py-2 text-right text-[10px] uppercase tracking-wider text-muted-foreground">
                rss
              </TableHead>
              <TableHead className="px-4 py-2 text-[10px] uppercase tracking-wider text-muted-foreground">command</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody className="divide-y divide-border/60">
            {processes.slice(0, 10).map((process) => (
              <TableRow key={process.pid} className="hover:bg-muted/50">
                <TableCell className="px-4 py-2 font-mono text-muted-foreground">{process.pid}</TableCell>
                <TableCell className="px-4 py-2 text-foreground/80">{process.user || '?'}</TableCell>
                <TableCell className={cn('px-4 py-2 text-right font-mono tabular-nums', toneText[usageTone(process.cpu)])}>
                  {process.cpu.toFixed(1)}
                </TableCell>
                <TableCell className="px-4 py-2 text-right font-mono tabular-nums text-info">
                  {process.memory.toFixed(1)}
                </TableCell>
                <TableCell className="px-4 py-2 text-right font-mono text-muted-foreground">
                  {formatBytes(process.rss)}
                </TableCell>
                <TableCell className="max-w-0 truncate px-4 py-2 text-foreground/90">{process.command || process.name}</TableCell>
              </TableRow>
            ))}
            {processes.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="px-4 py-4 text-center text-muted-foreground">
                  No process data yet
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </Card>

      <Card className="gap-0 overflow-hidden py-0">
        <header className="border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold text-foreground">Disks</h2>
        </header>
        <div className="divide-y divide-border/60">
          {stats.disks.slice(0, 6).map((disk) => (
            <div key={disk.mount_point} className="flex items-center gap-4 px-4 py-2.5 text-xs">
              <span className="w-32 shrink-0 truncate font-mono text-foreground/80">{disk.mount_point}</span>
              <ProgressBar value={disk.used_percent} size="sm" variant="auto" className="flex-1" />
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
      </Card>
    </div>
  )
}
