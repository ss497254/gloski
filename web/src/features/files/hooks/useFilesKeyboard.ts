import { useCallback, useEffect } from 'react'
import { useFiles } from '../context'
import { useFilesStore } from '../stores/files'

/**
 * Keyboard navigation hook for the files feature.
 * Consumes FilesContext for all state and actions.
 */
export function useFilesKeyboard() {
  const {
    sortedAndFilteredEntries: entries,
    focusedIndex,
    setFocusedIndex,
    handleEntrySelect,
    goUp,
    setDeleteDialog,
    setBulkDeleteDialog,
    handleRenameEntry,
    handleSelectAll,
    copyPath,
    closePreview,
    isDialogOpen,
  } = useFiles()

  // Use selectors for store values
  const selectedPaths = useFilesStore((s) => s.selectedPaths)
  const toggleSelection = useFilesStore((s) => s.toggleSelection)
  const clearSelection = useFilesStore((s) => s.clearSelection)
  const selectRange = useFilesStore((s) => s.selectRange)

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Don't handle if typing in an input or textarea
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
        return
      }

      if (isDialogOpen || entries.length === 0) return

      const focusedEntry = entries[focusedIndex]

      switch (e.key) {
        case 'ArrowUp':
          e.preventDefault()
          if (focusedIndex > 0) {
            if (e.shiftKey && focusedEntry) {
              selectRange(
                entries.map((entry) => entry.path),
                entries[focusedIndex - 1].path
              )
            }
            setFocusedIndex(focusedIndex - 1)
          }
          break

        case 'ArrowDown':
          e.preventDefault()
          if (focusedIndex < entries.length - 1) {
            if (e.shiftKey && focusedEntry) {
              selectRange(
                entries.map((entry) => entry.path),
                entries[focusedIndex + 1].path
              )
            }
            setFocusedIndex(focusedIndex + 1)
          }
          break

        case 'Enter':
          e.preventDefault()
          if (focusedEntry) {
            handleEntrySelect(focusedEntry)
          }
          break

        case 'Backspace':
          e.preventDefault()
          goUp()
          break

        case 'Delete':
          e.preventDefault()
          if (selectedPaths.size > 1) {
            setBulkDeleteDialog(true)
          } else if (selectedPaths.size === 1) {
            const selectedEntry = entries.find((entry) => selectedPaths.has(entry.path))
            if (selectedEntry) {
              setDeleteDialog(selectedEntry)
            }
          } else if (focusedEntry) {
            setDeleteDialog(focusedEntry)
          }
          break

        case 'F2':
          e.preventDefault()
          if (focusedEntry) {
            handleRenameEntry(focusedEntry)
          }
          break

        case ' ':
          e.preventDefault()
          if (focusedEntry) {
            toggleSelection(focusedEntry.path)
          }
          break

        case 'a':
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault()
            handleSelectAll()
          }
          break

        case 'c':
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault()
            if (focusedEntry) {
              copyPath(focusedEntry.path)
            }
          }
          break

        case 'Escape':
          e.preventDefault()
          if (selectedPaths.size > 0) {
            clearSelection()
          } else {
            closePreview()
          }
          break

        case 'Home':
          e.preventDefault()
          setFocusedIndex(0)
          break

        case 'End':
          e.preventDefault()
          setFocusedIndex(entries.length - 1)
          break
      }
    },
    [
      entries,
      focusedIndex,
      setFocusedIndex,
      handleEntrySelect,
      goUp,
      setDeleteDialog,
      setBulkDeleteDialog,
      handleRenameEntry,
      handleSelectAll,
      copyPath,
      closePreview,
      isDialogOpen,
      selectedPaths,
      toggleSelection,
      clearSelection,
      selectRange,
    ]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}
