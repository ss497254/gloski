import { cn } from '@/shared/lib/utils'
import { statusTone, type StatusTone } from '@/shared/lib/status'
import { Badge } from '@/ui/badge'
import { AlertCircle, AlertTriangle, CheckCircle2, CircleDashed, XCircle } from 'lucide-react'

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

/** A circled status mark for lists and cards: tick, warning, cross or dash. */
export function StatusMark({ tone, className }: { tone: StatusTone; className?: string }) {
  const classes = cn('size-4 shrink-0', className)
  switch (tone) {
    case 'success':
      return <CheckCircle2 className={cn(classes, 'text-success')} />
    case 'warning':
      return <AlertTriangle className={cn(classes, 'text-warning')} />
    case 'destructive':
      return <XCircle className={cn(classes, 'text-destructive')} />
    case 'info':
      return <AlertCircle className={cn(classes, 'text-info')} />
    default:
      return <CircleDashed className={cn(classes, 'text-muted-foreground/60')} />
  }
}
