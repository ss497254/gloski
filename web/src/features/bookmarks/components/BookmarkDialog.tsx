import { useState, useEffect } from 'react'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/ui/dialog'
import type { Bookmark } from '../stores/bookmarks'

interface BookmarkDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  bookmark: Bookmark | null
  folders: string[]
  onSubmit: (data: { title: string; url: string; description?: string; folder?: string; tags: string[] }) => void
}

export function BookmarkDialog({ open, onOpenChange, bookmark, folders, onSubmit }: BookmarkDialogProps) {
  const [title, setTitle] = useState('')
  const [url, setUrl] = useState('')
  const [description, setDescription] = useState('')
  const [folder, setFolder] = useState('')
  const [tags, setTags] = useState('')

  const isEditing = !!bookmark

  // Reset form when dialog opens/closes or bookmark changes
  useEffect(() => {
    if (open && bookmark) {
      setTitle(bookmark.title)
      setUrl(bookmark.url)
      setDescription(bookmark.description || '')
      setFolder(bookmark.folder || '')
      setTags(bookmark.tags.join(', '))
    } else if (open) {
      setTitle('')
      setUrl('')
      setDescription('')
      setFolder('')
      setTags('')
    }
  }, [open, bookmark])

  const handleSubmit = () => {
    const parsedTags = tags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)

    onSubmit({
      title,
      url,
      description: description || undefined,
      folder: folder || undefined,
      tags: parsedTags,
    })
  }

  const isValid = title.trim() && url.trim()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Bookmark' : 'Add Bookmark'}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Title</label>
            <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="My Bookmark" />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">URL</label>
            <Input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://example.com" />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Description</label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description"
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Folder</label>
            <Input
              value={folder}
              onChange={(e) => setFolder(e.target.value)}
              placeholder="e.g., Development"
              list="folders"
            />
            <datalist id="folders">
              {folders.map((f) => (
                <option key={f} value={f} />
              ))}
            </datalist>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Tags</label>
            <Input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="tag1, tag2, tag3" />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={!isValid}>
            {isEditing ? 'Save' : 'Add'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
