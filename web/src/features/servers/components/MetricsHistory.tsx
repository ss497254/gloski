import { useEffect, useState } from 'react'
import { Cpu, HardDrive, Network } from 'lucide-react'
import { useServer } from '../context'
import { MetricsChart } from '@/shared/components/MetricsChart'
import type { StatsSample } from '@/shared/lib/types'

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

/**
 * Displays time-series charts for CPU, Memory, and Network metrics
 * Fetches last 10 minutes of historical data
 */
export function MetricsHistory({ enabled = true }: MetricsHistoryProps) {
  const { server } = useServer()
  const [data, setData] = useState<ChartData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch historical data
  useEffect(() => {
    if (!enabled) return

    let mounted = true

    const fetchHistory = async () => {
      try {
        setLoading(true)
        setError(null)

        const client = server.getClient()
        const history = await client.system.getHistory('10m')

        if (!mounted) return

        // Transform data for charts
        const chartData: ChartData[] = history.samples.map((sample: StatsSample) => ({
          timestamp: sample.timestamp,
          cpu: sample.stats.cpu.usage_percent,
          memory: sample.stats.memory.used_percent,
          networkRx: sample.stats.network.total_rx,
          networkTx: sample.stats.network.total_tx,
        }))

        setData(chartData)
      } catch (err) {
        if (!mounted) return
        setError(err instanceof Error ? err.message : 'Failed to fetch metrics history')
      } finally {
        if (mounted) setLoading(false)
      }
    }

    fetchHistory()

    // Refresh every 30 seconds
    const interval = setInterval(fetchHistory, 30000)

    return () => {
      mounted = false
      clearInterval(interval)
    }
  }, [server, enabled])

  // Format bytes to human-readable
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
  }

  // Calculate max network value for chart scaling
  const maxNetwork = Math.max(
    ...data.map(d => Math.max(d.networkRx, d.networkTx)),
    1024 * 1024 * 100 // At least 100MB scale
  )

  if (error) {
    return (
      <div className="col-span-full p-6 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900 rounded-lg">
        <p className="text-sm text-red-600 dark:text-red-400">
          Failed to load metrics history: {error}
        </p>
      </div>
    )
  }

  return (
    <>
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
