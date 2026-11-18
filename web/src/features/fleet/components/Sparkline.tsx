import { cn } from '@/shared/lib/utils'

interface SparklineProps {
  values: number[]
  max?: number
  className?: string
  height?: number
}

/** A compact sparkline that inherits its color from the current text color. */
export function Sparkline({ values, max, className, height = 34 }: SparklineProps) {
  const width = 100
  if (values.length < 2) {
    return <div className={className} style={{ height }} />
  }

  const top = Math.max(max ?? 0, ...values, 1)
  const step = width / (values.length - 1)
  const points = values
    .map((value, index) => {
      const x = index * step
      const y = height - (Math.min(value, top) / top) * (height - 2) - 1
      return `${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(' ')

  const area = `0,${height} ${points} ${width},${height}`

  return (
    <svg
      className={cn('block w-full', className)}
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      style={{ height }}
      aria-hidden
    >
      <polygon points={area} fill="currentColor" opacity={0.12} />
      <polyline points={points} fill="none" stroke="currentColor" strokeWidth={1.5} vectorEffect="non-scaling-stroke" />
    </svg>
  )
}
