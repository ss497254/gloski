import type { Download, ShareLink } from '@/shared/lib/types'
import { formatRelativeTime } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/ui/dialog'
import { Clock, Copy, Link2, Loader2, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

interface ShareDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  download: Download
  onCreateLink: (id: string, expiresIn?: number) => Promise<ShareLink | undefined>
  onRevokeLink: (downloadId: string, token: string) => void
}

const expiryOptions = [
  { label: '1 hour', value: 3600 },
  { label: '24 hours', value: 86400 },
  { label: '7 days', value: 604800 },
  { label: '30 days', value: 2592000 },
  { label: 'Never', value: undefined },
]

export function ShareDialog({ open, onOpenChange, download, onCreateLink, onRevokeLink }: ShareDialogProps) {
  const [selectedExpiry, setSelectedExpiry] = useState<number | undefined>(86400) // 24 hours default
  const [creating, setCreating] = useState(false)

  const shareLinks = download.share_links || []

  const handleCreateLink = async () => {
    setCreating(true)
    try {
      await onCreateLink(download.id, selectedExpiry)
    } finally {
      setCreating(false)
    }
  }

  const copyToClipboard = async (url: string) => {
    await navigator.clipboard.writeText(url)
    toast.success('Link copied to clipboard')
  }

  const formatExpiry = (link: ShareLink): string => {
    if (!link.expires_at) return 'Never expires'
    const expiresAt = new Date(link.expires_at)
    const now = new Date()
    if (expiresAt < now) return 'Expired'
    return `Expires ${formatRelativeTime(link.expires_at)}`
  }

  const isExpired = (link: ShareLink): boolean => {
    if (!link.expires_at) return false
    return new Date(link.expires_at) < new Date()
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Share "{download.filename}"</DialogTitle>
          <DialogDescription>Create shareable links for this download</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Create new link */}
          <div className="space-y-3">
            <label className="text-sm font-medium">Create a share link</label>
            <div className="flex flex-wrap gap-2">
              {expiryOptions.map((option) => (
                <Button
                  key={option.label}
                  variant={selectedExpiry === option.value ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setSelectedExpiry(option.value)}
                >
                  {option.label}
                </Button>
              ))}
            </div>
            <Button onClick={handleCreateLink} disabled={creating} className="w-full">
              {creating ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Plus className="h-4 w-4 mr-2" />}
              Create Link
            </Button>
          </div>

          {/* Existing links */}
          {shareLinks.length > 0 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium">Active Links</label>
                <span className="text-xs text-muted-foreground">
                  {shareLinks.length} link{shareLinks.length > 1 ? 's' : ''}
                </span>
              </div>
              <div className="space-y-2 max-h-64 overflow-y-auto">
                {shareLinks.map((link) => {
                  const expired = isExpired(link)
                  return (
                    <div
                      key={link.token}
                      className={`flex items-center gap-2 p-3 rounded-lg border ${
                        expired ? 'bg-muted/50 opacity-60' : 'bg-muted/30'
                      }`}
                    >
                      <Link2 className="h-4 w-4 text-muted-foreground shrink-0" />
                      <div className="flex-1 min-w-0">
                        <p className="text-xs font-mono truncate" title={link.url}>
                          {link.url}
                        </p>
                        <p className="text-xs text-muted-foreground flex items-center gap-1 mt-0.5">
                          <Clock className="h-3 w-3" />
                          {formatExpiry(link)}
                        </p>
                      </div>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => copyToClipboard(link.url)}
                          title="Copy link"
                        >
                          <Copy className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-destructive hover:text-destructive"
                          onClick={() => onRevokeLink(download.id, link.token)}
                          title="Revoke link"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {shareLinks.length === 0 && (
            <p className="text-sm text-muted-foreground text-center py-4">No share links yet. Create one above.</p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
