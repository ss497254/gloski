import { useCallback, useMemo } from 'react'
import { useDialog, useFilter } from '@/shared/hooks'
import { type Bookmark, useBookmarksStore } from '../stores/bookmarks'

export function useBookmarksPage() {
  const bookmarks = useBookmarksStore((s) => s.bookmarks)
  const folders = useBookmarksStore((s) => s.folders)
  const addBookmark = useBookmarksStore((s) => s.addBookmark)
  const updateBookmark = useBookmarksStore((s) => s.updateBookmark)
  const deleteBookmark = useBookmarksStore((s) => s.deleteBookmark)

  // Filter and search logic
  const {
    search,
    setSearch,
    filter: selectedFolder,
    setFilter: setSelectedFolder,
    filteredItems: filteredBookmarks,
  } = useFilter<Bookmark>({
    items: bookmarks,
    searchFields: ['title', 'description', 'tags'],
    filterFn: (item, folder) => !folder || item.folder === folder,
  })

  // Dialog logic
  const dialog = useDialog<Bookmark>({
    onSubmit: (data, isEditing) => {
      if (isEditing) {
        updateBookmark(data.id, data)
      } else {
        addBookmark(data)
      }
    },
  })

  // Folder items for sidebar
  const folderItems = useMemo(
    () =>
      folders.map((folder) => ({
        id: folder,
        label: folder,
        count: bookmarks.filter((b) => b.folder === folder).length,
      })),
    [folders, bookmarks]
  )

  // Handle bookmark submission from form
  const handleSubmit = useCallback(
    (formData: { title: string; url: string; description?: string; folder?: string; tags: string[] }) => {
      if (dialog.isEditing && dialog.editingItem) {
        updateBookmark(dialog.editingItem.id, formData)
      } else {
        addBookmark(formData)
      }
      dialog.close()
    },
    [dialog, updateBookmark, addBookmark]
  )

  return {
    // Data
    bookmarks: filteredBookmarks,
    folders,
    folderItems,
    totalCount: bookmarks.length,

    // Search/Filter state
    search,
    setSearch,
    selectedFolder,
    setSelectedFolder,

    // Dialog state
    dialogOpen: dialog.isOpen,
    editingBookmark: dialog.editingItem,
    isEditing: dialog.isEditing,
    openDialog: dialog.open,
    closeDialog: dialog.close,

    // Actions
    handleSubmit,
    deleteBookmark,
  }
}
