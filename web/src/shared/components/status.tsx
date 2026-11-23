import { cn } from '@/shared/lib/utils'
import { statusTone } from '@/shared/lib/status'
import { Badge } from '@/ui/badge'

const dotColors = {
  success: 'bg-success',
  warning: 'bg-warning',
  info: 'bg-info',
  destructive: 'bg-destructive',
  muted: 'bg-muted-foreground/40',
} as const

const badgeVariants = {
  success: 'success',
  warning: 'warning',
  info: 'info',
  destructive: 'destructive',
  muted: 'secondary',
} as const

interface StatusDotProps {
  status: string
  className?: string
  /** Override the default pulse for transient states such as "connecting". */
  pulse?: boolean
}

export function StatusDot({ status, className, pulse }: StatusDotProps) {
  const animated = pulse ?? (status === 'connecting' || status === 'running')
  return (
    <span
      aria-hidden="true"
      className={cn('size-2 shrink-0 rounded-full', dotColors[statusTone(status)], animated && 'animate-pulse', className)}
    />
  )
}

interface StatusBadgeProps {
  status: string
  label?: string
  className?: string
}

export function StatusBadge({ status, label, className }: StatusBadgeProps) {
  const text = label ?? status.charAt(0).toUpperCase() + status.slice(1)
  return (
    <Badge variant={badgeVariants[statusTone(status)]} className={cn('gap-1.5', className)}>
      <StatusDot status={status} className="size-1.5" />
      {text}
    </Badge>
  )
}
