/* eslint-disable react-refresh/only-export-components */
import { type Server, useServersStore, getSortedServers } from '@/features/servers'
import type { ServerWithStats } from '@/shared/lib/types'
import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

const POLLING_INTERVAL = 10000

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
    let cancelled = false
    let currentAbort: AbortController | null = null

    const fetchAllStats = async () => {
      // Get fresh server list from store to avoid stale closures
      const { servers: currentServers } = useServersStore.getState()
      // Create a new AbortController for each fetch cycle to avoid race conditions
      const abortController = new AbortController()
      currentAbort = abortController

      const results = await Promise.all(
        currentServers.map(async (server) => {
          if (!server.apiKey && !server.token) {
            return { ...server, stats: null, statsError: 'No authentication configured' }
          }

          // Skip polling offline servers to reduce unnecessary network requests
          if (server.status === 'offline') {
            return { ...server, stats: null, statsError: 'Server offline' }
          }

          try {
            const stats = await server.getClient().system.getStats({ signal: abortController.signal })
            if (server.status !== 'online') {
              updateServer(server.id, { status: 'online' })
            }
            return { ...server, status: 'online' as const, stats, statsError: null }
          } catch (err) {
            if (err instanceof Error && err.name === 'AbortError') {
              return { ...server, stats: null, statsError: null }
            }
            const isAuthError = err instanceof Error && err.message.includes('401')
            const newStatus = isAuthError ? 'unauthorized' : 'offline'
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
      if (!cancelled) {
        setServersWithStats(results)
      }
    }

    const { servers: currentServers } = useServersStore.getState()
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
      cancelled = true
      clearInterval(interval)
      currentAbort?.abort()
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
