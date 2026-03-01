import type { Server } from '@/shared/store/servers'
import type { ServerHealthReport } from '@gloski/sdk'
import { useCallback, useRef, useState } from 'react'

interface UseServerStatusOptions {
  onSuccess?: (status: ServerHealthReport) => void
  onError?: (error: string) => void
}

interface UseServerStatusReturn {
  serverStatus: ServerHealthReport | null
  loading: boolean
  error: string | null
  fetchStatus: () => Promise<void>
}

/**
 * Hook for fetching server status
 * Can be used independently or alongside the server context
 */
export function useServerStatus(server: Server, options: UseServerStatusOptions = {}): UseServerStatusReturn {
  const [serverStatus, setServerStatus] = useState<ServerHealthReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Use ref for callbacks to avoid re-creating fetchStatus on every render
  const optionsRef = useRef(options)
  optionsRef.current = options

  const fetchStatus = useCallback(async () => {
    if (!server.isAuthenticated()) {
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
