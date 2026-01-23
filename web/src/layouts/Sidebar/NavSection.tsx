import type { ReactNode } from 'react'
import { cn } from '@/shared/lib/utils'

interface NavSectionProps {
  title?: string
  collapsed?: boolean
  children: ReactNode
  className?: string
}

export function NavSection({
  title,
  collapsed,
  children,
  className,
}: NavSectionProps) {
  return (
    <div className={cn('space-y-1', className)}>
      {title && !collapsed && (
        <div className="px-3 py-2">
          <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
            {title}
          </span>
        </div>
      )}
      {children}
    </div>
  )
}
