import { useParams, useNavigate } from 'react-router-dom'
import { useServersStore } from '../stores/servers'
import type { GloskiClient } from '@gloski/sdk'

/**
 * Hook to get the current server from route params
 * Use server.getClient() to get the API client instance
 * Redirects to dashboard if server not found
 */
export function useServer() {
  const { serverId } = useParams<{ serverId: string }>()
  const navigate = useNavigate()

  // Use selector for minimal re-renders
  const server = useServersStore((state) => (serverId ? state.servers.find((s) => s.id === serverId) : undefined))

  // Redirect if server not found
  if (serverId && !server) {
    navigate('/', { replace: true })
  }

  return {
    server,
    serverId,
    isAuthenticated: !!(server?.apiKey || server?.token),
  }
}

// Export GloskiClient as ServerApi for backwards compatibility
export type ServerApi = GloskiClient
