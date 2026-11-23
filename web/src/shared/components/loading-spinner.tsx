import { cn } from '@/shared/lib/utils'
import { Spinner } from '@/ui/spinner'
import { Server } from 'lucide-react'

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export function LoadingSpinner({ size = 'md', className }: LoadingSpinnerProps) {
  const sizes = {
    sm: 'size-4',
    md: 'size-8',
    lg: 'size-12',
  }

  return <Spinner className={cn('text-primary', sizes[size], className)} />
}

interface PageLoaderProps {
  message?: string
  submessage?: string
}

export function PageLoader({ message = 'Loading...', submessage }: PageLoaderProps) {
  return (
    <div className="h-full flex flex-col items-center justify-center gap-4">
      <div className="relative">
        <div className="h-16 w-16 rounded-2xl bg-primary/10 flex items-center justify-center">
          <Server className="h-8 w-8 text-primary" />
        </div>
        <div className="absolute -bottom-1 -right-1 h-5 w-5 rounded-full bg-background flex items-center justify-center">
          <LoadingSpinner size="sm" />
        </div>
      </div>
      <div className="text-center">
        <p className="text-sm font-medium text-foreground">{message}</p>
        {submessage && <p className="text-xs text-muted-foreground mt-0.5">{submessage}</p>}
      </div>
    </div>
  )
}
