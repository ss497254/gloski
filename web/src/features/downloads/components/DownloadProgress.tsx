import { formatBytes } from '@/shared/lib/utils'
import type { DownloadStatus } from '@/shared/lib/types'

interface DownloadProgressProps {
  progress: number
  total: number
  speed: number
  status: DownloadStatus
}

export function DownloadProgress({ progress, total, speed, status }: DownloadProgressProps) {
  const hasTotal = total > 0
  const percentage = hasTotal ? Math.round((progress / total) * 100) : 0
  const isPaused = status === 'paused'

  // Calculate ETA
  const etaSeconds = speed > 0 && hasTotal ? Math.round((total - progress) / speed) : 0
  const etaFormatted = formatEta(etaSeconds)

  return (
    <div className="space-y-1">
      {/* Progress bar */}
      <div className="h-2 bg-muted rounded-full overflow-hidden">
        <div
          className={`h-full transition-all duration-300 ${isPaused ? 'bg-orange-500' : 'bg-primary'}`}
          style={{ width: hasTotal ? `${percentage}%` : '100%' }}
        />
      </div>

      {/* Stats */}
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <div className="flex items-center gap-2">
          {hasTotal ? (
            <>
              <span>{percentage}%</span>
              <span className="text-muted-foreground/60">
                {formatBytes(progress)} / {formatBytes(total)}
              </span>
            </>
          ) : (
            <span>{formatBytes(progress)}</span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {!isPaused && speed > 0 && <span>{formatBytes(speed)}/s</span>}
          {!isPaused && etaFormatted && hasTotal && (
            <span className="text-muted-foreground/60">{etaFormatted} left</span>
          )}
          {isPaused && <span className="text-orange-500">Paused</span>}
        </div>
      </div>
    </div>
  )
}

function formatEta(seconds: number): string {
  if (seconds <= 0) return ''
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${mins}m`
}
