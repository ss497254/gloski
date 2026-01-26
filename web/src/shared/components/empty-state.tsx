import { cn } from '@/shared/lib/utils'
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
      container: 'bg-yellow-500/10',
      icon: 'text-yellow-600',
    },
  }

  const styles = variantStyles[variant]

  return (
    <div className={cn('flex flex-col items-center justify-center py-12 px-4 text-center', className)}>
      {Icon && (
        <div className={cn('mb-4 rounded-2xl p-4', styles.container)}>
          <Icon className={cn('h-8 w-8', styles.icon)} />
        </div>
      )}
      <h3 className="text-lg font-semibold">{title}</h3>
      {description && <p className="mt-2 text-sm text-muted-foreground max-w-md leading-relaxed">{description}</p>}
      {action && <div className="mt-6">{action}</div>}
    </div>
  )
}
