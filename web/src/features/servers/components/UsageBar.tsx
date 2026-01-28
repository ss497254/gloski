import { cn } from '@/shared/lib/utils'

function getUsageColor(percent: number): string {
  if (percent >= 90) return 'bg-red-500'
  if (percent >= 75) return 'bg-orange-500'
  if (percent >= 50) return 'bg-yellow-500'
  return 'bg-emerald-500'
}

interface UsageBarProps {
  value: number
  color?: string
}

export function UsageBar({ value, color }: UsageBarProps) {
  return (
    <div className="h-1.5 bg-muted rounded-full overflow-hidden">
      <div
        className={cn('h-full rounded-full transition-all duration-500', color || getUsageColor(value))}
        style={{ width: `${Math.min(value, 100)}%` }}
      />
    </div>
  )
}
