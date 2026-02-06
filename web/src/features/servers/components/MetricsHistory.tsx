import { useCallback, useEffect, useMemo, useState } from 'react'
import { Cpu, HardDrive, Network } from 'lucide-react'
import { useServer } from '../context'
import { MetricsChart } from '@/shared/components/MetricsChart'
import { Button } from '@/ui/button'
import type { StatsSample } from '@/shared/lib/types'
import { cn } from '@/shared/lib/utils'

interface ChartData {
  timestamp: string
  cpu: number
  memory: number
  networkRx: number
  networkTx: number
  [key: string]: string | number
}

interface MetricsHistoryProps {
  enabled?: boolean
}

type TimeRange = '2m' | '5m' | '10m'

const TIME_RANGE_CONFIG = {
  '2m': { duration: '2m', samples: 60, label: '2 min' },
  '5m': { duration: '5m', samples: 150, label: '5 min' },
  '10m': { duration: '10m', samples: 300, label: '10 min' },
} as const

/**
 * Displays time-series charts for CPU, Memory, and Network metrics
 * Fetches historical data once on mount, then updates in real-time via WebSocket
 */
export function MetricsHistory({ enabled = true }: MetricsHistoryProps) {
  const { server, statsStore } = useServer()
  const [data, setData] = useState<ChartData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [timeRange, setTimeRange] = useState<TimeRange>('10m')

  // Fetch historical data when time range changes
  useEffect(() => {
    if (!enabled) return

    const abortController = new AbortController()
    let mounted = true

    const fetchHistory = async () => {
      try {
        setLoading(true)
        setError(null)

        const config = TIME_RANGE_CONFIG[timeRange]
        const client = server.getClient()
        const history = await client.system.getHistory(config.duration, { signal: abortController.signal })

        if (!mounted) return

        // Transform data for charts
        const chartData: ChartData[] = history.samples.map((sample: StatsSample) => ({
          timestamp: sample.timestamp,
          cpu: sample.stats.cpu.usage_percent,
          memory: sample.stats.memory.used_percent,
          networkRx: sample.stats.network.total_rx,
          networkTx: sample.stats.network.total_tx,
        }))

        // Pad with empty data if server has less than requested duration
        const expectedSamples = config.samples
        if (chartData.length < expectedSamples) {
          const missingCount = expectedSamples - chartData.length
          const now = Date.now()
          const emptyData: ChartData[] = Array.from({ length: missingCount }, (_, i) => ({
            timestamp: new Date(now - (expectedSamples - i) * 2000).toISOString(),
            cpu: 0,
            memory: 0,
            networkRx: 0,
            networkTx: 0,
          }))
          setData([...emptyData, ...chartData])
        } else {
          setData(chartData)
        }
      } catch (err) {
        if (!mounted) return
        // Ignore aborted requests
        if (err instanceof Error && err.name === 'AbortError') return
        setError(err instanceof Error ? err.message : 'Failed to fetch metrics history')
      } finally {
        if (mounted) setLoading(false)
      }
    }

    fetchHistory()

    return () => {
      mounted = false
      abortController.abort()
    }
  }, [server, enabled, timeRange])

  // Subscribe to stats store and append to history
  useEffect(() => {
    if (!enabled) return

    const unsubscribe = statsStore.subscribe(({ stats }) => {
      if (!stats) return

      const newDataPoint: ChartData = {
        timestamp: new Date().toISOString(),
        cpu: stats.cpu.usage_percent,
        memory: stats.memory.used_percent,
        networkRx: stats.network.total_rx,
        networkTx: stats.network.total_tx,
      }

      setData((prevData) => {
        // Append new data point and keep only the selected time range
        const maxSamples = TIME_RANGE_CONFIG[timeRange].samples
        const updatedData = [...prevData, newDataPoint]
        return updatedData.length > maxSamples ? updatedData.slice(-maxSamples) : updatedData
      })
    })

    return unsubscribe
  }, [statsStore, enabled, timeRange])

  // Format bytes to human-readable (memoized to prevent recreation on every render)
  const formatBytes = useCallback((bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
  }, [])

  // Calculate max network value for chart scaling (memoized to prevent recalculation)
  const maxNetwork = useMemo(
    () =>
      Math.max(
        ...data.map((d) => Math.max(d.networkRx, d.networkTx)),
        1024 * 1024 * 100 // At least 100MB scale
      ),
    [data]
  )

  if (error) {
    return (
      <div className="col-span-full p-6 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900 rounded-lg">
        <p className="text-sm text-red-600 dark:text-red-400">Failed to load metrics history: {error}</p>
      </div>
    )
  }

  return (
    <>
      {/* Time Range Selector */}
      <div className="col-span-full flex items-center justify-between mb-2">
        <h3 className="text-lg font-semibold">Metrics History</h3>
        <div className="flex items-center gap-1 border rounded-lg p-1 bg-muted/50">
          {(Object.keys(TIME_RANGE_CONFIG) as TimeRange[]).map((range) => (
            <Button
              key={range}
              variant="ghost"
              size="sm"
              onClick={() => setTimeRange(range)}
              className={cn(
                'h-8 px-3 text-xs font-medium transition-colors',
                timeRange === range
                  ? 'bg-background shadow-sm text-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {TIME_RANGE_CONFIG[range].label}
            </Button>
          ))}
        </div>
      </div>

      {/* CPU Chart */}
      <MetricsChart
        title="CPU Usage"
        data={data}
        dataKey="cpu"
        color="hsl(142, 71%, 45%)" // emerald-600
        unit="%"
        max={100}
        icon={<Cpu className="h-5 w-5" />}
        loading={loading}
      />

      {/* Memory Chart */}
      <MetricsChart
        title="Memory Usage"
        data={data}
        dataKey="memory"
        color="hsl(217, 91%, 60%)" // blue-500
        unit="%"
        max={100}
        icon={<HardDrive className="h-5 w-5" />}
        loading={loading}
      />

      {/* Network RX Chart */}
      <MetricsChart
        title="Network Received"
        data={data}
        dataKey="networkRx"
        color="hsl(262, 83%, 58%)" // violet-600
        unit=""
        max={maxNetwork}
        formatValue={formatBytes}
        icon={<Network className="h-5 w-5" />}
        loading={loading}
      />

      {/* Network TX Chart */}
      <MetricsChart
        title="Network Sent"
        data={data}
        dataKey="networkTx"
        color="hsl(316, 70%, 50%)" // fuchsia-600
        unit=""
        max={maxNetwork}
        formatValue={formatBytes}
        icon={<Network className="h-5 w-5" />}
        loading={loading}
      />
    </>
  )
}
