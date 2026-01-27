import { useFilter } from '@/shared/hooks'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNotesStore, type Note } from '../stores/notes'

export function useNotesPage() {
  const { notes, folders, selectedNoteId, addNote, updateNote, deleteNote, togglePin, selectNote } = useNotesStore()

  // Edit state for the selected note
  const [editTitle, setEditTitle] = useState('')
  const [editContent, setEditContent] = useState('')

  // Filter and search
  const {
    search,
    setSearch,
    filter: selectedFolder,
    setFilter: setSelectedFolder,
    filteredItems,
  } = useFilter<Note>({
    items: notes,
    searchFields: ['title', 'content'],
    filterFn: (item, folder) => !folder || item.folder === folder,
    sortFn: (a, b) => {
      // Pinned first, then by updated date
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
      return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
    },
  })

  const selectedNote = useMemo(() => notes.find((n) => n.id === selectedNoteId) || null, [notes, selectedNoteId])

  // Sync edit state when selected note changes
  useEffect(() => {
    if (selectedNote) {
      setEditTitle(selectedNote.title)
      setEditContent(selectedNote.content)
    }
  }, [selectedNoteId]) // Only run when selection changes, not on every note update

  // Create a new note
  const handleCreateNote = useCallback(() => {
    addNote({
      title: 'Untitled Note',
      content: '',
      tags: [],
    })
  }, [addNote])

  // Save the current note
  const handleSave = useCallback(() => {
    if (selectedNoteId && selectedNote && (editTitle !== selectedNote.title || editContent !== selectedNote.content)) {
      updateNote(selectedNoteId, {
        title: editTitle,
        content: editContent,
      })
    }
  }, [selectedNoteId, selectedNote, editTitle, editContent, updateNote])

  // Auto-save with debounce
  useEffect(() => {
    const timer = setTimeout(handleSave, 1000)
    return () => clearTimeout(timer)
  }, [editTitle, editContent, handleSave])

  // Folder items for sidebar
  const folderItems = useMemo(
    () =>
      folders.map((folder) => ({
        id: folder,
        label: folder,
        count: notes.filter((n) => n.folder === folder).length,
      })),
    [folders, notes]
  )

  return {
    // Data
    notes: filteredItems,
    folders,
    folderItems,
    totalCount: notes.length,
    selectedNote,

    // Search/Filter
    search,
    setSearch,
    selectedFolder,
    setSelectedFolder,

    // Edit state
    editTitle,
    setEditTitle,
    editContent,
    setEditContent,

    // Actions
    selectNote,
    createNote: handleCreateNote,
    deleteNote,
    togglePin,
  }
}
