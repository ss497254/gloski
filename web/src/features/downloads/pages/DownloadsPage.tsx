import { useServer } from '@/features/servers/context'
import { PageLayout } from '@/layouts/PageLayout'
import { EmptyState } from '@/shared/components'
import { Button } from '@/ui/button'
import { Download as DownloadIcon, Loader2, Plus, RefreshCw } from 'lucide-react'
import { AddDownloadDialog, DownloadItem, ShareDialog } from '../components'
import { DownloadsProvider, useDownloads, type DownloadFilter } from '../context'

// Filter buttons config
const FILTER_BUTTONS: { value: DownloadFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
]

function DownloadsContent() {
  const { server } = useServer()
  const {
    downloads,
    sortedDownloads,
    loading,
    filter,
    setFilter,
    showAddDialog,
    setShowAddDialog,
    shareDownload,
    setShareDownload,
    handleAdd,
    handleCreateShareLink,
    handleRevokeShareLink,
    refresh,
  } = useDownloads()

  return (
    <PageLayout
      title="Downloads"
      description="Download files from URLs to your server"
      actions={
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={refresh} disabled={loading}>
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
            <DownloadItem key={download.id} download={download} />
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

export function DownloadsPage() {
  return (
    <DownloadsProvider>
      <DownloadsContent />
    </DownloadsProvider>
  )
}
