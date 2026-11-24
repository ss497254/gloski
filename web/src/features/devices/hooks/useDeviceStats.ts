import { deviceApiPrefix, deviceStatsSocketUrl } from '@/shared/tower'
import type { ProcessInfo, SystemStats } from '@/shared/api'
import { useEffect, useRef, useState } from 'react'

interface DeviceStatsState {
  stats: SystemStats | null
  processes: ProcessInfo[]
  connected: boolean
  error: string | null
}

interface ProcessResponse {
  processes?: ProcessInfo[]
  data?: { processes?: ProcessInfo[] }
}

/** Live stats over the master's WebSocket proxy plus a processes poll. */
export function useDeviceStats(deviceId: string | undefined, enabled = true) {
  const [state, setState] = useState<DeviceStatsState>({
    stats: null,
    processes: [],
    connected: false,
    error: null,
  })
  const reconnectDelay = useRef(1000)

  useEffect(() => {
    if (!deviceId || !enabled) return

    let stopped = false
    let socket: WebSocket | null = null
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null
    let processTimer: ReturnType<typeof setInterval> | null = null

    const connect = () => {
      if (stopped) return
      socket = new WebSocket(deviceStatsSocketUrl(deviceId))

      socket.onopen = () => {
        if (stopped) return
        reconnectDelay.current = 1000
        setState((current) => ({ ...current, connected: true, error: null }))
      }

      socket.onmessage = (event) => {
        if (stopped) return
        try {
          const stats = JSON.parse(event.data) as SystemStats
          setState((current) => ({ ...current, stats, error: null }))
        } catch {
          // Ignore malformed frames.
        }
      }

      socket.onerror = () => {
        if (stopped) return
        setState((current) => ({ ...current, connected: false }))
      }

      socket.onclose = () => {
        if (stopped) return
        setState((current) => ({ ...current, connected: false }))
        reconnectTimer = setTimeout(connect, reconnectDelay.current)
        reconnectDelay.current = Math.min(reconnectDelay.current * 2, 15000)
      }
    }

    const fetchProcesses = async () => {
      try {
        const response = await fetch(`${deviceApiPrefix(deviceId)}/system/processes?limit=20&sort=cpu`, {
          cache: 'no-store',
        })
        if (!response.ok) return
        const payload = (await response.json()) as ProcessResponse
        const list = payload.data?.processes ?? payload.processes
        if (!stopped && list) {
          setState((current) => ({ ...current, processes: list }))
        }
      } catch {
        // The stats WebSocket already surfaces connectivity.
      }
    }

    connect()
    fetchProcesses()
    processTimer = setInterval(fetchProcesses, 2500)

    return () => {
      stopped = true
      if (reconnectTimer) clearTimeout(reconnectTimer)
      if (processTimer) clearInterval(processTimer)
      socket?.close()
    }
  }, [deviceId, enabled])

  return state
}
