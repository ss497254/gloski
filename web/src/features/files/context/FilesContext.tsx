/* eslint-disable react-refresh/only-export-components */
import { useServer } from '@/features/servers'
import type { FileEntry, ListResponse } from '@/shared/lib/types'
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from 'react'
import { useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { getFileType } from '../lib/file-types'
import { getParentPath, joinPath } from '../lib/file-utils'
import { useFilesStore } from '../stores/files'

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
  loadFileContent: (entry: FileEntry) => void

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

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

interface FilesProviderProps {
  children: ReactNode
}

export function FilesProvider({ children }: FilesProviderProps) {
  const { server } = useServer()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryPath = searchParams.get('path') || '~'

  // ─────────────────────────────────────────────────────────────────────────────
  // Files store - use selectors for optimized subscriptions
  // ─────────────────────────────────────────────────────────────────────────────

  const setPinnedFolders = useFilesStore((s) => s.setPinnedFolders)
  const addPinnedFolder = useFilesStore((s) => s.addPinnedFolder)
  const removePinnedFolder = useFilesStore((s) => s.removePinnedFolder)
  const isPinned = useFilesStore((s) => s.isPinned)
  const getPinnedFolderByPath = useFilesStore((s) => s.getPinnedFolderByPath)
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

  const [data, setData] = useState<ListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [homeDir, setHomeDir] = useState<string | undefined>()

  // Use the actual path from server response for file operations
  // This is the resolved absolute path, not the URL query param
  const currentPath = data?.path || queryPath

  // ─────────────────────────────────────────────────────────────────────────────
  // File preview state
  // ─────────────────────────────────────────────────────────────────────────────

  const [selectedFile, setSelectedFile] = useState<FileEntry | null>(null)
  const [fileContent, setFileContent] = useState<string | null>(null)
  const [fileLoading, setFileLoading] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [editedContent, setEditedContent] = useState('')
  const [saving, setSaving] = useState(false)

  // ─────────────────────────────────────────────────────────────────────────────
  // UI state
  // ─────────────────────────────────────────────────────────────────────────────

  const [searchQuery, setSearchQuery] = useState('')
  const [focusedIndex, setFocusedIndex] = useState(0)

  // ─────────────────────────────────────────────────────────────────────────────
  // Dialog state
  // ─────────────────────────────────────────────────────────────────────────────

  const [newFolderDialog, setNewFolderDialog] = useState(false)
  const [newFileDialog, setNewFileDialog] = useState(false)
  const [deleteDialog, setDeleteDialog] = useState<FileEntry | null>(null)
  const [renameDialog, setRenameDialog] = useState<FileEntry | null>(null)
  const [bulkDeleteDialog, setBulkDeleteDialog] = useState(false)
  const [newName, setNewName] = useState('')

  // ─────────────────────────────────────────────────────────────────────────────
  // Upload state
  // ─────────────────────────────────────────────────────────────────────────────

  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // ─────────────────────────────────────────────────────────────────────────────
  // Computed values
  // ─────────────────────────────────────────────────────────────────────────────

  const sortedAndFilteredEntries = useMemo(() => {
    if (!data?.entries) return []

    let entries = [...data.entries]

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
  }, [data?.entries, searchQuery, sortBy, sortOrder])

  const hasUnsavedChanges = isEditing && editedContent !== fileContent
  const isDialogOpen = !!newFolderDialog || !!newFileDialog || !!deleteDialog || !!renameDialog || bulkDeleteDialog

  // ─────────────────────────────────────────────────────────────────────────────
  // Effects
  // ─────────────────────────────────────────────────────────────────────────────

  // Clear selection when path changes
  useEffect(() => {
    clearSelection()
    setFocusedIndex(0)
  }, [queryPath, clearSelection])

  // Load pinned folders and home directory from server
  useEffect(() => {
    server
      .getClient()
      .files.pinned.list()
      .then((response) => {
        setPinnedFolders(response.folders || [])
        setHomeDir(response.home_dir)
      })
      .catch((err) => {
        console.error('Failed to load pinned folders:', err)
      })
  }, [server, setPinnedFolders])

  // ─────────────────────────────────────────────────────────────────────────────
  // Data Loading
  // ─────────────────────────────────────────────────────────────────────────────

  const loadDirectory = useCallback(
    async (path: string) => {
      setLoading(true)
      setError('')
      try {
        const result = await server.getClient().files.list(path)
        setData(result)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load directory')
      } finally {
        setLoading(false)
      }
    },
    [server]
  )

  useEffect(() => {
    loadDirectory(queryPath)
  }, [queryPath, loadDirectory])

  // ─────────────────────────────────────────────────────────────────────────────
  // Navigation
  // ─────────────────────────────────────────────────────────────────────────────

  const navigateTo = useCallback(
    (path: string) => {
      setSearchParams({ path })
      setSelectedFile(null)
      setFileContent(null)
      setIsEditing(false)
    },
    [setSearchParams]
  )

  const goUp = useCallback(() => {
    navigateTo(getParentPath(currentPath))
  }, [currentPath, navigateTo])

  const refresh = useCallback(() => {
    loadDirectory(queryPath)
  }, [loadDirectory, queryPath])

  // ─────────────────────────────────────────────────────────────────────────────
  // File Operations
  // ─────────────────────────────────────────────────────────────────────────────

  const loadFileContent = useCallback(
    async (entry: FileEntry) => {
      setFileLoading(true)
      try {
        const result = await server.getClient().files.read(entry.path)
        setFileContent(result.content)
        setEditedContent(result.content)
      } catch (err) {
        setFileContent(`Error loading file: ${err instanceof Error ? err.message : 'Unknown error'}`)
      } finally {
        setFileLoading(false)
      }
    },
    [server]
  )

  const handleEntrySelect = useCallback(
    async (entry: FileEntry) => {
      if (entry.type === 'directory') {
        navigateTo(entry.path)
      } else {
        setSelectedFile(entry)
        setIsEditing(false)

        // Only load content for text files
        const fileType = getFileType(entry.name)
        if (fileType === 'code' || fileType === 'text') {
          loadFileContent(entry)
        } else {
          setFileContent(null)
        }
      }
    },
    [navigateTo, loadFileContent]
  )

  const handleEdit = useCallback(
    (entry: FileEntry) => {
      handleEntrySelect(entry)
      setTimeout(() => setIsEditing(true), 100)
    },
    [handleEntrySelect]
  )

  const handleSave = useCallback(async () => {
    if (!selectedFile) return
    setSaving(true)
    try {
      await server.getClient().files.write(selectedFile.path, editedContent)
      setFileContent(editedContent)
      setIsEditing(false)
      toast.success('File saved successfully')
    } catch {
      toast.error('Failed to save file')
    } finally {
      setSaving(false)
    }
  }, [server, selectedFile, editedContent])

  const handleCreateFolder = useCallback(async () => {
    if (!newName.trim()) return
    try {
      await server.getClient().files.mkdir(joinPath(currentPath, newName))
      toast.success('Folder created')
      setNewFolderDialog(false)
      setNewName('')
      loadDirectory(currentPath)
    } catch {
      toast.error('Failed to create folder')
    }
  }, [server, newName, currentPath, loadDirectory])

  const handleCreateFile = useCallback(async () => {
    if (!newName.trim()) return
    try {
      await server.getClient().files.write(joinPath(currentPath, newName), '')
      toast.success('File created')
      setNewFileDialog(false)
      setNewName('')
      loadDirectory(currentPath)
    } catch {
      toast.error('Failed to create file')
    }
  }, [server, newName, currentPath, loadDirectory])

  const handleDelete = useCallback(async () => {
    if (!deleteDialog) return
    try {
      await server.getClient().files.delete(deleteDialog.path)
      toast.success(`${deleteDialog.type === 'directory' ? 'Folder' : 'File'} deleted`)
      setDeleteDialog(null)
      if (selectedFile?.path === deleteDialog.path) {
        setSelectedFile(null)
        setFileContent(null)
      }
      loadDirectory(currentPath)
    } catch {
      toast.error('Failed to delete')
    }
  }, [server, deleteDialog, selectedFile?.path, currentPath, loadDirectory])

  const handleRename = useCallback(async () => {
    if (!renameDialog || !newName.trim()) return
    const newPath = joinPath(getParentPath(renameDialog.path), newName)
    try {
      await server.getClient().files.rename(renameDialog.path, newPath)
      toast.success('Renamed successfully')
      setRenameDialog(null)
      setNewName('')
      if (selectedFile?.path === renameDialog.path) {
        setSelectedFile(null)
        setFileContent(null)
      }
      loadDirectory(currentPath)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to rename')
    }
  }, [server, renameDialog, newName, selectedFile?.path, currentPath, loadDirectory])

  const handleBulkDelete = useCallback(async () => {
    if (selectedPaths.size === 0) return

    let successCount = 0
    let failCount = 0

    for (const path of selectedPaths) {
      try {
        await server.getClient().files.delete(path)
        successCount++
      } catch {
        failCount++
      }
    }

    setBulkDeleteDialog(false)
    clearSelection()

    if (successCount > 0) {
      toast.success(`Deleted ${successCount} item(s)`)
      loadDirectory(currentPath)
    }
    if (failCount > 0) {
      toast.error(`Failed to delete ${failCount} item(s)`)
    }
  }, [server, selectedPaths, currentPath, loadDirectory, clearSelection])

  const handleDropUpload = useCallback(
    async (files: File[]) => {
      if (files.length === 0) return

      setUploading(true)
      let successCount = 0
      let failCount = 0

      for (const file of files) {
        try {
          await server.getClient().files.upload(currentPath, file)
          successCount++
        } catch (err) {
          failCount++
          console.error(`Failed to upload ${file.name}:`, err)
        }
      }

      setUploading(false)

      if (successCount > 0) {
        toast.success(`Uploaded ${successCount} file(s)`)
        loadDirectory(currentPath)
      }
      if (failCount > 0) {
        toast.error(`Failed to upload ${failCount} file(s)`)
      }
    },
    [server, currentPath, loadDirectory]
  )

  const handleUpload = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files
      if (!files || files.length === 0) return
      handleDropUpload(Array.from(files))
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    },
    [handleDropUpload]
  )

  const handleDownload = useCallback(
    (entry: FileEntry) => {
      const url = server.getClient().files.getDownloadUrl(entry.path)
      const link = document.createElement('a')
      link.href = url
      link.download = entry.name
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
    },
    [server]
  )

  const handleDownloadSelected = useCallback(() => {
    if (!data?.entries) return
    const selectedEntries = data.entries.filter((e: FileEntry) => selectedPaths.has(e.path) && e.type === 'file')
    selectedEntries.forEach((entry: FileEntry) => handleDownload(entry))
  }, [data?.entries, selectedPaths, handleDownload])

  const copyPath = useCallback((path: string) => {
    navigator.clipboard.writeText(path)
    toast.success('Path copied to clipboard')
  }, [])

  const closePreview = useCallback(() => {
    if (isEditing && editedContent !== fileContent) {
      if (!window.confirm('You have unsaved changes. Are you sure you want to close?')) {
        return
      }
    }
    setSelectedFile(null)
    setFileContent(null)
    setIsEditing(false)
  }, [isEditing, editedContent, fileContent])

  const cancelEdit = useCallback(() => {
    setIsEditing(false)
    setEditedContent(fileContent || '')
  }, [fileContent])

  const startEdit = useCallback(() => {
    setIsEditing(true)
  }, [])

  // ─────────────────────────────────────────────────────────────────────────────
  // Pin/Unpin Folder
  // ─────────────────────────────────────────────────────────────────────────────

  const handleTogglePinEntry = useCallback(
    async (entry: FileEntry) => {
      if (entry.type !== 'directory') return

      if (isPinned(entry.path)) {
        const folder = getPinnedFolderByPath(entry.path)
        if (folder) {
          try {
            await server.getClient().files.pinned.unpin(folder.id)
            removePinnedFolder(folder.id)
            toast.success('Folder unpinned')
          } catch {
            toast.error('Failed to unpin folder')
          }
        }
      } else {
        try {
          const folder = await server.getClient().files.pinned.pin(entry.path, entry.name)
          addPinnedFolder(folder)
          toast.success('Folder pinned')
        } catch {
          toast.error('Failed to pin folder')
        }
      }
    },
    [server, isPinned, getPinnedFolderByPath, removePinnedFolder, addPinnedFolder]
  )

  const handleSidebarUnpin = useCallback(
    async (id: string) => {
      try {
        await server.getClient().files.pinned.unpin(id)
        removePinnedFolder(id)
        toast.success('Folder unpinned')
      } catch {
        toast.error('Failed to unpin folder')
      }
    },
    [server, removePinnedFolder]
  )

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

  const handleRenameEntry = useCallback((entry: FileEntry) => {
    setRenameDialog(entry)
    setNewName(entry.name)
  }, [])

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
      entries: data?.entries || [],
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
      loading,
      error,
      homeDir,
      sortedAndFilteredEntries,
      selectedFile,
      fileContent,
      isEditing,
      editedContent,
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
      handleDropUpload,
      handleUpload,
      newFolderDialog,
      newFileDialog,
      deleteDialog,
      renameDialog,
      bulkDeleteDialog,
      newName,
      searchQuery,
      focusedIndex,
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
