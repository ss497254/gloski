interface SparklineProps {
  values: number[]
  max?: number
  color?: string
  className?: string
  height?: number
}

export function Sparkline({ values, max, color = '#7ee787', className, height = 34 }: SparklineProps) {
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
      className={className}
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      style={{ height, width: '100%' }}
    >
      <polygon points={area} fill={color} opacity={0.12} />
      <polyline points={points} fill="none" stroke={color} strokeWidth={1.5} vectorEffect="non-scaling-stroke" />
    </svg>
  )
}
