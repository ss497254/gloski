import type { Download } from '@/shared/lib/types'
import { cn, formatBytes, formatRelativeTime } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import {
  AlertCircle,
  CheckCircle2,
  Clock,
  Download as DownloadIcon,
  FileIcon,
  Link2,
  Loader2,
  MoreVertical,
  Pause,
  Play,
  RotateCcw,
  Share2,
  Trash2,
  X,
  XCircle,
} from 'lucide-react'
import { memo, useState } from 'react'
import { useDownloads } from '../context'
import { DownloadProgress } from './DownloadProgress'

interface DownloadItemProps {
  download: Download
}

const statusConfig: Record<
  string,
  {
    icon: typeof Clock
    label: string
    color: string
    bgColor: string
    animate?: boolean
  }
> = {
  pending: {
    icon: Clock,
    label: 'Pending',
    color: 'text-yellow-500',
    bgColor: 'bg-yellow-500/10',
  },
  downloading: {
    icon: Loader2,
    label: 'Downloading',
    color: 'text-blue-500',
    bgColor: 'bg-blue-500/10',
    animate: true,
  },
  paused: {
    icon: Pause,
    label: 'Paused',
    color: 'text-orange-500',
    bgColor: 'bg-orange-500/10',
  },
  completed: {
    icon: CheckCircle2,
    label: 'Completed',
    color: 'text-green-500',
    bgColor: 'bg-green-500/10',
  },
  failed: {
    icon: XCircle,
    label: 'Failed',
    color: 'text-red-500',
    bgColor: 'bg-red-500/10',
  },
  cancelled: {
    icon: X,
    label: 'Cancelled',
    color: 'text-gray-500',
    bgColor: 'bg-gray-500/10',
  },
}

export const DownloadItem = memo(function DownloadItem({ download }: DownloadItemProps) {
  const {
    handlePause,
    handleResume,
    handleCancel,
    handleRetry,
    handleDelete,
    handleDownload,
    setShareDownload,
  } = useDownloads()

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const handleShare = () => setShareDownload(download)

  const status = statusConfig[download.status]
  const StatusIcon = status.icon
  const isActive = download.status === 'downloading' || download.status === 'pending'
  const canPause = download.status === 'downloading'
  const canResume = download.status === 'paused'
  const canCancel = isActive || download.status === 'paused'
  const canRetry = download.status === 'failed' || download.status === 'cancelled'
  const canShare = download.status === 'completed'
  const canDownloadFile = download.status === 'completed'

  const shareLinksCount = download.share_links?.length || 0

  return (
    <div className="rounded-lg border bg-card p-4 shadow-sm">
      <div className="flex items-start gap-4">
        {/* File icon */}
        <div className={cn('rounded-lg p-3', status.bgColor)}>
          <FileIcon className={cn('h-6 w-6', status.color)} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Filename and status */}
          <div className="flex items-center gap-2 mb-1">
            <h3 className="font-medium truncate">{download.filename}</h3>
            <span
              className={cn(
                'inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full',
                status.bgColor,
                status.color
              )}
            >
              <StatusIcon className={cn('h-3 w-3', status.animate && 'animate-spin')} />
              {status.label}
            </span>
            {shareLinksCount > 0 && (
              <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
                <Link2 className="h-3 w-3" />
                {shareLinksCount} link{shareLinksCount > 1 ? 's' : ''}
              </span>
            )}
          </div>

          {/* URL */}
          <p className="text-xs text-muted-foreground truncate mb-2" title={download.url}>
            {download.url}
          </p>

          {/* Progress bar for active downloads */}
          {(download.status === 'downloading' || download.status === 'paused') && (
            <DownloadProgress
              progress={download.progress}
              total={download.total}
              speed={download.speed}
              status={download.status}
            />
          )}

          {/* Error message */}
          {download.error && (
            <div className="flex items-center gap-2 mt-2 text-sm text-red-500">
              <AlertCircle className="h-4 w-4" />
              <span>{download.error}</span>
            </div>
          )}

          {/* Bottom info row */}
          <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
            <span title={download.destination}>{download.destination}</span>
            {download.status === 'completed' && download.total > 0 && <span>{formatBytes(download.total)}</span>}
            <span>{formatRelativeTime(download.created_at)}</span>
            {download.retries > 0 && (
              <span>
                Retries: {download.retries}/{download.max_retries}
              </span>
            )}
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1">
          {/* Primary actions */}
          {canPause && (
            <Button variant="ghost" size="icon" onClick={() => handlePause(download.id)} title="Pause">
              <Pause className="h-4 w-4" />
            </Button>
          )}
          {canResume && (
            <Button variant="ghost" size="icon" onClick={() => handleResume(download.id)} title="Resume">
              <Play className="h-4 w-4" />
            </Button>
          )}
          {canCancel && (
            <Button variant="ghost" size="icon" onClick={() => handleCancel(download.id)} title="Cancel">
              <X className="h-4 w-4" />
            </Button>
          )}
          {canRetry && (
            <Button variant="ghost" size="icon" onClick={() => handleRetry(download.id)} title="Retry">
              <RotateCcw className="h-4 w-4" />
            </Button>
          )}
          {canDownloadFile && (
            <Button variant="ghost" size="icon" onClick={() => handleDownload(download)} title="Download">
              <DownloadIcon className="h-4 w-4" />
            </Button>
          )}
          {canShare && (
            <Button variant="ghost" size="icon" onClick={handleShare} title="Share">
              <Share2 className="h-4 w-4" />
            </Button>
          )}

          {/* More actions dropdown */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {canDownloadFile && (
                <DropdownMenuItem onClick={() => handleDownload(download)}>
                  <DownloadIcon className="h-4 w-4 mr-2" />
                  Download to browser
                </DropdownMenuItem>
              )}
              {canShare && (
                <DropdownMenuItem onClick={handleShare}>
                  <Share2 className="h-4 w-4 mr-2" />
                  Share
                </DropdownMenuItem>
              )}
              {(canDownloadFile || canShare) && <DropdownMenuSeparator />}
              <DropdownMenuItem
                onClick={() => setShowDeleteConfirm(true)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Delete confirmation */}
      {showDeleteConfirm && (
        <div className="mt-4 p-3 bg-destructive/10 rounded-lg border border-destructive/20">
          <p className="text-sm mb-3">Delete this download?</p>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => setShowDeleteConfirm(false)}>
              Cancel
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                handleDelete(download.id, false)
                setShowDeleteConfirm(false)
              }}
            >
              Remove from list
            </Button>
            {download.status === 'completed' && (
              <Button
                variant="destructive"
                size="sm"
                onClick={() => {
                  handleDelete(download.id, true)
                  setShowDeleteConfirm(false)
                }}
              >
                Delete file too
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  )
})
