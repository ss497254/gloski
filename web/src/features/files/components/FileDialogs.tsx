import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/ui/dialog'
import type { FileEntry } from '@/shared/lib/types'

interface NewFolderDialogProps {
  open: boolean
  name: string
  onOpenChange: (open: boolean) => void
  onNameChange: (name: string) => void
  onCreate: () => void
}

export function NewFolderDialog({ open, name, onOpenChange, onNameChange, onCreate }: NewFolderDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Folder</DialogTitle>
          <DialogDescription>Enter a name for the new folder</DialogDescription>
        </DialogHeader>
        <Input
          placeholder="Folder name"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && onCreate()}
          autoFocus
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={onCreate}>Create</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface NewFileDialogProps {
  open: boolean
  name: string
  onOpenChange: (open: boolean) => void
  onNameChange: (name: string) => void
  onCreate: () => void
}

export function NewFileDialog({ open, name, onOpenChange, onNameChange, onCreate }: NewFileDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New File</DialogTitle>
          <DialogDescription>Enter a name for the new file</DialogDescription>
        </DialogHeader>
        <Input
          placeholder="File name (e.g., index.ts)"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && onCreate()}
          autoFocus
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={onCreate}>Create</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface DeleteDialogProps {
  entry: FileEntry | null
  onClose: () => void
  onDelete: () => void
}

export function DeleteDialog({ entry, onClose, onDelete }: DeleteDialogProps) {
  return (
    <Dialog open={!!entry} onOpenChange={() => onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete {entry?.type === 'directory' ? 'Folder' : 'File'}</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete "{entry?.name}"? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onDelete}>
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface RenameDialogProps {
  entry: FileEntry | null
  name: string
  onClose: () => void
  onNameChange: (name: string) => void
  onRename: () => void
}

export function RenameDialog({ entry, name, onClose, onNameChange, onRename }: RenameDialogProps) {
  const handleRename = () => {
    if (!name.trim()) return
    if (name.includes('/') || name.includes('\\')) return
    onRename()
  }

  return (
    <Dialog open={!!entry} onOpenChange={() => onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Rename {entry?.type === 'directory' ? 'Folder' : 'File'}</DialogTitle>
          <DialogDescription>Enter a new name for "{entry?.name}"</DialogDescription>
        </DialogHeader>
        <Input
          placeholder="New name"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleRename()}
          autoFocus
        />
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={handleRename} disabled={!name.trim() || name === entry?.name}>
            Rename
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface BulkDeleteDialogProps {
  count: number
  open: boolean
  onClose: () => void
  onDelete: () => void
}

export function BulkDeleteDialog({ count, open, onClose, onDelete }: BulkDeleteDialogProps) {
  return (
    <Dialog open={open} onOpenChange={() => onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete {count} Items</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete {count} selected item{count !== 1 ? 's' : ''}? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onDelete}>
            Delete {count} Item{count !== 1 ? 's' : ''}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
