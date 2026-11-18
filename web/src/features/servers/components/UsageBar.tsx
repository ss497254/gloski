import { cn } from '@/shared/lib/utils'

function getUsageColor(percent: number): string {
  if (percent >= 90) return 'bg-destructive'
  if (percent >= 75) return 'bg-warning'
  if (percent >= 50) return 'bg-warning'
  return 'bg-success'
}

interface UsageBarProps {
  value: number
  color?: string
}

export function UsageBar({ value, color }: UsageBarProps) {
  return (
    <div
      role="progressbar"
      aria-valuenow={Math.round(Math.min(value, 100))}
      aria-valuemin={0}
      aria-valuemax={100}
      className="h-1.5 bg-muted rounded-full overflow-hidden"
    >
      <div
        className={cn('h-full rounded-full transition-all duration-500', color || getUsageColor(value))}
        style={{ width: `${Math.min(value, 100)}%` }}
      />
    </div>
  )
}
