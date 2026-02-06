import { useMemo } from 'react'
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { Card } from '@/ui/card'
import { cn } from '@/shared/lib/utils'

interface DataPoint {
  timestamp: string
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
  icon?: React.ReactNode
  loading?: boolean
  className?: string
}

/**
 * Reusable time-series chart component for system metrics
 * Supports dark/light themes with smooth animations
 */
export function MetricsChart({
  title,
  data,
  dataKey,
  color,
  unit = '%',
  formatValue,
  max = 100,
  icon,
  loading,
  className,
}: MetricsChartProps) {
  // Get current value and stats
  const stats = useMemo(() => {
    if (data.length === 0)
      return {
        current: null,
        min: null,
        max: null,
        avg: null,
      }

    const values = data.map((d) => d[dataKey] as number)
    const current = values[values.length - 1]
    const min = Math.min(...values)
    const max = Math.max(...values)
    const avg = values.reduce((a, b) => a + b, 0) / values.length

    return { current, min, max, avg }
  }, [data, dataKey])

  // Format value for display
  const displayValue = useMemo(() => {
    if (stats.current === null) return '—'
    if (formatValue) return formatValue(stats.current)
    return `${stats.current.toFixed(1)}${unit}`
  }, [stats.current, formatValue, unit])

  // Determine color based on value percentage
  const getColorClass = (value: number | null) => {
    if (value === null) return 'text-muted-foreground'
    const percent = (value / max) * 100
    if (percent >= 90) return 'text-red-500 dark:text-red-400'
    if (percent >= 70) return 'text-yellow-500 dark:text-yellow-400'
    return 'text-emerald-600 dark:text-emerald-400'
  }

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
          <div className={cn('text-3xl font-bold tabular-nums tracking-tight', getColorClass(stats.current))}>
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
      <div className="h-48 -mx-2">
        {loading ? (
          <div className="w-full h-full animate-pulse bg-muted rounded-lg" />
        ) : data.length === 0 ? (
          <div className="w-full h-full flex flex-col items-center justify-center text-muted-foreground">
            <div className="text-4xl mb-2">📊</div>
            <p className="text-sm">No data available</p>
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id={`gradient-${dataKey}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color} stopOpacity={0.4} />
                  <stop offset="50%" stopColor={color} stopOpacity={0.2} />
                  <stop offset="100%" stopColor={color} stopOpacity={0} />
                </linearGradient>
                <filter id={`glow-${dataKey}`}>
                  <feGaussianBlur stdDeviation="2" result="coloredBlur" />
                  <feMerge>
                    <feMergeNode in="coloredBlur" />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.2} vertical={false} />
              <XAxis
                dataKey="timestamp"
                tickFormatter={formatTime}
                stroke="var(--muted-foreground)"
                fontSize={11}
                tickLine={false}
                axisLine={false}
                dy={10}
                interval="preserveStartEnd"
                tickCount={6}
                minTickGap={50}
              />
              <YAxis
                domain={[0, max]}
                stroke="var(--muted-foreground)"
                fontSize={11}
                tickLine={false}
                axisLine={false}
                dx={-10}
                width={40}
              />
              <Tooltip
                content={<CustomTooltip formatValue={formatValue} unit={unit} />}
                cursor={{ stroke: color, strokeWidth: 1, strokeDasharray: '5 5' }}
              />
              <Area
                type="monotone"
                dataKey={dataKey}
                stroke={color}
                strokeWidth={3}
                fill={`url(#gradient-${dataKey})`}
                animationDuration={800}
                animationEasing="ease-in-out"
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>
    </Card>
  )
}

interface CustomTooltipProps {
  active?: boolean
  payload?: Array<{ value: number }>
  label?: string
  formatValue?: (value: number) => string
  unit?: string
}

// Custom tooltip component with theme support
const CustomTooltip = ({ active, payload, label, formatValue, unit }: CustomTooltipProps) => {
  if (!active || !payload || payload.length === 0) return null

  const value = payload[0].value
  const displayVal = formatValue ? formatValue(value) : `${value.toFixed(1)}${unit}`

  return (
    <div className="rounded-lg border bg-popover px-3 py-2 text-sm shadow-md">
      <p className="font-medium text-popover-foreground">{displayVal}</p>
      <p className="text-xs text-muted-foreground">{formatTime(label || '')}</p>
    </div>
  )
}

// Format timestamp for X-axis
const formatTime = (timestamp: string) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}
