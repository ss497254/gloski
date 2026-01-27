import { useServer } from '@/features/servers/context'
import { PageLayout } from '@/layouts/PageLayout'
import { EmptyState } from '@/shared/components'
import type { Download } from '@/shared/lib/types'
import { Button } from '@/ui/button'
import { Download as DownloadIcon, Loader2, Plus, RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { AddDownloadDialog, DownloadItem, ShareDialog } from '../components'

// Filter buttons config moved outside component to avoid recreation
const FILTER_BUTTONS = [
  { value: 'all', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
] as const

export function DownloadsPage() {
  const { server, serverId } = useServer()
  const [downloads, setDownloads] = useState<Download[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [shareDownload, setShareDownload] = useState<Download | null>(null)
  const [filter, setFilter] = useState<'all' | 'active' | 'completed' | 'failed'>('all')

  const fetchDownloads = useCallback(async () => {
    try {
      const response = await server.getClient().downloads.list()
      setDownloads(response.downloads || [])
    } catch {
      // Silently fail for polling
    }
  }, [server])

  const loadData = useCallback(async () => {
    setLoading(true)
    await fetchDownloads()
    setLoading(false)
  }, [fetchDownloads])

  // Initial load
  useEffect(() => {
    loadData()
  }, [serverId, server, loadData])

  // Polling - faster when there are active downloads
  useEffect(() => {
    const hasActive = downloads.some((d) => d.status === 'downloading' || d.status === 'pending')
    const interval = setInterval(fetchDownloads, hasActive ? 1000 : 10000)
    return () => clearInterval(interval)
  }, [downloads, fetchDownloads, server])

  const handleAdd = useCallback(
    async (url: string, destination: string, filename?: string) => {
      try {
        await server.getClient().downloads.add(url, destination, filename)
        toast.success('Download added')
        fetchDownloads()
        setShowAddDialog(false)
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to add download')
      }
    },
    [server, fetchDownloads]
  )

  const handlePause = useCallback(
    async (id: string) => {
      try {
        await server.getClient().downloads.pause(id)
        toast.success('Download paused')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to pause download')
      }
    },
    [server, fetchDownloads]
  )

  const handleResume = useCallback(
    async (id: string) => {
      try {
        await server.getClient().downloads.resume(id)
        toast.success('Download resumed')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to resume download')
      }
    },
    [server, fetchDownloads]
  )

  const handleCancel = useCallback(
    async (id: string) => {
      try {
        await server.getClient().downloads.cancel(id)
        toast.success('Download cancelled')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to cancel download')
      }
    },
    [server, fetchDownloads]
  )

  const handleRetry = useCallback(
    async (id: string) => {
      try {
        await server.getClient().downloads.retry(id)
        toast.success('Retrying download')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to retry download')
      }
    },
    [server, fetchDownloads]
  )

  const handleDelete = useCallback(
    async (id: string, deleteFile: boolean) => {
      try {
        await server.getClient().downloads.delete(id, deleteFile)
        toast.success('Download deleted')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to delete download')
      }
    },
    [server, fetchDownloads]
  )

  const handleDownloadFile = useCallback(
    (download: Download) => {
      const url = server.getClient().downloads.getDownloadUrl(download.id)
      window.open(url, '_blank')
    },
    [server]
  )

  const handleCreateShareLink = useCallback(
    async (id: string, expiresIn?: number) => {
      try {
        const link = await server
          .getClient()
          .downloads.createShareLink(id, expiresIn ? { expires_in: expiresIn } : undefined)
        await navigator.clipboard.writeText(link.url)
        toast.success('Share link copied to clipboard')
        fetchDownloads()
        return link
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to create share link')
        throw err
      }
    },
    [server, fetchDownloads]
  )

  const handleRevokeShareLink = useCallback(
    async (downloadId: string, token: string) => {
      try {
        await server.getClient().downloads.revokeShareLink(downloadId, token)
        toast.success('Share link revoked')
        fetchDownloads()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Failed to revoke share link')
      }
    },
    [server, fetchDownloads]
  )

  // Filter and sort downloads - memoized to avoid unnecessary recalculations
  const sortedDownloads = useMemo(() => {
    const filtered = downloads.filter((d) => {
      switch (filter) {
        case 'active':
          return d.status === 'downloading' || d.status === 'pending' || d.status === 'paused'
        case 'completed':
          return d.status === 'completed'
        case 'failed':
          return d.status === 'failed' || d.status === 'cancelled'
        default:
          return true
      }
    })
    // Sort by created_at (newest first)
    return [...filtered].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
  }, [downloads, filter])

  return (
    <PageLayout
      title="Downloads"
      description="Download files from URLs to your server"
      actions={
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={loadData} disabled={loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setShowAddDialog(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Add Download
          </Button>
        </div>
      }
    >
      {/* Filter tabs */}
      <div className="flex gap-1 mb-4 border-b">
        {FILTER_BUTTONS.map((btn) => (
          <button
            key={btn.value}
            onClick={() => setFilter(btn.value)}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              filter === btn.value
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {btn.label}
            {btn.value === 'all' && ` (${downloads.length})`}
            {btn.value === 'active' &&
              ` (${downloads.filter((d) => ['downloading', 'pending', 'paused'].includes(d.status)).length})`}
            {btn.value === 'completed' && ` (${downloads.filter((d) => d.status === 'completed').length})`}
            {btn.value === 'failed' &&
              ` (${downloads.filter((d) => ['failed', 'cancelled'].includes(d.status)).length})`}
          </button>
        ))}
      </div>

      {/* Downloads list */}
      {loading && downloads.length === 0 ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : sortedDownloads.length === 0 ? (
        <EmptyState
          icon={DownloadIcon}
          title={filter === 'all' ? 'No downloads yet' : `No ${filter} downloads`}
          description={
            filter === 'all' ? 'Add a download to get started' : 'Downloads matching this filter will appear here'
          }
          action={
            filter === 'all' ? (
              <Button onClick={() => setShowAddDialog(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Add Download
              </Button>
            ) : undefined
          }
        />
      ) : (
        <div className="space-y-3">
          {sortedDownloads.map((download) => (
            <DownloadItem
              key={download.id}
              download={download}
              onPause={handlePause}
              onResume={handleResume}
              onCancel={handleCancel}
              onRetry={handleRetry}
              onDelete={handleDelete}
              onDownload={handleDownloadFile}
              onShare={setShareDownload}
            />
          ))}
        </div>
      )}

      {/* Dialogs */}
      <AddDownloadDialog open={showAddDialog} onOpenChange={setShowAddDialog} onAdd={handleAdd} server={server} />

      {shareDownload && (
        <ShareDialog
          open={!!shareDownload}
          onOpenChange={(open) => !open && setShareDownload(null)}
          download={shareDownload}
          onCreateLink={handleCreateShareLink}
          onRevokeLink={handleRevokeShareLink}
        />
      )}
    </PageLayout>
  )
}
