import { useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useServersStore } from '@/stores/servers'
import { createServerApi, type ServerApi } from '@/services/api'

/**
 * Hook to get the current server from route params and create its API instance
 * Redirects to dashboard if server not found
 */
export function useServer() {
  const { serverId } = useParams<{ serverId: string }>()
  const navigate = useNavigate()
  const { getServer } = useServersStore()

  const server = serverId ? getServer(serverId) : undefined

  const api = useMemo(() => {
    if (!server) return null
    return createServerApi(server)
  }, [server])

  // Redirect if server not found
  if (serverId && !server) {
    navigate('/', { replace: true })
  }

  return {
    server,
    serverId,
    api,
    isAuthenticated: !!(server?.apiKey || server?.token),
  }
}

export type { ServerApi }
