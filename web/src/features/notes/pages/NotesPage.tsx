import { Button } from '@/ui/button'
import { ScrollArea } from '@/ui/scroll-area'
import { PageLayout } from '@/layouts'
import { SearchInput, FilterSidebar, EmptyState } from '@/shared/components'
import { useNotesPage } from '../hooks/use-notes-page'
import { NoteListItem, NoteEditor } from '../components'
import { Plus, FileText, Folder } from 'lucide-react'

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

  return (
    <PageLayout
      title="Notes"
      description={`${totalCount} notes`}
      actions={
        <Button onClick={createNote}>
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
            <SearchInput
              value={search}
              onChange={setSearch}
              placeholder="Search notes..."
            />
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
                  onSelect={() => selectNote(note.id)}
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
            <NoteEditor
              note={selectedNote}
              title={editTitle}
              content={editContent}
              onTitleChange={setEditTitle}
              onContentChange={setEditContent}
            />
          ) : (
            <div className="flex-1 flex items-center justify-center">
              <EmptyState
                icon={FileText}
                title="No note selected"
                description="Select a note or create a new one"
              />
            </div>
          )}
        </div>
      </div>
    </PageLayout>
  )
}
