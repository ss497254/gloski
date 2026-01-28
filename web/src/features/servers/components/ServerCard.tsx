import { cn } from '@/shared/lib/utils'
import { Badge } from '@/ui/badge'
import { Card, CardContent } from '@/ui/card'
import {
  ChevronRight,
  Clock,
  Cpu,
  HardDrive,
  MemoryStick,
  Server as ServerIcon,
  ShieldAlert,
  Wifi,
  WifiOff,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import type { ServerWithStats } from '../../dashboard/context'
import { ServerCardSkeleton } from './ServerCardSkeleton'
import { UsageBar } from './UsageBar'

// ─────────────────────────────────────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function getHostname(url: string): string {
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Status config
// ─────────────────────────────────────────────────────────────────────────────

const SERVER_STATUS_CONFIG = {
  online: {
    icon: Wifi,
    label: 'Online',
    variant: 'default' as const,
    className: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20',
  },
  connecting: {
    icon: Wifi,
    label: 'Connecting',
    variant: 'secondary' as const,
    className: 'bg-yellow-500/10 text-yellow-600 border-yellow-500/20 animate-pulse',
  },
  unauthorized: {
    icon: ShieldAlert,
    label: 'Auth Error',
    variant: 'destructive' as const,
    className: 'bg-orange-500/10 text-orange-600 border-orange-500/20',
  },
  offline: {
    icon: WifiOff,
    label: 'Offline',
    variant: 'destructive' as const,
    className: 'bg-red-500/10 text-red-600 border-red-500/20',
  },
} as const

// ─────────────────────────────────────────────────────────────────────────────
// Component
// ─────────────────────────────────────────────────────────────────────────────

interface ServerCardProps extends ServerWithStats {}

export function ServerCard(props: ServerCardProps) {
  const { stats, statsLoading, statsError, ...server } = props

  const status = SERVER_STATUS_CONFIG[server.status]
  const StatusIcon = status.icon

  if (statsLoading && !stats) {
    return <ServerCardSkeleton />
  }

  return (
    <Link to={`/servers/${server.id}`} className="block group">
      <Card className="h-full transition-all duration-200 hover:shadow-md hover:border-primary/30 group-hover:bg-accent/30">
        <CardContent>
          {/* Header */}
          <div className="flex items-start justify-between mb-4">
            <div className="flex items-center gap-3">
              <div
                className={cn(
                  'h-10 w-10 rounded-lg flex items-center justify-center transition-colors',
                  server.status === 'online' ? 'bg-primary/10' : 'bg-muted'
                )}
              >
                <ServerIcon
                  className={cn('h-5 w-5', server.status === 'online' ? 'text-primary' : 'text-muted-foreground')}
                />
              </div>
              <div>
                <h3 className="font-semibold text-sm group-hover:text-primary transition-colors">{server.name}</h3>
                <p className="text-xs text-muted-foreground truncate max-w-40">
                  {stats?.hostname || getHostname(server.url)}
                </p>
              </div>
            </div>
            <Badge variant="outline" className={cn('text-xs font-medium', status.className)}>
              <StatusIcon className="h-3 w-3 mr-1" />
              {status.label}
            </Badge>
          </div>

          {/* Stats */}
          {stats ? (
            <div className="space-y-3">
              {/* CPU */}
              <div className="space-y-1.5">
                <div className="flex items-center justify-between text-xs">
                  <span className="flex items-center gap-1.5 text-muted-foreground">
                    <Cpu className="h-3 w-3" />
                    CPU
                  </span>
                  <span className="font-medium tabular-nums">{stats.cpu.usage_percent.toFixed(1)}%</span>
                </div>
                <UsageBar value={stats.cpu.usage_percent} />
              </div>

              {/* Memory */}
              <div className="space-y-1.5">
                <div className="flex items-center justify-between text-xs">
                  <span className="flex items-center gap-1.5 text-muted-foreground">
                    <MemoryStick className="h-3 w-3" />
                    Memory
                  </span>
                  <span className="font-medium tabular-nums">{stats.memory.used_percent.toFixed(1)}%</span>
                </div>
                <UsageBar value={stats.memory.used_percent} />
              </div>

              {/* Disk */}
              {stats.disks[0] && (
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between text-xs">
                    <span className="flex items-center gap-1.5 text-muted-foreground">
                      <HardDrive className="h-3 w-3" />
                      Disk
                    </span>
                    <span className="font-medium tabular-nums">
                      {formatBytes(stats.disks[0].used)} / {formatBytes(stats.disks[0].total)}
                    </span>
                  </div>
                  <UsageBar value={stats.disks[0].used_percent} />
                </div>
              )}

              {/* Footer */}
              <div className="flex items-center justify-between pt-3 border-t text-xs text-muted-foreground">
                <span className="flex items-center gap-1.5">
                  <Clock className="h-3 w-3" />
                  Up {formatUptime(stats.uptime)}
                </span>
                <span className="flex items-center gap-1 text-primary opacity-0 group-hover:opacity-100 transition-opacity">
                  View details
                  <ChevronRight className="h-3 w-3" />
                </span>
              </div>
            </div>
          ) : statsError ? (
            <div className="py-6 text-center">
              <div className="h-10 w-10 rounded-full bg-destructive/10 flex items-center justify-center mx-auto mb-2">
                <WifiOff className="h-5 w-5 text-destructive" />
              </div>
              <p className="text-xs text-muted-foreground">
                {server.status === 'unauthorized' ? 'Invalid API key' : 'Unable to connect'}
              </p>
            </div>
          ) : (
            <div className="py-6 text-center">
              <div className="h-10 w-10 rounded-full bg-muted flex items-center justify-center mx-auto mb-2">
                <ServerIcon className="h-5 w-5 text-muted-foreground" />
              </div>
              <p className="text-xs text-muted-foreground">
                {server.status === 'offline' ? 'Server offline' : 'Loading...'}
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </Link>
  )
}
