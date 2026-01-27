import { PageLayout } from '@/layouts'
import { EmptyState, FilterSidebar, SearchInput } from '@/shared/components'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import { ScrollArea } from '@/ui/scroll-area'
import { ArrowLeft, FileText, Folder, Plus } from 'lucide-react'
import { useCallback, useState } from 'react'
import { NoteEditor, NoteListItem } from '../components'
import { useNotesPage } from '../hooks/use-notes-page'

export function NotesPage() {
  const {
    notes,
    folderItems,
    totalCount,
    selectedNote,
    search,
    setSearch,
    selectedFolder,
    setSelectedFolder,
    editTitle,
    setEditTitle,
    editContent,
    setEditContent,
    selectNote,
    createNote,
    deleteNote,
    togglePin,
  } = useNotesPage()

  // Mobile view state: 'list' or 'editor'
  const [mobileView, setMobileView] = useState<'list' | 'editor'>('list')

  // When selecting a note on mobile, switch to editor view
  const handleSelectNote = useCallback(
    (noteId: string) => {
      selectNote(noteId)
      setMobileView('editor')
    },
    [selectNote]
  )

  // Go back to list view on mobile
  const handleBackToList = useCallback(() => {
    setMobileView('list')
  }, [])

  // Handle create note - switch to editor view on mobile
  const handleCreateNote = useCallback(() => {
    createNote()
    setMobileView('editor')
  }, [createNote])

  return (
    <PageLayout
      title="Notes"
      description={`${totalCount} notes`}
      actions={
        <Button onClick={handleCreateNote} size="sm" className="md:size-default">
          <Plus className="h-4 w-4 md:mr-2" />
          <span className="hidden md:inline">New Note</span>
        </Button>
      }
      noPadding
    >
      <div className="flex h-full">
        {/* Sidebar / List (hidden on mobile when viewing editor) */}
        <div className={cn('w-full md:w-80 border-r flex flex-col', mobileView === 'editor' && 'hidden md:flex')}>
          {/* Search */}
          <div className="p-3 border-b">
            <SearchInput value={search} onChange={setSearch} placeholder="Search notes..." />
          </div>

          {/* Folders */}
          <div className="p-3">
            <FilterSidebar
              selected={selectedFolder}
              onSelect={setSelectedFolder}
              allItem={{
                label: 'All Notes',
                icon: FileText,
                count: totalCount,
              }}
              sections={[
                {
                  items: folderItems.map((f) => ({
                    ...f,
                    icon: Folder,
                  })),
                },
              ]}
            />
          </div>

          {/* Notes list */}
          <ScrollArea className="flex-1 px-3 pb-3">
            <div className="space-y-1">
              {notes.map((note) => (
                <NoteListItem
                  key={note.id}
                  note={note}
                  isSelected={selectedNote?.id === note.id}
                  onSelect={handleSelectNote}
                  onTogglePin={togglePin}
                  onDelete={deleteNote}
                />
              ))}
            </div>
          </ScrollArea>
        </div>

        {/* Editor (hidden on mobile when viewing list) */}
        <div className={cn('flex-1 flex flex-col', mobileView === 'list' && 'hidden md:flex')}>
          {selectedNote ? (
            <>
              {/* Mobile back button */}
              <div className="md:hidden border-b px-3 py-2">
                <Button variant="ghost" size="sm" onClick={handleBackToList} className="gap-2">
                  <ArrowLeft className="h-4 w-4" />
                  Back to notes
                </Button>
              </div>
              <NoteEditor
                note={selectedNote}
                title={editTitle}
                content={editContent}
                onTitleChange={setEditTitle}
                onContentChange={setEditContent}
              />
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center">
              <EmptyState icon={FileText} title="No note selected" description="Select a note or create a new one" />
            </div>
          )}
        </div>
      </div>
    </PageLayout>
  )
}
