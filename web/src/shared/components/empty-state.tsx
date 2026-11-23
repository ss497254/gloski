import { cn } from '@/shared/lib/utils'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/ui/empty'
import type { LucideIcon } from 'lucide-react'

interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
  variant?: 'default' | 'error' | 'warning'
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
  variant = 'default',
}: EmptyStateProps) {
  const variantStyles = {
    default: {
      container: 'bg-muted',
      icon: 'text-muted-foreground',
    },
    error: {
      container: 'bg-destructive/10',
      icon: 'text-destructive',
    },
    warning: {
      container: 'bg-warning/15',
      icon: 'text-warning',
    },
  }

  const styles = variantStyles[variant]

  return (
    <Empty className={cn('py-12', className)}>
      <EmptyHeader>
        {Icon && (
          <EmptyMedia variant="icon" className={cn('size-14 rounded-2xl', styles.container)}>
            <Icon className={cn('size-8', styles.icon)} />
          </EmptyMedia>
        )}
        <EmptyTitle>{title}</EmptyTitle>
        {description && <EmptyDescription>{description}</EmptyDescription>}
      </EmptyHeader>
      {action && <EmptyContent>{action}</EmptyContent>}
    </Empty>
  )
}
