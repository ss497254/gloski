import { useMemo, memo, Component, type ReactNode } from 'react'
import { Line } from 'react-chartjs-2'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Filler,
  type ChartOptions,
} from 'chart.js'
import { Card } from '@/ui/card'
import { cn } from '@/shared/lib/utils'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler)

// ============================================================================
// Types
// ============================================================================

interface DataPoint {
  [key: string]: string | number
}

interface MetricsChartProps {
  title: string
  data: DataPoint[]
  dataKey: string
  color: string
  unit?: string
  formatValue?: (value: number) => string
  max?: number
  icon?: ReactNode
  loading?: boolean
  className?: string
}

// ============================================================================
// Utilities & Helpers
// ============================================================================

const COLOR_CACHE = new Map<string, string>()

function hslToRgba(hsl: string, alpha: number): string {
  const cacheKey = `${hsl}-${alpha}`
  if (COLOR_CACHE.has(cacheKey)) return COLOR_CACHE.get(cacheKey)!

  const match = hsl.match(/hsl\((\d+),\s*(\d+)%,\s*(\d+)%\)/)
  if (!match) return `rgba(100, 100, 100, ${alpha})`

  const h = parseInt(match[1])
  const s = parseInt(match[2]) / 100
  const l = parseInt(match[3]) / 100
  const c = (1 - Math.abs(2 * l - 1)) * s
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = l - c / 2

  let r = 0,
    g = 0,
    b = 0
  if (h < 60) {
    r = c
    g = x
    b = 0
  } else if (h < 120) {
    r = x
    g = c
    b = 0
  } else if (h < 180) {
    r = 0
    g = c
    b = x
  } else if (h < 240) {
    r = 0
    g = x
    b = c
  } else if (h < 300) {
    r = x
    g = 0
    b = c
  } else {
    r = c
    g = 0
    b = x
  }

  const result = `rgba(${Math.round((r + m) * 255)}, ${Math.round((g + m) * 255)}, ${Math.round((b + m) * 255)}, ${alpha})`
  COLOR_CACHE.set(cacheKey, result)
  return result
}

function isValidNumber(value: unknown): value is number {
  return typeof value === 'number' && isFinite(value)
}

function getColorClass(value: number | null, max: number): string {
  if (value === null) return 'text-muted-foreground'
  const percent = (value / max) * 100
  if (percent >= 90) return 'text-red-500 dark:text-red-400'
  if (percent >= 70) return 'text-yellow-500 dark:text-yellow-400'
  return 'text-emerald-600 dark:text-emerald-400'
}

function getThemeAwareColor(): string {
  if (typeof document === 'undefined') return 'rgba(128, 128, 128, 0.6)'
  const isDark = document.documentElement.classList.contains('dark')
  return isDark ? 'rgba(255, 255, 255, 0.6)' : 'rgba(0, 0, 0, 0.6)'
}

// ============================================================================
// Error Boundary
// ============================================================================

class ChartErrorBoundary extends Component<{ children: ReactNode }, { hasError: boolean }> {
  state = { hasError: false }

  static getDerivedStateFromError() {
    return { hasError: true }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="h-48 w-full flex flex-col items-center justify-center text-muted-foreground border border-dashed rounded-lg">
          <div className="text-4xl mb-2">⚠️</div>
          <p className="text-sm">Chart failed to render</p>
        </div>
      )
    }
    return this.props.children
  }
}

// ============================================================================
// Main Component
// ============================================================================

export const MetricsChart = memo(function MetricsChart({
  title,
  data,
  dataKey,
  color,
  unit = '%',
  formatValue,
  max = 100,
  icon,
  loading = false,
  className,
}: MetricsChartProps) {
  // Calculate statistics
  const stats = useMemo(() => {
    if (data.length === 0) {
      return { current: null, min: null, max: null, avg: null }
    }

    let min = Infinity,
      max = -Infinity,
      sum = 0,
      current = 0,
      validCount = 0

    for (let i = 0; i < data.length; i++) {
      const value = data[i][dataKey] as number
      if (isValidNumber(value)) {
        if (value < min) min = value
        if (value > max) max = value
        sum += value
        validCount++
        if (i === data.length - 1) current = value
      }
    }

    if (validCount === 0) {
      return { current: null, min: null, max: null, avg: null }
    }

    return {
      current,
      min: min === Infinity ? 0 : min,
      max: max === -Infinity ? 0 : max,
      avg: sum / validCount,
    }
  }, [data, dataKey])

  // Prepare chart data
  const chartData = useMemo(() => {
    const values: number[] = []

    for (let i = 0; i < data.length; i++) {
      const value = data[i][dataKey] as number
      values.push(isValidNumber(value) ? value : 0)
    }

    return {
      labels: Array.from({ length: data.length }, (_, i) => i.toString()),
      datasets: [
        {
          data: values,
          borderColor: color,
          backgroundColor: hslToRgba(color, 0.15),
          borderWidth: 2,
          fill: true,
          tension: 0.4,
          pointRadius: 0,
        },
      ],
    }
  }, [data, dataKey, color])

  // Chart options
  const options = useMemo<ChartOptions<'line'>>(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      plugins: {
        legend: { display: false },
        tooltip: { enabled: false },
      },
      scales: {
        x: { display: false },
        y: {
          display: true,
          min: 0,
          max,
          grid: { display: false },
          border: { display: false },
          ticks: {
            count: 5,
            font: { size: 10 },
            color: () => getThemeAwareColor(),
            callback: (tickValue) => {
              const value = Number(tickValue)
              return formatValue ? formatValue(value) : `${value}${unit}`
            },
          },
        },
      },
    }),
    [max, formatValue, unit]
  )

  // Format display values
  const displayValue =
    stats.current === null ? '—' : formatValue ? formatValue(stats.current) : `${stats.current.toFixed(1)}${unit}`
  const colorClass = getColorClass(stats.current, max)

  return (
    <Card className={cn('p-6 hover:shadow-lg transition-shadow duration-300', className)}>
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-2">
          {icon && <div className="p-2 rounded-lg bg-muted/50 text-muted-foreground">{icon}</div>}
          <div>
            <h3 className="text-sm font-medium text-foreground">{title}</h3>
            {stats.avg !== null && (
              <p className="text-xs text-muted-foreground mt-0.5">
                Avg: {formatValue ? formatValue(stats.avg) : `${stats.avg.toFixed(1)}${unit}`}
              </p>
            )}
          </div>
        </div>

        <div className="text-right">
          <div className={cn('text-3xl font-bold tabular-nums tracking-tight', colorClass)}>
            {loading ? <div className="h-9 w-24 animate-pulse bg-muted rounded" /> : displayValue}
          </div>
          {stats.min !== null && stats.max !== null && !loading && (
            <p className="text-xs text-muted-foreground mt-1">
              {formatValue ? formatValue(stats.min) : `${stats.min.toFixed(1)}${unit}`}
              {' - '}
              {formatValue ? formatValue(stats.max) : `${stats.max.toFixed(1)}${unit}`}
            </p>
          )}
        </div>
      </div>

      {/* Chart */}
      <div className="h-48 w-full" role="img" aria-label={`${title} chart showing ${displayValue}`}>
        {loading ? (
          <div className="w-full h-full animate-pulse bg-muted rounded-lg" />
        ) : data.length === 0 ? (
          <div className="w-full h-full flex flex-col items-center justify-center text-muted-foreground">
            <div className="text-4xl mb-2">📊</div>
            <p className="text-sm">No data available</p>
          </div>
        ) : (
          <ChartErrorBoundary>
            <Line data={chartData} options={options} />
          </ChartErrorBoundary>
        )}
      </div>
    </Card>
  )
})
