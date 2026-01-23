import { cn } from '@/lib/utils'

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
      if (percentage >= 90) return 'bg-red-500'
      if (percentage >= 70) return 'bg-yellow-500'
      return 'bg-green-500'
    }

    const variants = {
      default: 'bg-primary',
      success: 'bg-green-500',
      warning: 'bg-yellow-500',
      danger: 'bg-red-500',
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
      <div className={cn('w-full rounded-full bg-muted overflow-hidden', sizes[size])}>
        <div
          className={cn(
            'h-full rounded-full transition-all duration-500',
            getVariantClass(),
            barClassName
          )}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  )
}
