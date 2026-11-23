import { getDeviceHistory } from '@/shared/tower'
import type { MetricSample } from '@/shared/tower'
import { useEffect, useState } from 'react'

const refreshEvery = 60_000

/** Master-recorded metric history, refreshed slowly; live stats use the WS. */
export function useDeviceHistory(deviceId: string | undefined, enabled = true) {
  const [samples, setSamples] = useState<MetricSample[]>([])

  useEffect(() => {
    if (!deviceId || !enabled) return
    let stopped = false

    const load = async () => {
      try {
        // One batch per five minutes across roughly eight series is about
        // 2300 rows a day, so 5000 keeps the full 24-hour window even after
        // the master switches to returning the newest rows.
        const data = await getDeviceHistory(deviceId, { range: '24h', limit: 5000 })
        if (!stopped) setSamples(data)
      } catch {
        // History is a chart; the live WebSocket already shows connectivity.
      }
    }

    void load()
    const timer = setInterval(() => void load(), refreshEvery)
    return () => {
      stopped = true
      clearInterval(timer)
    }
  }, [deviceId, enabled])

  return samples
}
