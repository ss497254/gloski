import type { ReactNode } from 'react'
import { cn } from '@/shared/lib/utils'

interface PageLayoutProps {
  title: string
  description?: ReactNode
  actions?: ReactNode
  children: ReactNode
  className?: string
  fullHeight?: boolean
  noPadding?: boolean
}

export function PageLayout({
  title,
  description,
  actions,
  children,
  className,
  fullHeight = true,
  noPadding = false,
}: PageLayoutProps) {
  return (
    <div className={cn('flex flex-col animate-in fade-in duration-200', fullHeight && 'h-full')}>
      {/* Header */}
      <header className="border-b px-4 h-18 md:px-6 shrink-0 flex items-start md:items-center justify-between gap-3 flex-col sm:flex-row">
        <div className="min-w-0">
          <h1 className="text-lg md:text-xl font-semibold truncate">{title}</h1>
          {description && (
            <p className="text-xs md:text-sm text-muted-foreground mt-0.5 md:mt-1 line-clamp-2">{description}</p>
          )}
        </div>
        {actions && <div className="flex items-center gap-2 flex-wrap shrink-0">{actions}</div>}
      </header>

      {/* Content */}
      <div className={cn('flex-1 overflow-auto', !noPadding && 'p-4 md:p-6', className)}>{children}</div>
    </div>
  )
}
