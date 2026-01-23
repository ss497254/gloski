import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  RefreshCw,
  FolderOpen,
  Terminal,
  ListTodo,
  Search,
  WifiOff,
  ShieldAlert,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { EmptyState } from '@/components/common'
import {
  SystemOverview,
  QuickStats,
  DiskUsage,
  NetworkStatsWidget,
  MemoryWidget,
} from '@/components/dashboard'
import { useServersStore } from '@/stores/servers'
import { createServerApi } from '@/services/api'
import type { SystemStats } from '@/lib/types'
import { cn } from '@/lib/utils'

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
  const { getServer, updateServer } = useServersStore()

  const server = serverId ? getServer(serverId) : undefined

  const [stats, setStats] = useState<SystemStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchStats = useCallback(async () => {
    if (!server) return

    if (!server.apiKey && !server.token) {
      setError('No authentication configured. Add an API key to connect.')
      setLoading(false)
      return
    }

    try {
      const api = createServerApi(server)
      const data = await api.stats()
      setStats(data)
      setError(null)
      updateServer(server.id, { status: 'online' })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      setError(message)
      const isAuthError = message.includes('401') || message.includes('Unauthorized')
      updateServer(server.id, { status: isAuthError ? 'unauthorized' : 'offline' })
    } finally {
      setLoading(false)
    }
  }, [server, updateServer])

  useEffect(() => {
    if (!server) {
      navigate('/')
      return
    }

    fetchStats()

    // Poll every 3 seconds
    const interval = setInterval(fetchStats, 3000)
    return () => clearInterval(interval)
  }, [server, fetchStats, navigate])

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

  const quickActions = [
    { to: `/servers/${serverId}/files`, icon: FolderOpen, label: 'Files' },
    { to: `/servers/${serverId}/terminal`, icon: Terminal, label: 'Terminal' },
    { to: `/servers/${serverId}/tasks`, icon: ListTodo, label: 'Tasks' },
    { to: `/servers/${serverId}/search`, icon: Search, label: 'Search' },
  ]

  return (
    <div className="h-full overflow-auto">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b">
        <div className="flex items-center justify-between px-6 py-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight">{server.name}</h1>
              <Badge variant="outline" className="bg-emerald-500/10 text-emerald-600 border-emerald-500/20">
                Online
              </Badge>
            </div>
            <p className="text-sm text-muted-foreground">{stats.hostname} &middot; {server.url}</p>
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
            <Button
              variant="outline"
              size="sm"
              onClick={fetchStats}
              disabled={loading}
            >
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
      </div>
    </div>
  )
}
