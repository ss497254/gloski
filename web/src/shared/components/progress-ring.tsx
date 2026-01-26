import { cn } from '@/shared/lib/utils'

interface ProgressRingProps {
  value: number
  size?: number
  strokeWidth?: number
  className?: string
  trackClassName?: string
  progressClassName?: string
  showValue?: boolean
  valueClassName?: string
  label?: string
  labelClassName?: string
}

export function ProgressRing({
  value,
  size = 120,
  strokeWidth = 10,
  className,
  trackClassName,
  progressClassName,
  showValue = true,
  valueClassName,
  label,
  labelClassName,
}: ProgressRingProps) {
  const normalizedValue = Math.min(100, Math.max(0, value))
  const radius = (size - strokeWidth) / 2
  const circumference = radius * 2 * Math.PI
  const offset = circumference - (normalizedValue / 100) * circumference

  // Color based on value
  const getColor = () => {
    if (normalizedValue >= 90) return 'text-red-500'
    if (normalizedValue >= 70) return 'text-yellow-500'
    return 'text-green-500'
  }

  return (
    <div className={cn('relative inline-flex items-center justify-center', className)}>
      <svg width={size} height={size} className="-rotate-90">
        {/* Track */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          className={cn('stroke-muted', trackClassName)}
        />
        {/* Progress */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          className={cn('stroke-current transition-all duration-500', getColor(), progressClassName)}
        />
      </svg>
      {(showValue || label) && (
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          {showValue && (
            <span className={cn('text-2xl font-bold', valueClassName)}>{Math.round(normalizedValue)}%</span>
          )}
          {label && <span className={cn('text-xs text-muted-foreground', labelClassName)}>{label}</span>}
        </div>
      )}
    </div>
  )
}
