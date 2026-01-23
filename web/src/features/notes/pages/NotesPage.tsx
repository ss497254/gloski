import { useState, useMemo, useEffect } from 'react'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Badge } from '@/ui/badge'
import { ScrollArea } from '@/ui/scroll-area'
import { PageLayout } from '@/layouts'
import { useNotesStore, type Note } from '../stores/notes'
import { cn, formatRelativeTime } from '@/shared/lib/utils'
import {
  Plus,
  Search,
  FileText,
  Folder,
  Pin,
  Trash2,
  MoreVertical,
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'

function NoteItem({
  note,
  isSelected,
  onClick,
  onTogglePin,
  onDelete,
}: {
  note: Note
  isSelected: boolean
  onClick: () => void
  onTogglePin: () => void
  onDelete: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'w-full text-left p-3 rounded-lg transition-colors group',
        isSelected ? 'bg-accent' : 'hover:bg-muted/50'
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            {note.pinned && (
              <Pin className="h-3 w-3 text-primary shrink-0 fill-current" />
            )}
            <span className="font-medium truncate">{note.title}</span>
          </div>
          <p className="text-sm text-muted-foreground line-clamp-2 mt-1">
            {note.content.slice(0, 100)}
          </p>
          <div className="flex items-center gap-2 mt-2">
            <span className="text-xs text-muted-foreground">
              {formatRelativeTime(note.updatedAt)}
            </span>
            {note.folder && (
              <Badge variant="secondary" className="text-xs h-5">
                {note.folder}
              </Badge>
            )}
          </div>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 opacity-0 group-hover:opacity-100 shrink-0"
              onClick={(e) => e.stopPropagation()}
            >
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onTogglePin}>
              <Pin className="h-4 w-4 mr-2" />
              {note.pinned ? 'Unpin' : 'Pin'}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={onDelete} className="text-destructive">
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </button>
  )
}

export function NotesPage() {
  const {
    notes,
    folders,
    selectedNoteId,
    addNote,
    updateNote,
    deleteNote,
    togglePin,
    selectNote,
  } = useNotesStore()

  const [search, setSearch] = useState('')
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null)
  const [editTitle, setEditTitle] = useState('')
  const [editContent, setEditContent] = useState('')

  const selectedNote = notes.find((n) => n.id === selectedNoteId)

  // Sync edit state with selected note
  useEffect(() => {
    if (selectedNote) {
      setEditTitle(selectedNote.title)
      setEditContent(selectedNote.content)
    }
  }, [selectedNoteId])

  const filteredNotes = useMemo(() => {
    return notes
      .filter((n) => {
        const matchesSearch =
          !search ||
          n.title.toLowerCase().includes(search.toLowerCase()) ||
          n.content.toLowerCase().includes(search.toLowerCase())

        const matchesFolder = !selectedFolder || n.folder === selectedFolder

        return matchesSearch && matchesFolder
      })
      .sort((a, b) => {
        // Pinned first
        if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
        // Then by updated date
        return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
      })
  }, [notes, search, selectedFolder])

  const handleCreateNote = () => {
    const id = addNote({
      title: 'Untitled Note',
      content: '',
      tags: [],
    })
    selectNote(id)
  }

  const handleSave = () => {
    if (selectedNoteId && (editTitle !== selectedNote?.title || editContent !== selectedNote?.content)) {
      updateNote(selectedNoteId, {
        title: editTitle,
        content: editContent,
      })
    }
  }

  // Auto-save on blur or after delay
  useEffect(() => {
    const timer = setTimeout(handleSave, 1000)
    return () => clearTimeout(timer)
  }, [editTitle, editContent])

  return (
    <PageLayout
      title="Notes"
      description={`${notes.length} notes`}
      actions={
        <Button onClick={handleCreateNote}>
          <Plus className="h-4 w-4 mr-2" />
          New Note
        </Button>
      }
      noPadding
    >
      <div className="flex h-full">
        {/* Sidebar */}
        <div className="w-72 border-r flex flex-col">
          {/* Search */}
          <div className="p-3 border-b">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search notes..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
          </div>

          {/* Folders */}
          <div className="p-3 space-y-1">
            <button
              onClick={() => setSelectedFolder(null)}
              className={cn(
                'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                !selectedFolder
                  ? 'bg-accent text-accent-foreground'
                  : 'text-muted-foreground hover:bg-accent/50'
              )}
            >
              <FileText className="h-4 w-4" />
              All Notes
              <span className="ml-auto text-xs">{notes.length}</span>
            </button>

            {folders.map((folder) => {
              const count = notes.filter((n) => n.folder === folder).length
              return (
                <button
                  key={folder}
                  onClick={() => setSelectedFolder(folder)}
                  className={cn(
                    'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                    selectedFolder === folder
                      ? 'bg-accent text-accent-foreground'
                      : 'text-muted-foreground hover:bg-accent/50'
                  )}
                >
                  <Folder className="h-4 w-4" />
                  {folder}
                  <span className="ml-auto text-xs">{count}</span>
                </button>
              )
            })}
          </div>

          {/* Notes list */}
          <ScrollArea className="flex-1 px-3 pb-3">
            <div className="space-y-1">
              {filteredNotes.map((note) => (
                <NoteItem
                  key={note.id}
                  note={note}
                  isSelected={selectedNoteId === note.id}
                  onClick={() => selectNote(note.id)}
                  onTogglePin={() => togglePin(note.id)}
                  onDelete={() => deleteNote(note.id)}
                />
              ))}
            </div>
          </ScrollArea>
        </div>

        {/* Editor */}
        <div className="flex-1 flex flex-col">
          {selectedNote ? (
            <>
              <div className="p-4 border-b">
                <Input
                  value={editTitle}
                  onChange={(e) => setEditTitle(e.target.value)}
                  className="text-xl font-semibold border-0 p-0 h-auto focus-visible:ring-0"
                  placeholder="Note title"
                />
                <div className="flex items-center gap-2 mt-2 text-xs text-muted-foreground">
                  <span>Last edited {formatRelativeTime(selectedNote.updatedAt)}</span>
                  {selectedNote.folder && (
                    <Badge variant="secondary" className="text-xs">
                      {selectedNote.folder}
                    </Badge>
                  )}
                </div>
              </div>

              <div className="flex-1 p-4">
                <textarea
                  value={editContent}
                  onChange={(e) => setEditContent(e.target.value)}
                  className="w-full h-full resize-none bg-transparent border-0 focus:outline-none font-mono text-sm"
                  placeholder="Start writing..."
                />
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-muted-foreground">
              <div className="text-center">
                <FileText className="h-16 w-16 mx-auto mb-4 opacity-50" />
                <p>Select a note or create a new one</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </PageLayout>
  )
}
