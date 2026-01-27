import { useCallback, useState } from 'react'

interface UseAsyncState<T> {
  data: T | null
  loading: boolean
  error: string | null
}

interface UseAsyncReturn<T, Args extends unknown[]> extends UseAsyncState<T> {
  execute: (...args: Args) => Promise<T | null>
  reset: () => void
}

/**
 * Hook for handling async operations with loading and error states
 */
export function useAsync<T, Args extends unknown[] = []>(
  asyncFn: (...args: Args) => Promise<T>
): UseAsyncReturn<T, Args> {
  const [state, setState] = useState<UseAsyncState<T>>({
    data: null,
    loading: false,
    error: null,
  })

  const execute = useCallback(
    async (...args: Args): Promise<T | null> => {
      setState((prev) => ({ ...prev, loading: true, error: null }))

      try {
        const data = await asyncFn(...args)
        setState({ data, loading: false, error: null })
        return data
      } catch (err) {
        const error = err instanceof Error ? err.message : 'An error occurred'
        setState((prev) => ({ ...prev, loading: false, error }))
        return null
      }
    },
    [asyncFn]
  )

  const reset = useCallback(() => {
    setState({ data: null, loading: false, error: null })
  }, [])

  return { ...state, execute, reset }
}
