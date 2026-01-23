import { useState, useMemo } from 'react'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Card, CardContent } from '@/ui/card'
import { Badge } from '@/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import { PageLayout } from '@/layouts'
import { SearchInput, FilterSidebar, EmptyState } from '@/shared/components'
import { useBookmarksStore, type Bookmark } from '../stores/bookmarks'
import {
  Plus,
  ExternalLink,
  Folder,
  MoreVertical,
  Pencil,
  Trash2,
  Globe,
} from 'lucide-react'

function BookmarkCard({
  bookmark,
  onEdit,
  onDelete,
}: {
  bookmark: Bookmark
  onEdit: (b: Bookmark) => void
  onDelete: (id: string) => void
}) {
  return (
    <Card className="group hover:border-primary/50 transition-colors">
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center shrink-0">
            <Globe className="h-5 w-5 text-muted-foreground" />
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2">
              <a
                href={bookmark.url}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium hover:text-primary truncate block"
              >
                {bookmark.title}
              </a>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                  >
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => window.open(bookmark.url, '_blank')}>
                    <ExternalLink className="h-4 w-4 mr-2" />
                    Open
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => onEdit(bookmark)}>
                    <Pencil className="h-4 w-4 mr-2" />
                    Edit
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => onDelete(bookmark.id)}
                    className="text-destructive"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            {bookmark.description && (
              <p className="text-sm text-muted-foreground line-clamp-2 mt-1">
                {bookmark.description}
              </p>
            )}

            <div className="flex items-center gap-2 mt-2 flex-wrap">
              {bookmark.folder && (
                <Badge variant="secondary" className="text-xs">
                  <Folder className="h-3 w-3 mr-1" />
                  {bookmark.folder}
                </Badge>
              )}
              {bookmark.tags.slice(0, 3).map((tag) => (
                <Badge key={tag} variant="outline" className="text-xs">
                  {tag}
                </Badge>
              ))}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export function BookmarksPage() {
  const { bookmarks, folders, addBookmark, updateBookmark, deleteBookmark } =
    useBookmarksStore()

  const [search, setSearch] = useState('')
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingBookmark, setEditingBookmark] = useState<Bookmark | null>(null)

  // Form state
  const [formTitle, setFormTitle] = useState('')
  const [formUrl, setFormUrl] = useState('')
  const [formDescription, setFormDescription] = useState('')
  const [formFolder, setFormFolder] = useState('')
  const [formTags, setFormTags] = useState('')

  const filteredBookmarks = useMemo(() => {
    return bookmarks.filter((b) => {
      const matchesSearch =
        !search ||
        b.title.toLowerCase().includes(search.toLowerCase()) ||
        b.description?.toLowerCase().includes(search.toLowerCase()) ||
        b.tags.some((t) => t.toLowerCase().includes(search.toLowerCase()))

      const matchesFolder = !selectedFolder || b.folder === selectedFolder

      return matchesSearch && matchesFolder
    })
  }, [bookmarks, search, selectedFolder])

  const openDialog = (bookmark?: Bookmark) => {
    if (bookmark) {
      setEditingBookmark(bookmark)
      setFormTitle(bookmark.title)
      setFormUrl(bookmark.url)
      setFormDescription(bookmark.description || '')
      setFormFolder(bookmark.folder || '')
      setFormTags(bookmark.tags.join(', '))
    } else {
      setEditingBookmark(null)
      setFormTitle('')
      setFormUrl('')
      setFormDescription('')
      setFormFolder('')
      setFormTags('')
    }
    setDialogOpen(true)
  }

  const handleSubmit = () => {
    const tags = formTags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)

    if (editingBookmark) {
      updateBookmark(editingBookmark.id, {
        title: formTitle,
        url: formUrl,
        description: formDescription || undefined,
        folder: formFolder || undefined,
        tags,
      })
    } else {
      addBookmark({
        title: formTitle,
        url: formUrl,
        description: formDescription || undefined,
        folder: formFolder || undefined,
        tags,
      })
    }

    setDialogOpen(false)
  }

  return (
    <PageLayout
      title="Bookmarks"
      description="Save and organize your favorite links"
      actions={
        <Button onClick={() => openDialog()}>
          <Plus className="h-4 w-4 mr-2" />
          Add Bookmark
        </Button>
      }
    >
      <div className="flex gap-6">
        {/* Sidebar - Folders */}
        <FilterSidebar
          selected={selectedFolder}
          onSelect={setSelectedFolder}
          allItem={{
            label: 'All Bookmarks',
            icon: Globe,
            count: bookmarks.length,
          }}
          sections={[
            {
              title: 'Folders',
              items: folders.map((folder) => ({
                id: folder,
                label: folder,
                icon: Folder,
                count: bookmarks.filter((b) => b.folder === folder).length,
              })),
            },
          ]}
        />

        {/* Main content */}
        <div className="flex-1 space-y-4">
          <SearchInput
            value={search}
            onChange={setSearch}
            placeholder="Search bookmarks..."
          />

          {/* Bookmarks grid */}
          {filteredBookmarks.length > 0 ? (
            <div className="grid gap-3 md:grid-cols-2">
              {filteredBookmarks.map((bookmark) => (
                <BookmarkCard
                  key={bookmark.id}
                  bookmark={bookmark}
                  onEdit={openDialog}
                  onDelete={deleteBookmark}
                />
              ))}
            </div>
          ) : (
            <EmptyState
              icon={Globe}
              title="No bookmarks found"
              description={search ? 'Try a different search term' : 'Add your first bookmark to get started'}
            />
          )}
        </div>
      </div>

      {/* Add/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingBookmark ? 'Edit Bookmark' : 'Add Bookmark'}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Title</label>
              <Input
                value={formTitle}
                onChange={(e) => setFormTitle(e.target.value)}
                placeholder="My Bookmark"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">URL</label>
              <Input
                value={formUrl}
                onChange={(e) => setFormUrl(e.target.value)}
                placeholder="https://example.com"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Description</label>
              <Input
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder="Optional description"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Folder</label>
              <Input
                value={formFolder}
                onChange={(e) => setFormFolder(e.target.value)}
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
              <Input
                value={formTags}
                onChange={(e) => setFormTags(e.target.value)}
                placeholder="tag1, tag2, tag3"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} disabled={!formTitle || !formUrl}>
              {editingBookmark ? 'Save' : 'Add'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageLayout>
  )
}
