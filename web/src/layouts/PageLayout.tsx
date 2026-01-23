import type { ReactNode } from 'react'
import { cn } from '@/shared/lib/utils'

interface PageLayoutProps {
  title: string
  description?: string
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
    <div className={cn('flex flex-col', fullHeight && 'h-full')}>
      {/* Header */}
      <header className="border-b px-6 py-4 shrink-0">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold">{title}</h1>
            {description && (
              <p className="text-sm text-muted-foreground mt-1">{description}</p>
            )}
          </div>
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
      </header>

      {/* Content */}
      <div
        className={cn(
          'flex-1 overflow-auto',
          !noPadding && 'p-6',
          className
        )}
      >
        {children}
      </div>
    </div>
  )
}
