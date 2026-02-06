/* eslint-disable react-refresh/only-export-components */
import { type Server, useServersStore, getSortedServers } from '@/features/servers'
import type { SystemStats } from '@/shared/lib/types'
import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

const POLLING_INTERVAL = 10000

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export interface ServerWithStats extends Server {
  stats?: SystemStats | null
  statsLoading?: boolean
  statsError?: string | null
}

interface DashboardContextValue {
  // Data
  servers: Server[]
  serversWithStats: ServerWithStats[]
  displayServers: ServerWithStats[]

  // Computed
  onlineCount: number
  offlineCount: number

  // Loading state
  isLoading: boolean

  // Error count (servers with errors)
  errorCount: number
}

const DashboardContext = createContext<DashboardContextValue | null>(null)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

interface DashboardProviderProps {
  children: ReactNode
}

export function DashboardProvider({ children }: DashboardProviderProps) {
  const servers = useServersStore((state) => state.servers)
  const updateServer = useServersStore((state) => state.updateServer)
  const [serversWithStats, setServersWithStats] = useState<ServerWithStats[]>([])

  // Memoize sorted servers to prevent unnecessary recalculations
  const sortedServers = useMemo(() => getSortedServers(servers), [servers])

  // Track server IDs to detect changes (not the full server objects)
  const serverIds = servers.map((s) => s.id).join(',')

  // Fetch stats for all servers
  useEffect(() => {
    // AbortController to cancel pending requests on unmount
    const abortController = new AbortController()
    // Get fresh server list from store to avoid stale closures
    const { servers: currentServers } = useServersStore.getState()

    const fetchAllStats = async () => {
      const results = await Promise.all(
        currentServers.map(async (server) => {
          if (!server.apiKey && !server.token) {
            return { ...server, stats: null, statsError: 'No authentication configured' }
          }

          try {
            const stats = await server.getClient().system.getStats({ signal: abortController.signal })
            // Only update if status changed to avoid unnecessary re-renders
            if (server.status !== 'online') {
              updateServer(server.id, { status: 'online' })
            }
            return { ...server, status: 'online' as const, stats, statsError: null }
          } catch (err) {
            // Ignore aborted requests
            if (err instanceof Error && err.name === 'AbortError') {
              return { ...server, stats: null, statsError: null }
            }
            const isAuthError = err instanceof Error && err.message.includes('401')
            const newStatus = isAuthError ? 'unauthorized' : 'offline'
            // Only update if status changed
            if (server.status !== newStatus) {
              updateServer(server.id, { status: newStatus })
            }
            return {
              ...server,
              status: newStatus as 'unauthorized' | 'offline',
              stats: null,
              statsError: err instanceof Error ? err.message : 'Unknown error',
            }
          }
        })
      )
      setServersWithStats(results)
    }

    if (currentServers.length === 0) {
      setServersWithStats([])
      return
    }

    // Initial fetch
    setServersWithStats(currentServers.map((s) => ({ ...s, statsLoading: true })))
    fetchAllStats()

    // Poll for stats updates
    const interval = setInterval(fetchAllStats, POLLING_INTERVAL)
    return () => {
      clearInterval(interval)
      abortController.abort() // Cancel pending requests on cleanup
    }
  }, [serverIds, updateServer])

  // Merge sorted order with stats - memoized to avoid unnecessary recalculations
  const displayServers = useMemo(
    () =>
      sortedServers.map((server) => {
        const withStats = serversWithStats.find((s) => s.id === server.id)
        return withStats || { ...server, statsLoading: true }
      }),
    [sortedServers, serversWithStats]
  )

  const onlineCount = useMemo(
    () => serversWithStats.filter((s) => s.status === 'online').length,
    [serversWithStats]
  )

  const offlineCount = useMemo(
    () => serversWithStats.filter((s) => s.status !== 'online' && s.status !== 'connecting').length,
    [serversWithStats]
  )

  const isLoading = useMemo(
    () => serversWithStats.length === 0 && servers.length > 0,
    [serversWithStats.length, servers.length]
  )

  const errorCount = useMemo(
    () => serversWithStats.filter((s) => s.statsError != null).length,
    [serversWithStats]
  )

  const value: DashboardContextValue = useMemo(
    () => ({
      servers,
      serversWithStats,
      displayServers,
      onlineCount,
      offlineCount,
      isLoading,
      errorCount,
    }),
    [servers, serversWithStats, displayServers, onlineCount, offlineCount, isLoading, errorCount]
  )

  return <DashboardContext.Provider value={value}>{children}</DashboardContext.Provider>
}

// ─────────────────────────────────────────────────────────────────────────────
// Hook
// ─────────────────────────────────────────────────────────────────────────────

export function useDashboard(): DashboardContextValue {
  const context = useContext(DashboardContext)

  if (!context) {
    throw new Error('useDashboard must be used within a DashboardProvider.')
  }

  return context
}
