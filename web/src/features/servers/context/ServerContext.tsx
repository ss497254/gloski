/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, type ReactNode } from 'react'
import { useParams, Navigate } from 'react-router-dom'
import { useServersStore, type Server } from '../stores/servers'

interface ServerContextValue {
  server: Server
  serverId: string
  isAuthenticated: boolean
}

const ServerContext = createContext<ServerContextValue | null>(null)

interface ServerProviderProps {
  children: ReactNode
}

/**
 * Provider component that wraps server-scoped routes.
 * Automatically gets serverId from URL params and provides the server to children.
 * Redirects to home if server is not found.
 */
export function ServerProvider({ children }: ServerProviderProps) {
  const { serverId } = useParams<{ serverId: string }>()
  const server = useServersStore((state) => (serverId ? state.servers.find((s) => s.id === serverId) : undefined))

  // Redirect if no serverId or server not found
  if (!serverId || !server) {
    return <Navigate to="/" replace />
  }

  const value: ServerContextValue = {
    server,
    serverId,
    isAuthenticated: !!(server.apiKey || server.token),
  }

  return <ServerContext.Provider value={value}>{children}</ServerContext.Provider>
}

/**
 * Hook to access the current server from context.
 * Must be used within a ServerProvider.
 *
 * @example
 * function FilesPage() {
 *   const { server, serverId, isAuthenticated } = useServer()
 *   const client = server.getClient()
 *   // ...
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
