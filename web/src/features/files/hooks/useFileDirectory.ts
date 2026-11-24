import type { DeviceClient } from '@/shared/lib/device'
import { unwrap, type ListResponse } from '@/shared/api'
import { useCallback, useEffect, useRef, useState } from 'react'

/** Owns directory request state and ignores out-of-order responses. */
export function useFileDirectory(device: DeviceClient, queryPath: string) {
  const [data, setData] = useState<ListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const requestRef = useRef(0)

  const loadDirectory = useCallback(
    async (path: string) => {
      const request = ++requestRef.current
      setLoading(true)
      setError('')
      try {
        const result = await device.getClient().files.list(path)
        if (request === requestRef.current) setData(unwrap(result))
      } catch (err) {
        if (request === requestRef.current) {
          setError(err instanceof Error ? err.message : 'Failed to load directory')
        }
      } finally {
        if (request === requestRef.current) setLoading(false)
      }
    },
    [device]
  )

  useEffect(() => {
    const requestRefForCleanup = requestRef
    void loadDirectory(queryPath)
    return () => {
      requestRefForCleanup.current++
    }
  }, [loadDirectory, queryPath])

  return { data, loading, error, loadDirectory }
}
