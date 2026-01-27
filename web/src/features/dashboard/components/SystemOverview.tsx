import { ProgressRing } from '@/shared/components'
import type { SystemStats } from '@/shared/lib/types'
import { Activity, Cpu, HardDrive, MemoryStick } from 'lucide-react'

interface SystemOverviewProps {
  stats: SystemStats
}

export function SystemOverview({ stats }: SystemOverviewProps) {
  const metrics = [
    {
      label: 'CPU',
      value: stats.cpu.usage_percent,
      icon: Cpu,
      detail: `${stats.cpu.cores} cores`,
    },
    {
      label: 'Memory',
      value: stats.memory.used_percent,
      icon: MemoryStick,
      detail: formatBytes(stats.memory.used) + ' / ' + formatBytes(stats.memory.total),
    },
    {
      label: 'Disk',
      value: stats.disks[0]?.used_percent || 0,
      icon: HardDrive,
      detail: stats.disks[0] ? formatBytes(stats.disks[0].used) + ' / ' + formatBytes(stats.disks[0].total) : 'N/A',
    },
    {
      label: 'Load',
      value: (stats.load_avg.load1 / stats.cpu.cores) * 100,
      icon: Activity,
      detail: `${stats.load_avg.load1.toFixed(2)} / ${stats.load_avg.load5.toFixed(2)} / ${stats.load_avg.load15.toFixed(2)}`,
    },
  ]

  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <h2 className="text-lg font-semibold mb-6">System Overview</h2>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-8">
        {metrics.map((metric) => (
          <div key={metric.label} className="flex flex-col items-center text-center">
            <ProgressRing value={metric.value} size={100} strokeWidth={8} label={metric.label} />
            <p className="mt-3 text-xs text-muted-foreground">{metric.detail}</p>
          </div>
        ))}
      </div>
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
