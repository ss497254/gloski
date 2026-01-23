import { useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useServersStore } from '../stores/servers'
import { createServerApi, type ServerApi } from '@/shared/services/api'

/**
 * Hook to get the current server from route params and create its API instance
 * Redirects to dashboard if server not found
 */
export function useServer() {
  const { serverId } = useParams<{ serverId: string }>()
  const navigate = useNavigate()

  // Use selector to only re-render when the specific server changes
  const server = useServersStore((state) =>
    serverId ? state.servers.find((s) => s.id === serverId) : undefined
  )

  // Extract the properties that the API actually needs for stability
  // This prevents recreating the API when unrelated server properties change (like status)
  const serverUrl = server?.url
  const serverApiKey = server?.apiKey
  const serverToken = server?.token

  const api = useMemo(() => {
    if (!server) return null
    return createServerApi(server)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only recreate when auth-related props change
  }, [serverUrl, serverApiKey, serverToken])

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
