import { useState, useCallback, useRef } from 'react'
import type { ServerStatus } from '@/shared/lib/types'
import type { Server } from '../stores/servers'

interface UseServerStatusOptions {
  onSuccess?: (status: ServerStatus) => void
  onError?: (error: string) => void
}

interface UseServerStatusReturn {
  serverStatus: ServerStatus | null
  loading: boolean
  error: string | null
  fetchStatus: () => Promise<void>
}

/**
 * Hook for fetching server status
 * Can be used independently or alongside the server context
 */
export function useServerStatus(server: Server, options: UseServerStatusOptions = {}): UseServerStatusReturn {
  const [serverStatus, setServerStatus] = useState<ServerStatus | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Use ref for callbacks to avoid re-creating fetchStatus on every render
  const optionsRef = useRef(options)
  optionsRef.current = options

  const fetchStatus = useCallback(async () => {
    if (!server.apiKey && !server.token) {
      const errorMsg = 'No authentication configured'
      setError(errorMsg)
      setLoading(false)
      optionsRef.current.onError?.(errorMsg)
      return
    }

    try {
      setLoading(true)
      setError(null)

      const client = server.getClient()
      const status = await client.system.getStatus()

      setServerStatus(status)
      setError(null)
      optionsRef.current.onSuccess?.(status)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      setError(message)
      optionsRef.current.onError?.(message)
    } finally {
      setLoading(false)
    }
  }, [server])

  return {
    serverStatus,
    loading,
    error,
    fetchStatus,
  }
}
