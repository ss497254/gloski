import { cn } from '@/shared/lib/utils'
import { Progress } from '@/ui/progress'

interface ProgressBarProps {
  value: number
  max?: number
  className?: string
  barClassName?: string
  showLabel?: boolean
  label?: string
  size?: 'sm' | 'md' | 'lg'
  variant?: 'default' | 'success' | 'warning' | 'danger' | 'auto'
}

export function ProgressBar({
  value,
  max = 100,
  className,
  barClassName,
  showLabel = false,
  label,
  size = 'md',
  variant = 'auto',
}: ProgressBarProps) {
  const percentage = Math.min(100, Math.max(0, (value / max) * 100))

  const getVariantClass = () => {
    if (variant === 'auto') {
      if (percentage >= 90) return 'bg-destructive'
      if (percentage >= 70) return 'bg-warning'
      return 'bg-success'
    }

    const variants = {
      default: 'bg-primary',
      success: 'bg-success',
      warning: 'bg-warning',
      danger: 'bg-destructive',
    }

    return variants[variant]
  }

  const sizes = {
    sm: 'h-1.5',
    md: 'h-2.5',
    lg: 'h-4',
  }

  return (
    <div className={cn('w-full', className)}>
      {(showLabel || label) && (
        <div className="flex justify-between text-sm mb-1.5">
          {label && <span className="text-muted-foreground">{label}</span>}
          {showLabel && <span className="font-medium">{Math.round(percentage)}%</span>}
        </div>
      )}
      <Progress
        value={percentage}
        aria-label={label || 'Progress'}
        className={cn('w-full bg-muted', sizes[size])}
        indicatorClassName={cn('rounded-full duration-500', getVariantClass(), barClassName)}
      />
    </div>
  )
}
