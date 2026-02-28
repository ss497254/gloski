/* eslint-disable react-refresh/only-export-components */
import type { ServerStats } from '@/shared/lib/types'
import { useServersStore, type Server } from '@/shared/store/servers'
import { createContext, useContext, useEffect, useMemo, type ReactNode } from 'react'

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

const POLLING_INTERVAL = 10000

interface DashboardContextValue {
  // Data
  servers: Server[]
  serversWithStats: ServerStats[]
  displayServers: ServerStats[]

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

  // Track server IDs to detect changes (not the full server objects)
  const serverIds = servers.map((s) => s.id).join(',')

  // Fetch stats for all servers
  useEffect(() => {
    let currentAbort: AbortController | null = null

    const fetchAllStats = async () => {
      // Get fresh server list from store to avoid stale closures
      const { servers: currentServers } = useServersStore.getState()
      // Create a new AbortController for each fetch cycle to avoid race conditions
      const abortController = new AbortController()
      currentAbort = abortController

      await Promise.all(currentServers.map(async (server) => server.getStats({ signal: abortController.signal })))
    }

    // Initial fetch
    fetchAllStats()

    // Poll for stats updates
    const interval = setInterval(fetchAllStats, POLLING_INTERVAL)
    return () => {
      clearInterval(interval)
      currentAbort?.abort()
    }
  }, [serverIds])

  const value: DashboardContextValue = useMemo(
    () => ({
      servers,
      serversWithStats: [],
      displayServers: [],
      onlineCount: 0,
      offlineCount: 0,
      errorCount: 0,
      isLoading: false,
    }),
    [servers]
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
