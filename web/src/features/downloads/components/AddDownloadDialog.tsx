import type { Server } from '@/features/servers'
import type { FileEntry } from '@/shared/lib/types'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/ui/dialog'
import { Input } from '@/ui/input'
import { ArrowLeft, ChevronRight, Download, Folder, FolderOpen, HardDrive, Home, Loader2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

interface AddDownloadDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onAdd: (url: string, destination: string, filename?: string) => void
  server: Server
}

const defaultDestinations = [
  { label: 'Downloads', path: '~/Downloads', icon: Download },
  { label: 'Home', path: '~', icon: Home },
  { label: 'Temp', path: '/tmp', icon: HardDrive },
]

export function AddDownloadDialog({ open, onOpenChange, onAdd, server }: AddDownloadDialogProps) {
  const [url, setUrl] = useState('')
  const [destination, setDestination] = useState('~/Downloads')
  const [filename, setFilename] = useState('')
  const [showBrowser, setShowBrowser] = useState(false)
  const [browserPath, setBrowserPath] = useState('~')
  const [browserEntries, setBrowserEntries] = useState<FileEntry[]>([])
  const [browserLoading, setBrowserLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const loadDirectory = useCallback(
    async (path: string) => {
      setBrowserLoading(true)
      try {
        const response = await server.getClient().files.list(path)
        // Filter to only show directories
        setBrowserEntries(response.entries.filter((e) => e.type === 'directory'))
      } catch {
        setBrowserEntries([])
      }
      setBrowserLoading(false)
    },
    [server]
  )

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      setUrl('')
      setFilename('')
      setShowBrowser(false)
    }
  }, [open])

  // Load directory when browser path changes
  useEffect(() => {
    if (showBrowser) {
      loadDirectory(browserPath)
    }
  }, [browserPath, showBrowser, loadDirectory])

  const handleSubmit = async () => {
    if (!url.trim() || !destination.trim()) return
    setSubmitting(true)
    try {
      await onAdd(url.trim(), destination.trim(), filename.trim() || undefined)
    } finally {
      setSubmitting(false)
    }
  }

  const selectDestination = (path: string) => {
    setDestination(path)
    setShowBrowser(false)
  }

  const navigateToParent = () => {
    const parts = browserPath.split('/')
    if (parts.length > 1) {
      parts.pop()
      setBrowserPath(parts.join('/') || '/')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Add Download</DialogTitle>
          <DialogDescription>Enter a URL to download a file to your server</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* URL Input */}
          <div className="space-y-2">
            <label className="text-sm font-medium">URL</label>
            <Input
              placeholder="https://example.com/file.zip"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              autoFocus
            />
          </div>

          {/* Destination */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Destination Folder</label>

            {!showBrowser ? (
              <>
                {/* Quick select buttons */}
                <div className="flex flex-wrap gap-2 mb-2">
                  {defaultDestinations.map((dest) => (
                    <Button
                      key={dest.path}
                      variant={destination === dest.path ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setDestination(dest.path)}
                    >
                      <dest.icon className="h-4 w-4 mr-1" />
                      {dest.label}
                    </Button>
                  ))}
                </div>

                {/* Custom path input */}
                <div className="flex gap-2">
                  <Input
                    value={destination}
                    onChange={(e) => setDestination(e.target.value)}
                    placeholder="/path/to/folder"
                  />
                  <Button
                    variant="outline"
                    onClick={() => {
                      setBrowserPath(destination.startsWith('~') ? destination : destination || '/')
                      setShowBrowser(true)
                    }}
                  >
                    <FolderOpen className="h-4 w-4" />
                  </Button>
                </div>
              </>
            ) : (
              /* Folder browser */
              <div className="border rounded-lg">
                {/* Browser header */}
                <div className="flex items-center gap-2 p-2 border-b bg-muted/50">
                  <Button variant="ghost" size="icon" onClick={() => setShowBrowser(false)} className="h-8 w-8">
                    <ArrowLeft className="h-4 w-4" />
                  </Button>
                  <span className="text-sm font-medium truncate flex-1">{browserPath}</span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={navigateToParent}
                    disabled={browserPath === '/' || browserPath === '~'}
                  >
                    Up
                  </Button>
                </div>

                {/* Quick destinations in browser */}
                <div className="flex flex-wrap gap-1 p-2 border-b">
                  {defaultDestinations.map((dest) => (
                    <Button
                      key={dest.path}
                      variant="ghost"
                      size="sm"
                      onClick={() => setBrowserPath(dest.path)}
                      className="h-7 text-xs"
                    >
                      <dest.icon className="h-3 w-3 mr-1" />
                      {dest.label}
                    </Button>
                  ))}
                </div>

                {/* Directory list */}
                <div className="max-h-48 overflow-y-auto">
                  {browserLoading ? (
                    <div className="flex items-center justify-center py-8">
                      <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                    </div>
                  ) : browserEntries.length === 0 ? (
                    <div className="text-center py-8 text-sm text-muted-foreground">No subfolders</div>
                  ) : (
                    browserEntries.map((entry) => (
                      <button
                        key={entry.path}
                        className={cn(
                          'flex items-center gap-2 w-full px-3 py-2 text-left text-sm',
                          'hover:bg-muted transition-colors'
                        )}
                        onClick={() => setBrowserPath(entry.path)}
                        onDoubleClick={() => selectDestination(entry.path)}
                      >
                        <Folder className="h-4 w-4 text-muted-foreground" />
                        <span className="truncate">{entry.name}</span>
                        <ChevronRight className="h-4 w-4 ml-auto text-muted-foreground" />
                      </button>
                    ))
                  )}
                </div>

                {/* Select button */}
                <div className="flex justify-end gap-2 p-2 border-t bg-muted/50">
                  <Button variant="outline" size="sm" onClick={() => setShowBrowser(false)}>
                    Cancel
                  </Button>
                  <Button size="sm" onClick={() => selectDestination(browserPath)}>
                    Select "{browserPath.split('/').pop() || browserPath}"
                  </Button>
                </div>
              </div>
            )}
          </div>

          {/* Optional filename */}
          <div className="space-y-2">
            <label className="text-sm font-medium">
              Filename <span className="text-muted-foreground font-normal">(optional)</span>
            </label>
            <Input placeholder="Auto-detect from URL" value={filename} onChange={(e) => setFilename(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={!url.trim() || !destination.trim() || submitting}>
            {submitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            Add Download
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
