import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { useServersStore, getSortedServers, type Server } from '@/stores/servers'
import { createServerApi } from '@/services/api'
import type { SystemStats } from '@/lib/types'
import { cn } from '@/lib/utils'
import {
  Server as ServerIcon,
  Plus,
  Cpu,
  HardDrive,
  MemoryStick,
  Clock,
  Wifi,
  WifiOff,
  ShieldAlert,
  ChevronRight,
} from 'lucide-react'

interface ServerWithStats extends Server {
  stats?: SystemStats | null
  statsLoading?: boolean
  statsError?: string | null
}

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

function getUsageColor(percent: number): string {
  if (percent >= 90) return 'bg-red-500'
  if (percent >= 75) return 'bg-orange-500'
  if (percent >= 50) return 'bg-yellow-500'
  return 'bg-emerald-500'
}

function UsageBar({ value, color }: { value: number; color?: string }) {
  return (
    <div className="h-1.5 bg-muted rounded-full overflow-hidden">
      <div
        className={cn('h-full rounded-full transition-all duration-500', color || getUsageColor(value))}
        style={{ width: `${Math.min(value, 100)}%` }}
      />
    </div>
  )
}

function ServerCardSkeleton() {
  return (
    <Card className="h-full">
      <CardContent className="pt-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <Skeleton className="h-10 w-10 rounded-lg" />
            <div className="space-y-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-32" />
            </div>
          </div>
          <Skeleton className="h-5 w-16 rounded-full" />
        </div>
        <div className="space-y-4">
          <div className="space-y-2">
            <div className="flex justify-between">
              <Skeleton className="h-3 w-8" />
              <Skeleton className="h-3 w-12" />
            </div>
            <Skeleton className="h-1.5 w-full rounded-full" />
          </div>
          <div className="space-y-2">
            <div className="flex justify-between">
              <Skeleton className="h-3 w-12" />
              <Skeleton className="h-3 w-12" />
            </div>
            <Skeleton className="h-1.5 w-full rounded-full" />
          </div>
          <div className="space-y-2">
            <div className="flex justify-between">
              <Skeleton className="h-3 w-8" />
              <Skeleton className="h-3 w-20" />
            </div>
            <Skeleton className="h-1.5 w-full rounded-full" />
          </div>
        </div>
        <div className="flex items-center justify-between mt-4 pt-4 border-t">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-3 w-16" />
        </div>
      </CardContent>
    </Card>
  )
}

function ServerCard(props: ServerWithStats) {
  const { stats, statsLoading, statsError, ...server } = props

  const statusConfig = {
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
  }

  const status = statusConfig[server.status]
  const StatusIcon = status.icon

  if (statsLoading && !stats) {
    return <ServerCardSkeleton />
  }

  return (
    <Link to={`/servers/${server.id}`} className="block group">
      <Card className="h-full transition-all duration-200 hover:shadow-md hover:border-primary/30 group-hover:bg-accent/30">
        <CardContent className="pt-6">
          {/* Header */}
          <div className="flex items-start justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className={cn(
                'h-10 w-10 rounded-lg flex items-center justify-center transition-colors',
                server.status === 'online' ? 'bg-primary/10' : 'bg-muted'
              )}>
                <ServerIcon className={cn(
                  'h-5 w-5',
                  server.status === 'online' ? 'text-primary' : 'text-muted-foreground'
                )} />
              </div>
              <div>
                <h3 className="font-semibold text-sm group-hover:text-primary transition-colors">
                  {server.name}
                </h3>
                <p className="text-xs text-muted-foreground truncate max-w-[160px]">
                  {stats?.hostname || new URL(server.url).hostname}
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
                  <span className="font-medium tabular-nums">
                    {stats.cpu.usage_percent.toFixed(1)}%
                  </span>
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
                  <span className="font-medium tabular-nums">
                    {stats.memory.used_percent.toFixed(1)}%
                  </span>
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

export function DashboardPage() {
  const { servers, updateServer } = useServersStore()
  const [serversWithStats, setServersWithStats] = useState<ServerWithStats[]>([])

  const sortedServers = getSortedServers(servers)

  // Fetch stats for all servers
  useEffect(() => {
    const fetchAllStats = async () => {
      const results = await Promise.all(
        servers.map(async (server) => {
          if (!server.apiKey && !server.token) {
            return { ...server, stats: null, statsError: 'No authentication configured' }
          }

          try {
            const api = createServerApi(server)
            const stats = await api.stats()
            updateServer(server.id, { status: 'online' })
            return { ...server, status: 'online' as const, stats, statsError: null }
          } catch (err) {
            const isAuthError = err instanceof Error && err.message.includes('401')
            updateServer(server.id, { status: isAuthError ? 'unauthorized' : 'offline' })
            return {
              ...server,
              status: isAuthError ? 'unauthorized' as const : 'offline' as const,
              stats: null,
              statsError: err instanceof Error ? err.message : 'Unknown error',
            }
          }
        })
      )
      setServersWithStats(results)
    }

    // Initial fetch
    setServersWithStats(servers.map((s) => ({ ...s, statsLoading: true })))
    fetchAllStats()

    // Poll every 10 seconds
    const interval = setInterval(fetchAllStats, 10000)
    return () => clearInterval(interval)
  }, [servers, updateServer])

  // Merge sorted order with stats
  const displayServers = sortedServers.map((server) => {
    const withStats = serversWithStats.find((s) => s.id === server.id)
    return withStats || { ...server, statsLoading: true }
  })

  const onlineCount = serversWithStats.filter((s) => s.status === 'online').length
  const offlineCount = serversWithStats.filter((s) => s.status !== 'online' && s.status !== 'connecting').length

  return (
    <div className="h-full overflow-auto">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b">
        <div className="flex items-center justify-between px-6 py-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
            <p className="text-sm text-muted-foreground">
              {servers.length === 0 ? (
                'No servers configured'
              ) : (
                <span className="flex items-center gap-3">
                  <span>{servers.length} server{servers.length !== 1 ? 's' : ''}</span>
                  {onlineCount > 0 && (
                    <span className="flex items-center gap-1 text-emerald-600">
                      <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                      {onlineCount} online
                    </span>
                  )}
                  {offlineCount > 0 && (
                    <span className="flex items-center gap-1 text-red-600">
                      <span className="h-1.5 w-1.5 rounded-full bg-red-500" />
                      {offlineCount} offline
                    </span>
                  )}
                </span>
              )}
            </p>
          </div>
          <Button asChild>
            <Link to="/add-server">
              <Plus className="h-4 w-4 mr-2" />
              Add Server
            </Link>
          </Button>
        </div>
      </div>

      {/* Content */}
      <div className="p-6">
        {servers.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="h-20 w-20 rounded-2xl bg-primary/10 flex items-center justify-center mb-6">
              <ServerIcon className="h-10 w-10 text-primary" />
            </div>
            <h2 className="text-xl font-semibold mb-2">No servers configured</h2>
            <p className="text-muted-foreground mb-6 max-w-sm">
              Add your first server to start monitoring system resources, managing files, and running commands.
            </p>
            <Button asChild size="lg">
              <Link to="/add-server">
                <Plus className="h-4 w-4 mr-2" />
                Add Your First Server
              </Link>
            </Button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {displayServers.map((server) => (
              <ServerCard key={server.id} {...server} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
