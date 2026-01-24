import { Button } from '@/ui/button'
import { PageLayout } from '@/layouts'
import { SearchInput, FilterSidebar, EmptyState } from '@/shared/components'
import { useBookmarksPage } from '../hooks/use-bookmarks-page'
import { BookmarkCard, BookmarkDialog } from '../components'
import { Plus, Folder, Globe } from 'lucide-react'

export function BookmarksPage() {
  const {
    bookmarks,
    folders,
    folderItems,
    totalCount,
    search,
    setSearch,
    selectedFolder,
    setSelectedFolder,
    dialogOpen,
    editingBookmark,
    openDialog,
    closeDialog,
    handleSubmit,
    deleteBookmark,
  } = useBookmarksPage()

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
            count: totalCount,
          }}
          sections={[
            {
              title: 'Folders',
              items: folderItems.map((f) => ({
                ...f,
                icon: Folder,
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
          {bookmarks.length > 0 ? (
            <div className="grid gap-3 md:grid-cols-2">
              {bookmarks.map((bookmark) => (
                <BookmarkCard
                  key={bookmark.id}
                  bookmark={bookmark}
                  onEdit={() => openDialog(bookmark)}
                  onDelete={() => deleteBookmark(bookmark.id)}
                />
              ))}
            </div>
          ) : (
            <EmptyState
              icon={Globe}
              title="No bookmarks found"
              description={
                search
                  ? 'Try a different search term'
                  : 'Add your first bookmark to get started'
              }
            />
          )}
        </div>
      </div>

      {/* Add/Edit Dialog */}
      <BookmarkDialog
        open={dialogOpen}
        onOpenChange={(open) => !open && closeDialog()}
        bookmark={editingBookmark}
        folders={folders}
        onSubmit={handleSubmit}
      />
    </PageLayout>
  )
}
