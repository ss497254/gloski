/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useMemo, useRef, type ReactNode } from 'react'
import { Navigate, useParams } from 'react-router-dom'
import { createStatsStore, type StatsStoreInstance } from '../../features/servers/stores/statsStore'
import { useServersStore, type Server } from '../store/servers'

// Context for server data and stats store instance
interface ServerContextValue {
  server: Server
  serverId: string
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

  // Create Zustand store instance once (stable reference)
  const statsStore = useMemo<StatsStoreInstance>(() => createStatsStore(), [])

  // Connect/disconnect WebSocket based on server ID (not the full object to avoid reconnect storms)
  const serverRef = useRef(server)
  useEffect(() => {
    serverRef.current = server
  }, [server]) // Update ref when server changes, but use the ref for connection logic to avoid unnecessary reconnects

  // Redirect if no serverId or server not found
  // Memoize context value to prevent cascading re-renders

  const value = useMemo<ServerContextValue>(
    () => ({
      server: server!,
      serverId: serverId!,
      statsStore,
    }),
    [server, serverId, statsStore]
  )
  if (!serverId || !server) {
    return <Navigate to="/" replace />
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
