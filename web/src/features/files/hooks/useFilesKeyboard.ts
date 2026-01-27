import type { FileEntry } from '@/shared/lib/types'
import { useCallback, useEffect } from 'react'

interface UseFilesKeyboardOptions {
  entries: FileEntry[]
  focusedIndex: number
  selectedPaths: Set<string>
  onFocusChange: (index: number) => void
  onOpen: (entry: FileEntry) => void
  onNavigateUp: () => void
  onDelete: (entries: FileEntry[]) => void
  onRename: (entry: FileEntry) => void
  onToggleSelection: (path: string) => void
  onSelectAll: () => void
  onClearSelection: () => void
  onSelectRange: (path: string) => void
  onCopyPath: (entry: FileEntry) => void
  onClosePreview: () => void
  disabled?: boolean
}

export function useFilesKeyboard({
  entries,
  focusedIndex,
  selectedPaths,
  onFocusChange,
  onOpen,
  onNavigateUp,
  onDelete,
  onRename,
  onToggleSelection,
  onSelectAll,
  onClearSelection,
  onSelectRange,
  onCopyPath,
  onClosePreview,
  disabled = false,
}: UseFilesKeyboardOptions) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Don't handle if typing in an input or textarea
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
        return
      }

      if (disabled || entries.length === 0) return

      const focusedEntry = entries[focusedIndex]

      switch (e.key) {
        case 'ArrowUp':
          e.preventDefault()
          if (focusedIndex > 0) {
            if (e.shiftKey && focusedEntry) {
              onSelectRange(entries[focusedIndex - 1].path)
            }
            onFocusChange(focusedIndex - 1)
          }
          break

        case 'ArrowDown':
          e.preventDefault()
          if (focusedIndex < entries.length - 1) {
            if (e.shiftKey && focusedEntry) {
              onSelectRange(entries[focusedIndex + 1].path)
            }
            onFocusChange(focusedIndex + 1)
          }
          break

        case 'Enter':
          e.preventDefault()
          if (focusedEntry) {
            onOpen(focusedEntry)
          }
          break

        case 'Backspace':
          e.preventDefault()
          onNavigateUp()
          break

        case 'Delete':
          e.preventDefault()
          if (selectedPaths.size > 0) {
            const selectedEntries = entries.filter((e) => selectedPaths.has(e.path))
            onDelete(selectedEntries)
          } else if (focusedEntry) {
            onDelete([focusedEntry])
          }
          break

        case 'F2':
          e.preventDefault()
          if (focusedEntry) {
            onRename(focusedEntry)
          }
          break

        case ' ':
          e.preventDefault()
          if (focusedEntry) {
            onToggleSelection(focusedEntry.path)
          }
          break

        case 'a':
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault()
            onSelectAll()
          }
          break

        case 'c':
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault()
            if (focusedEntry) {
              onCopyPath(focusedEntry)
            }
          }
          break

        case 'Escape':
          e.preventDefault()
          if (selectedPaths.size > 0) {
            onClearSelection()
          } else {
            onClosePreview()
          }
          break

        case 'Home':
          e.preventDefault()
          onFocusChange(0)
          break

        case 'End':
          e.preventDefault()
          onFocusChange(entries.length - 1)
          break
      }
    },
    [
      entries,
      focusedIndex,
      selectedPaths,
      onFocusChange,
      onOpen,
      onNavigateUp,
      onDelete,
      onRename,
      onToggleSelection,
      onSelectAll,
      onClearSelection,
      onSelectRange,
      onCopyPath,
      onClosePreview,
      disabled,
    ]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}
