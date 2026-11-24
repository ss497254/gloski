/* eslint-disable react-refresh/only-export-components */
import { useDevice } from '@/shared/context'
import { unwrap, type FileEntry, type ListResponse } from '@/shared/api'
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
  type RefObject,
} from 'react'
import { useSearchParams } from 'react-router-dom'
import { getParentPath } from '../lib/file-utils'
import { useFilesStore } from '../stores/files'
import { useFileDirectory } from '../hooks/useFileDirectory'
import { useFileOperations } from '../hooks/useFileOperations'
import { useFilePreview } from '../hooks/useFilePreview'

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

interface FilesContextValue {
  // Navigation
  currentPath: string
  queryPath: string
  navigateTo: (path: string) => void
  goUp: () => void
  refresh: () => void

  // Data
  data: ListResponse | null
  entries: FileEntry[]
  loading: boolean
  error: string
  homeDir: string | undefined
  sortedAndFilteredEntries: FileEntry[]

  // Preview state
  selectedFile: FileEntry | null
  fileContent: string | null
  isEditing: boolean
  editedContent: string
  fileLoading: boolean
  saving: boolean
  hasUnsavedChanges: boolean

  // Preview actions
  handleEntrySelect: (entry: FileEntry) => void
  handleEdit: (entry: FileEntry) => void
  handleSave: () => Promise<void>
  closePreview: () => void
  cancelEdit: () => void
  startEdit: () => void
  setEditedContent: (content: string) => void
  loadFileContent: (entry: FileEntry) => Promise<void>

  // File operations
  handleCreateFolder: () => Promise<void>
  handleCreateFile: () => Promise<void>
  handleDelete: () => Promise<void>
  handleRename: () => Promise<void>
  handleBulkDelete: () => Promise<void>
  handleDownload: (entry: FileEntry) => void
  handleDownloadSelected: () => void
  copyPath: (path: string) => void
  handleTogglePinEntry: (entry: FileEntry) => Promise<void>

  // Upload
  uploading: boolean
  handleDropUpload: (files: File[]) => Promise<void>
  handleUpload: (e: React.ChangeEvent<HTMLInputElement>) => void
  fileInputRef: RefObject<HTMLInputElement | null>

  // Dialogs
  newFolderDialog: boolean
  setNewFolderDialog: (open: boolean) => void
  newFileDialog: boolean
  setNewFileDialog: (open: boolean) => void
  deleteDialog: FileEntry | null
  setDeleteDialog: (entry: FileEntry | null) => void
  renameDialog: FileEntry | null
  setRenameDialog: (entry: FileEntry | null) => void
  bulkDeleteDialog: boolean
  setBulkDeleteDialog: (open: boolean) => void
  newName: string
  setNewName: (name: string) => void

  // UI state
  searchQuery: string
  setSearchQuery: (query: string) => void
  focusedIndex: number
  setFocusedIndex: (index: number) => void

  // Selection helpers (delegated to store but with computed entries)
  handleToggleCheck: (entry: FileEntry, shiftKey: boolean) => void
  handleSelectAll: () => void
  handleRenameEntry: (entry: FileEntry) => void
  handleSidebarUnpin: (id: string) => Promise<void>

  // Store values (for components that need them)
  selectedPaths: Set<string>
  viewMode: 'list' | 'grid'
  sortBy: 'name' | 'size' | 'modified'
  sortOrder: 'asc' | 'desc'
  sidebarOpen: boolean

  // Store actions
  setViewMode: (mode: 'list' | 'grid') => void
  setSortBy: (by: 'name' | 'size' | 'modified') => void
  toggleSortOrder: () => void
  setSidebarOpen: (open: boolean) => void
  toggleSidebar: () => void
  clearSelection: () => void
  isSelected: (path: string) => boolean
  isPinned: (path: string) => boolean

  // Dialog check for keyboard shortcuts
  isDialogOpen: boolean
}

const FilesContext = createContext<FilesContextValue | null>(null)
const emptyEntries: FileEntry[] = []

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

interface FilesProviderProps {
  children: ReactNode
}

export function FilesProvider({ children }: FilesProviderProps) {
  const { device } = useDevice()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryPath = searchParams.get('path') || '~'
  const setPath = useCallback((path: string) => setSearchParams({ path }), [setSearchParams])
  const { data, loading, error, loadDirectory } = useFileDirectory(device, queryPath)
  const preview = useFilePreview(device, setPath)
  const {
    selectedFile,
    fileContent,
    fileLoading,
    isEditing,
    editedContent,
    setEditedContent,
    saving,
    hasUnsavedChanges,
    clearPreview,
    handleEntrySelect,
    handleEdit,
    handleSave,
    closePreview,
    cancelEdit,
    startEdit,
    loadFileContent,
  } = preview

  // ─────────────────────────────────────────────────────────────────────────────
  // Files store - use selectors for optimized subscriptions
  // ─────────────────────────────────────────────────────────────────────────────

  const isPinned = useFilesStore((s) => s.isPinned)
  const selectedPaths = useFilesStore((s) => s.selectedPaths)
  const toggleSelection = useFilesStore((s) => s.toggleSelection)
  const selectAll = useFilesStore((s) => s.selectAll)
  const clearSelection = useFilesStore((s) => s.clearSelection)
  const selectRange = useFilesStore((s) => s.selectRange)
  const isSelected = useFilesStore((s) => s.isSelected)
  const viewMode = useFilesStore((s) => s.viewMode)
  const sortBy = useFilesStore((s) => s.sortBy)
  const sortOrder = useFilesStore((s) => s.sortOrder)
  const setViewMode = useFilesStore((s) => s.setViewMode)
  const setSortBy = useFilesStore((s) => s.setSortBy)
  const toggleSortOrder = useFilesStore((s) => s.toggleSortOrder)
  const sidebarOpen = useFilesStore((s) => s.sidebarOpen)
  const setSidebarOpen = useFilesStore((s) => s.setSidebarOpen)
  const toggleSidebar = useFilesStore((s) => s.toggleSidebar)

  // ─────────────────────────────────────────────────────────────────────────────
  // Data state
  // ─────────────────────────────────────────────────────────────────────────────

  const [homeDir, setHomeDir] = useState<string | undefined>()

  // Use the actual path from device response for file operations
  // This is the resolved absolute path, not the URL query param
  const currentPath = data?.path || queryPath
  const directoryEntries = data?.entries ?? emptyEntries
  const operations = useFileOperations({
    device,
    currentPath,
    entries: directoryEntries,
    selectedFile,
    clearPreview,
    loadDirectory,
  })
  const {
    newFolderDialog,
    setNewFolderDialog,
    newFileDialog,
    setNewFileDialog,
    deleteDialog,
    setDeleteDialog,
    renameDialog,
    setRenameDialog,
    bulkDeleteDialog,
    setBulkDeleteDialog,
    newName,
    setNewName,
    uploading,
    fileInputRef,
    handleCreateFolder,
    handleCreateFile,
    handleDelete,
    handleRename,
    handleBulkDelete,
    handleDropUpload,
    handleUpload,
    handleDownload,
    handleDownloadSelected,
    copyPath,
    handleTogglePinEntry,
    handleSidebarUnpin,
    handleRenameEntry,
    isDialogOpen,
    setPinnedFolders,
  } = operations

  // ─────────────────────────────────────────────────────────────────────────────
  // File preview state
  // ─────────────────────────────────────────────────────────────────────────────

  // ─────────────────────────────────────────────────────────────────────────────
  // UI state
  // ─────────────────────────────────────────────────────────────────────────────

  const [searchQuery, setSearchQuery] = useState('')
  const [focusState, setFocusState] = useState({ path: queryPath, index: 0 })
  const focusedIndex = focusState.path === queryPath ? focusState.index : 0
  const setFocusedIndex = useCallback((index: number) => setFocusState({ path: queryPath, index }), [queryPath])

  // ─────────────────────────────────────────────────────────────────────────────
  // ─────────────────────────────────────────────────────────────────────────────
  // Computed values
  // ─────────────────────────────────────────────────────────────────────────────

  const sortedAndFilteredEntries = useMemo(() => {

    let entries = [...directoryEntries]

    // Filter
    if (searchQuery) {
      entries = entries.filter((e) => e.name.toLowerCase().includes(searchQuery.toLowerCase()))
    }

    // Sort
    entries.sort((a, b) => {
      // Directories first
      if (a.type !== b.type) {
        return a.type === 'directory' ? -1 : 1
      }

      let comparison = 0
      switch (sortBy) {
        case 'name':
          comparison = a.name.localeCompare(b.name)
          break
        case 'size':
          comparison = a.size - b.size
          break
        case 'modified':
          comparison = new Date(a.modified).getTime() - new Date(b.modified).getTime()
          break
      }

      return sortOrder === 'asc' ? comparison : -comparison
    })

    return entries
  }, [directoryEntries, searchQuery, sortBy, sortOrder])

  // ─────────────────────────────────────────────────────────────────────────────
  // Effects
  // ─────────────────────────────────────────────────────────────────────────────

  // Clear selection when path changes
  useEffect(() => {
    clearSelection()
  }, [queryPath, clearSelection])

  // Load pinned folders and home directory from device
  useEffect(() => {
    device
      .getClient()
      .files.pinned.list()
      .then((result) => {
        const response = unwrap(result)
        setPinnedFolders(response.folders || [])
        setHomeDir(response.home_dir)
      })
      .catch((err) => {
        console.error('Failed to load pinned folders:', err)
      })
  }, [device, setPinnedFolders])

  // ─────────────────────────────────────────────────────────────────────────────
  // Navigation
  // ─────────────────────────────────────────────────────────────────────────────

  const navigateTo = useCallback(
    (path: string) => {
      clearPreview()
      setPath(path)
    },
    [clearPreview, setPath]
  )

  const goUp = useCallback(() => {
    navigateTo(getParentPath(currentPath))
  }, [currentPath, navigateTo])

  const refresh = useCallback(() => {
    loadDirectory(queryPath)
  }, [loadDirectory, queryPath])

  // ─────────────────────────────────────────────────────────────────────────────
  // Selection helpers
  // ─────────────────────────────────────────────────────────────────────────────

  const handleToggleCheck = useCallback(
    (entry: FileEntry, shiftKey: boolean) => {
      if (shiftKey) {
        selectRange(
          sortedAndFilteredEntries.map((e) => e.path),
          entry.path
        )
      } else {
        toggleSelection(entry.path)
      }
    },
    [toggleSelection, selectRange, sortedAndFilteredEntries]
  )

  const handleSelectAll = useCallback(() => {
    selectAll(sortedAndFilteredEntries.map((e) => e.path))
  }, [selectAll, sortedAndFilteredEntries])

  // ─────────────────────────────────────────────────────────────────────────────
  // Context value
  // ─────────────────────────────────────────────────────────────────────────────

  const value: FilesContextValue = useMemo(
    () => ({
      // Navigation
      currentPath,
      queryPath,
      navigateTo,
      goUp,
      refresh,

      // Data
      data,
      entries: directoryEntries,
      loading,
      error,
      homeDir,
      sortedAndFilteredEntries,

      // Preview state
      selectedFile,
      fileContent,
      isEditing,
      editedContent,
      fileLoading,
      saving,
      hasUnsavedChanges,

      // Preview actions
      handleEntrySelect,
      handleEdit,
      handleSave,
      closePreview,
      cancelEdit,
      startEdit,
      setEditedContent,
      loadFileContent,

      // File operations
      handleCreateFolder,
      handleCreateFile,
      handleDelete,
      handleRename,
      handleBulkDelete,
      handleDownload,
      handleDownloadSelected,
      copyPath,
      handleTogglePinEntry,

      // Upload
      uploading,
      handleDropUpload,
      handleUpload,
      fileInputRef,

      // Dialogs
      newFolderDialog,
      setNewFolderDialog,
      newFileDialog,
      setNewFileDialog,
      deleteDialog,
      setDeleteDialog,
      renameDialog,
      setRenameDialog,
      bulkDeleteDialog,
      setBulkDeleteDialog,
      newName,
      setNewName,

      // UI state
      searchQuery,
      setSearchQuery,
      focusedIndex,
      setFocusedIndex,

      // Selection helpers
      handleToggleCheck,
      handleSelectAll,
      handleRenameEntry,
      handleSidebarUnpin,

      // Store values
      selectedPaths,
      viewMode,
      sortBy,
      sortOrder,
      sidebarOpen,

      // Store actions
      setViewMode,
      setSortBy,
      toggleSortOrder,
      setSidebarOpen,
      toggleSidebar,
      clearSelection,
      isSelected,
      isPinned,

      // Dialog check
      isDialogOpen,
    }),
    [
      currentPath,
      queryPath,
      navigateTo,
      goUp,
      refresh,
      data,
      directoryEntries,
      loading,
      error,
      homeDir,
      sortedAndFilteredEntries,
      selectedFile,
      fileContent,
      isEditing,
      editedContent,
      setEditedContent,
      fileLoading,
      saving,
      hasUnsavedChanges,
      handleEntrySelect,
      handleEdit,
      handleSave,
      closePreview,
      cancelEdit,
      startEdit,
      loadFileContent,
      handleCreateFolder,
      handleCreateFile,
      handleDelete,
      handleRename,
      handleBulkDelete,
      handleDownload,
      handleDownloadSelected,
      copyPath,
      handleTogglePinEntry,
      uploading,
      fileInputRef,
      handleDropUpload,
      handleUpload,
      newFolderDialog,
      setNewFolderDialog,
      newFileDialog,
      setNewFileDialog,
      deleteDialog,
      setDeleteDialog,
      renameDialog,
      setRenameDialog,
      bulkDeleteDialog,
      setBulkDeleteDialog,
      newName,
      setNewName,
      searchQuery,
      focusedIndex,
      setFocusedIndex,
      handleToggleCheck,
      handleSelectAll,
      handleRenameEntry,
      handleSidebarUnpin,
      selectedPaths,
      viewMode,
      sortBy,
      sortOrder,
      sidebarOpen,
      setViewMode,
      setSortBy,
      toggleSortOrder,
      setSidebarOpen,
      toggleSidebar,
      clearSelection,
      isSelected,
      isPinned,
      isDialogOpen,
    ]
  )

  return <FilesContext.Provider value={value}>{children}</FilesContext.Provider>
}

// ─────────────────────────────────────────────────────────────────────────────
// Hook
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Hook to access the files context.
 * Must be used within a FilesProvider.
 *
 * @example
 * function FilesToolbar() {
 *   const { currentPath, navigateTo, viewMode, setViewMode } = useFiles()
 *   // ...
 * }
 */
export function useFiles(): FilesContextValue {
  const context = useContext(FilesContext)

  if (!context) {
    throw new Error('useFiles must be used within a FilesProvider. Wrap your files routes with <FilesProvider>.')
  }

  return context
}
