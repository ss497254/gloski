import { cn } from '@/shared/lib/utils'

interface MeterProps {
  value: number
  label?: string
  color?: string
  className?: string
  segments?: number
}

/** A compact block meter, in the spirit of btop's segmented bars. */
export function Meter({ value, label, color = 'bg-emerald-400', className, segments = 24 }: MeterProps) {
  const pct = Math.max(0, Math.min(100, value))
  const filled = Math.round((pct / 100) * segments)

  return (
    <div className={cn('space-y-1', className)}>
      {label !== undefined && (
        <div className="flex items-center justify-between font-mono text-[10px] uppercase tracking-wider text-white/45">
          <span>{label}</span>
          <span className={pct > 90 ? 'text-red-400' : 'text-white/70'}>{pct.toFixed(0)}%</span>
        </div>
      )}
      <div className="flex gap-[2px]" aria-hidden>
        {Array.from({ length: segments }).map((_, index) => (
          <span
            key={index}
            className={cn(
              'h-2 flex-1 rounded-[1px]',
              index < filled ? (pct > 90 ? 'bg-red-400' : pct > 70 ? 'bg-amber-400' : color) : 'bg-white/10'
            )}
          />
        ))}
      </div>
    </div>
  )
}
