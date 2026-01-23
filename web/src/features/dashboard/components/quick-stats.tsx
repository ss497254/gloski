import { Server, Clock, Cpu, Activity } from 'lucide-react'
import { StatCard } from '@/shared/components'
import type { SystemStats } from '@/shared/lib/types'

interface QuickStatsProps {
  stats: SystemStats
}

export function QuickStats({ stats }: QuickStatsProps) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <StatCard
        title="Hostname"
        value={stats.hostname}
        subtitle={stats.platform}
        icon={Server}
        iconClassName="bg-blue-500/10"
      />
      <StatCard
        title="Uptime"
        value={formatUptime(stats.uptime)}
        subtitle={`Boot: ${formatDate(stats.boot_time)}`}
        icon={Clock}
        iconClassName="bg-green-500/10"
      />
      <StatCard
        title="Processes"
        value={stats.processes}
        subtitle={`${stats.goroutines} goroutines`}
        icon={Activity}
        iconClassName="bg-purple-500/10"
      />
      <StatCard
        title="CPU Model"
        value={`${stats.cpu.cores} cores`}
        subtitle={truncate(stats.cpu.model_name, 35)}
        icon={Cpu}
        iconClassName="bg-orange-500/10"
      />
    </div>
  )
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

function formatDate(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function truncate(str: string, length: number): string {
  if (str.length <= length) return str
  return str.slice(0, length) + '...'
}
