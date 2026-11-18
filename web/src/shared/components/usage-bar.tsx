import { cn } from '@/shared/lib/utils'
import { ProgressBar } from './progress-bar'

interface UsageBarProps {
  label: string
  value: number
  className?: string
}

/** Compact labeled usage bar used by device lists and cards. */
export function UsageBar({ label, value, className }: UsageBarProps) {
  return (
    <div className={cn('flex min-w-24 items-center gap-2', className)}>
      <span className="w-8 text-[10px] uppercase tracking-wide text-muted-foreground">{label}</span>
      <ProgressBar value={value} size="sm" variant="auto" className="flex-1" />
      <span className="w-9 text-right font-mono text-xs tabular-nums text-foreground">{Math.round(value)}%</span>
    </div>
  )
}
