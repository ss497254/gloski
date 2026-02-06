/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useMemo, type ReactNode } from 'react'
import { Navigate, useParams } from 'react-router-dom'
import { useServersStore, type Server } from '../stores/servers'
import { createStatsStore, type StatsStoreInstance } from '../stores/statsStore'

// Context for server data and stats store instance
interface ServerContextValue {
  server: Server
  serverId: string
  isAuthenticated: boolean
  // Zustand store instance for stats (pass the store, not the data)
  statsStore: StatsStoreInstance
}

const ServerContext = createContext<ServerContextValue | null>(null)

interface ServerProviderProps {
  children: ReactNode
}

/**
 * Provider component that wraps server-scoped routes.
 * Automatically gets serverId from URL params and provides the server to children.
 * Creates a Zustand stats store and manages WebSocket connection.
 * Redirects to home if server is not found.
 *
 * Uses Zustand store pattern to prevent unnecessary re-renders:
 * - Context provides the store instance (not the data)
 * - Components subscribe to specific slices using selectors
 */
export function ServerProvider({ children }: ServerProviderProps) {
  const { serverId } = useParams<{ serverId: string }>()
  const server = useServersStore((state) => (serverId ? state.servers.find((s) => s.id === serverId) : undefined))
  const updateServer = useServersStore((state) => state.updateServer)

  // Create Zustand store instance once (stable reference)
  const statsStore = useMemo<StatsStoreInstance>(() => createStatsStore(), [])

  // Connect/disconnect WebSocket based on server
  useEffect(() => {
    if (!server) return
    // Connect when server is available
    statsStore.getState().connect(server, () => {
      // Update server status to online when receiving stats
      if (server.status !== 'online') {
        updateServer(server.id, { status: 'online' })
      }
    })

    return () => {
      // Disconnect when unmounting or server changes
      statsStore.getState().disconnect()
    }
  }, [server, updateServer, statsStore])

  // Redirect if no serverId or server not found
  if (!serverId || !server) {
    return <Navigate to="/" replace />
  }

  // Memoize context value (store instance is stable, so this rarely changes)
  const value: ServerContextValue = {
    server,
    serverId,
    isAuthenticated: !!(server.apiKey || server.token),
    statsStore: statsStore!,
  }

  return <ServerContext.Provider value={value}>{children}</ServerContext.Provider>
}

/**
 * Hook to access the current server info and stats store.
 * Must be used within a ServerProvider.
 *
 * @example
 * function ServerDetailPage() {
 *   const { server, statsStore } = useServer()
 *   // Subscribe to specific stats with selector
 *   const stats = useStore(statsStore, (s) => s.stats)
 *   const isConnected = useStore(statsStore, (s) => s.isConnected)
 * }
 *
 * @example
 * function MetricsHistory() {
 *   const { statsStore } = useServer()
 *   useEffect(() => {
 *     // Direct subscription for manual control
 *     const unsubscribe = statsStore.subscribe((state) => {
 *       console.log('New stats:', state.stats)
 *     })
 *     return unsubscribe
 *   }, [statsStore])
 * }
 */
export function useServer(): ServerContextValue {
  const context = useContext(ServerContext)

  if (!context) {
    throw new Error('useServer must be used within a ServerProvider. Wrap your server routes with <ServerProvider>.')
  }

  return context
}

/**
 * Hook to optionally access the current server from context.
 * Returns null if not within a ServerProvider.
 * Useful for components that may or may not be within server context.
 */
export function useServerOptional(): ServerContextValue | null {
  return useContext(ServerContext)
}
