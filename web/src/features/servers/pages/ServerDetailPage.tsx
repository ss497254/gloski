import { DiskUsage, MemoryWidget, NetworkStatsWidget, QuickStats, SystemOverview } from '@/features/servers/components'
import { EmptyState } from '@/shared/components'
import type { ServerStatus, SystemStats } from '@/shared/lib/types'
import { cn } from '@/shared/lib/utils'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/ui/card'
import { Skeleton } from '@/ui/skeleton'
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  CircleDashed,
  MonitorCog,
  RefreshCw,
  ShieldAlert,
  WifiOff,
} from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useServersStore } from '../stores/servers'

function ServerDetailSkeleton() {
  return (
    <div className="h-full overflow-auto">
      {/* Header Skeleton */}
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b">
        <div className="flex items-center justify-between px-6 py-4">
          <div className="space-y-2">
            <Skeleton className="h-7 w-48" />
            <Skeleton className="h-4 w-64" />
          </div>
          <div className="flex items-center gap-2">
            {[1, 2, 3, 4].map((i) => (
              <Skeleton key={i} className="h-9 w-24" />
            ))}
            <Skeleton className="h-9 w-24" />
          </div>
        </div>
      </div>

      {/* Content Skeleton */}
      <div className="p-6 space-y-6">
        {/* Quick Stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i}>
              <CardContent className="pt-6">
                <div className="flex items-center gap-4">
                  <Skeleton className="h-12 w-12 rounded-xl" />
                  <div className="space-y-2 flex-1">
                    <Skeleton className="h-3 w-16" />
                    <Skeleton className="h-6 w-24" />
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        {/* System Overview */}
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-32" />
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-3 gap-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="space-y-2">
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-3 w-full" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Widgets */}
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
          {[1, 2, 3].map((i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-5 w-24" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-40 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </div>
  )
}

export function ServerDetailPage() {
  const { serverId } = useParams<{ serverId: string }>()
  const navigate = useNavigate()
  const getServer = useServersStore((state) => state.getServer)
  const updateServer = useServersStore((state) => state.updateServer)

  const server = serverId ? getServer(serverId) : undefined

  const [stats, setStats] = useState<SystemStats | null>(null)
  const [serverStatus, setServerStatus] = useState<ServerStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const fetchInProgressRef = useRef(false)

  const fetchStats = useCallback(async () => {
    // Get fresh server data from store
    const currentServer = serverId ? useServersStore.getState().getServer(serverId) : undefined
    if (!currentServer || fetchInProgressRef.current) return

    if (!currentServer.apiKey && !currentServer.token) {
      setError('No authentication configured. Add an API key to connect.')
      setLoading(false)
      return
    }

    try {
      fetchInProgressRef.current = true
      const client = currentServer.getClient()
      // Fetch both stats and status in parallel
      const [statsData, statusData] = await Promise.all([client.system.getStats(), client.system.getStatus()])
      setStats(statsData)
      setServerStatus(statusData)
      setError(null)
      // Only update status if it changed
      if (currentServer.status !== 'online') {
        updateServer(currentServer.id, { status: 'online' })
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      setError(message)
      const isAuthError = message.includes('401') || message.includes('Unauthorized')
      const newStatus = isAuthError ? 'unauthorized' : 'offline'
      // Only update status if it changed
      if (currentServer.status !== newStatus) {
        updateServer(currentServer.id, { status: newStatus })
      }
    } finally {
      setLoading(false)
      fetchInProgressRef.current = false
    }
  }, [serverId, updateServer])

  useEffect(() => {
    if (!server) {
      navigate('/')
      return
    }

    fetchStats()

    // Poll every 3 seconds
    const interval = setInterval(fetchStats, 3000)
    return () => clearInterval(interval)
  }, [serverId, server, fetchStats, navigate])

  if (!server) {
    return null
  }

  // Initial loading state with skeleton
  if (loading && !stats) {
    return <ServerDetailSkeleton />
  }

  // Error state
  if (error && !stats) {
    const isAuthError = error.includes('401') || error.includes('Unauthorized') || error.includes('API key')
    const isOffline = error.includes('fetch') || error.includes('network') || error.includes('ECONNREFUSED')

    return (
      <div className="h-full flex flex-col">
        {/* Header */}
        <div className="border-b px-6 py-4">
          <h1 className="text-2xl font-bold tracking-tight">{server.name}</h1>
          <p className="text-sm text-muted-foreground">{server.url}</p>
        </div>

        {/* Error Content */}
        <div className="flex-1 flex items-center justify-center p-6">
          <EmptyState
            icon={isAuthError ? ShieldAlert : WifiOff}
            title={isAuthError ? 'Authentication Failed' : 'Connection Failed'}
            description={
              isAuthError
                ? 'The server rejected the API key. Please check your credentials and try again.'
                : isOffline
                  ? 'Unable to reach the server. Please check if the server is running and the URL is correct.'
                  : error
            }
            variant={isAuthError ? 'warning' : 'error'}
            action={
              <div className="flex items-center gap-3">
                <Button onClick={fetchStats} variant="outline">
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Try Again
                </Button>
                <Button variant="ghost" onClick={() => navigate('/')}>
                  Back to Dashboard
                </Button>
              </div>
            }
          />
        </div>
      </div>
    )
  }

  if (!stats) {
    return null
  }

  const quickActions = [{ to: `/servers/${serverId}/os`, icon: MonitorCog, label: 'OS' }]

  const getHealthCheckIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle2 className="h-4 w-4 text-emerald-500" />
      case 'unhealthy':
        return <AlertCircle className="h-4 w-4 text-red-500" />
      case 'warning':
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />
      default:
        return <CircleDashed className="h-4 w-4 text-muted-foreground" />
    }
  }

  return (
    <div className="h-full overflow-auto">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-backdrop-filter:bg-background/60 border-b">
        <div className="flex items-center justify-between px-6 h-18">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight">{server.name}</h1>
              <Badge
                variant="outline"
                className={cn(
                  serverStatus?.status === 'healthy'
                    ? 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20'
                    : 'bg-yellow-500/10 text-yellow-600 border-yellow-500/20'
                )}
              >
                {serverStatus?.status === 'healthy'
                  ? 'Healthy'
                  : serverStatus?.status === 'degraded'
                    ? 'Degraded'
                    : 'Online'}
              </Badge>
              {serverStatus?.version && (
                <Badge variant="secondary" className="text-xs">
                  v{serverStatus.version}
                </Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              {stats.hostname} &middot; {server.url}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {/* Quick Actions */}
            {quickActions.map((action) => (
              <Button key={action.to} variant="outline" size="sm" asChild>
                <Link to={action.to}>
                  <action.icon className="h-4 w-4 mr-2" />
                  {action.label}
                </Link>
              </Button>
            ))}
            <Button variant="outline" size="sm" onClick={fetchStats} disabled={loading}>
              <RefreshCw className={cn('h-4 w-4 mr-2', loading && 'animate-spin')} />
              Refresh
            </Button>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-6 space-y-6">
        {/* Quick Stats */}
        <QuickStats stats={stats} />

        {/* System Overview */}
        <SystemOverview stats={stats} />

        {/* Detailed Widgets */}
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
          <MemoryWidget memory={stats.memory} swap={stats.swap} />
          <DiskUsage disks={stats.disks} />
          <NetworkStatsWidget network={stats.network} />
        </div>

        {/* Server Status & Health Checks */}
        {serverStatus && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Health Checks */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Health Checks</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {Object.entries(serverStatus.checks).map(([name, check]) => (
                    <div key={name} className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        {getHealthCheckIcon(check.status)}
                        <span className="text-sm font-medium capitalize">{name}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge
                          variant="outline"
                          className={cn(
                            'text-xs capitalize',
                            check.status === 'healthy' && 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20',
                            check.status === 'unhealthy' && 'bg-red-500/10 text-red-600 border-red-500/20',
                            check.status === 'warning' && 'bg-yellow-500/10 text-yellow-600 border-yellow-500/20',
                            check.status === 'unavailable' && 'bg-gray-500/10 text-gray-600 border-gray-500/20'
                          )}
                        >
                          {check.status}
                        </Badge>
                        {check.message && <span className="text-xs text-muted-foreground">{check.message}</span>}
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Go Runtime Info */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Runtime Information</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground">Go Version</p>
                    <p className="text-sm font-medium">{serverStatus.go.version}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">CPU Cores</p>
                    <p className="text-sm font-medium">{serverStatus.go.num_cpu}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Goroutines</p>
                    <p className="text-sm font-medium">{serverStatus.go.goroutines}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Memory Usage</p>
                    <p className="text-sm font-medium">{serverStatus.go.memory_mb} MB</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Server Uptime</p>
                    <p className="text-sm font-medium">{formatUptime(serverStatus.uptime)}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Server Version</p>
                    <p className="text-sm font-medium">{serverStatus.version}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  )
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h ${minutes}m`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}
