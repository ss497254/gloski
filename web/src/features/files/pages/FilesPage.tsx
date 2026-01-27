import { useServer } from '@/features/servers'
import { EmptyState } from '@/shared/components'
import type { FileEntry, ListResponse } from '@/shared/lib/types'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import { AlertCircle, Folder, Loader2, Search } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import {
  BulkDeleteDialog,
  DeleteDialog,
  DropZone,
  FileEntryItem,
  FilePreview,
  FilesSidebar,
  FilesToolbar,
  NewFileDialog,
  NewFolderDialog,
  RenameDialog,
} from '../components'
import { useFilesKeyboard } from '../hooks/useFilesKeyboard'
import { getFileType } from '../lib/file-types'
import { getParentPath, joinPath } from '../lib/file-utils'
import { useFilesStore } from '../stores/files'

export function FilesPage() {
  const { server } = useServer()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryPath = searchParams.get('path') || '/'

  // Files store - use selectors for optimized subscriptions
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

  // Data state
  const [data, setData] = useState<ListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [homeDir, setHomeDir] = useState<string | undefined>()

  // Use the actual path from server response for file operations
  // This is the resolved absolute path, not the URL query param
  const currentPath = data?.path || queryPath

  // File preview state
  const [selectedFile, setSelectedFile] = useState<FileEntry | null>(null)
  const [fileContent, setFileContent] = useState<string | null>(null)
  const [fileLoading, setFileLoading] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [editedContent, setEditedContent] = useState('')
  const [saving, setSaving] = useState(false)

  // UI state
  const [searchQuery, setSearchQuery] = useState('')
  const [focusedIndex, setFocusedIndex] = useState(0)

  // Dialog state
  const [newFolderDialog, setNewFolderDialog] = useState(false)
  const [newFileDialog, setNewFileDialog] = useState(false)
  const [deleteDialog, setDeleteDialog] = useState<FileEntry | null>(null)
  const [renameDialog, setRenameDialog] = useState<FileEntry | null>(null)
  const [bulkDeleteDialog, setBulkDeleteDialog] = useState(false)
  const [newName, setNewName] = useState('')

  // Upload state
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

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
    // Use actual path from server for correct parent calculation
    navigateTo(getParentPath(currentPath))
  }, [currentPath, navigateTo])

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

        // Only load content for text files, and skip large file warning handling in preview
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
    async (e: React.ChangeEvent<HTMLInputElement>) => {
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
  }, [server, data?.entries, selectedPaths, handleDownload])

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

  // Stable callback for sidebar unpin action
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

  // Stable callback for rename action in list
  const handleRenameEntry = useCallback((entry: FileEntry) => {
    setRenameDialog(entry)
    setNewName(entry.name)
  }, [])

  // Stable callback for refresh
  const handleRefresh = useCallback(() => {
    loadDirectory(queryPath)
  }, [loadDirectory, queryPath])

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

  // ─────────────────────────────────────────────────────────────────────────────
  // Computed Values (must be before callbacks that use them)
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

  // ─────────────────────────────────────────────────────────────────────────────
  // Selection
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
  // Keyboard Shortcuts
  // ─────────────────────────────────────────────────────────────────────────────

  useFilesKeyboard({
    entries: sortedAndFilteredEntries,
    focusedIndex,
    selectedPaths,
    onFocusChange: setFocusedIndex,
    onOpen: handleEntrySelect,
    onNavigateUp: goUp,
    onDelete: (entries) => {
      if (entries.length === 1) {
        setDeleteDialog(entries[0])
      } else if (entries.length > 1) {
        setBulkDeleteDialog(true)
      }
    },
    onRename: (entry) => {
      setRenameDialog(entry)
      setNewName(entry.name)
    },
    onToggleSelection: toggleSelection,
    onSelectAll: handleSelectAll,
    onClearSelection: clearSelection,
    onSelectRange: (path) =>
      selectRange(
        sortedAndFilteredEntries.map((e) => e.path),
        path
      ),
    onCopyPath: (entry) => copyPath(entry.path),
    onClosePreview: closePreview,
    disabled: !!newFolderDialog || !!newFileDialog || !!deleteDialog || !!renameDialog || bulkDeleteDialog,
  })

  // ─────────────────────────────────────────────────────────────────────────────
  // Render
  // ─────────────────────────────────────────────────────────────────────────────

  return (
    <div className="h-full flex">
      {/* Sidebar */}
      <FilesSidebar
        currentPath={currentPath}
        homeDir={homeDir}
        onNavigate={navigateTo}
        onUnpin={handleSidebarUnpin}
        open={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
      />

      {/* Main Content */}
      <DropZone onDrop={handleDropUpload} disabled={uploading} className="flex-1 flex flex-col overflow-hidden">
        {/* Toolbar */}
        <FilesToolbar
          currentPath={currentPath}
          viewMode={viewMode}
          sortBy={sortBy}
          sortOrder={sortOrder}
          searchQuery={searchQuery}
          loading={loading}
          uploading={uploading}
          fileInputRef={fileInputRef}
          selectedCount={selectedPaths.size}
          totalCount={sortedAndFilteredEntries.length}
          onNavigate={navigateTo}
          onGoUp={goUp}
          onRefresh={handleRefresh}
          onViewModeChange={setViewMode}
          onSortByChange={setSortBy}
          onSortOrderToggle={toggleSortOrder}
          onSearchChange={setSearchQuery}
          onNewFolder={() => setNewFolderDialog(true)}
          onNewFile={() => setNewFileDialog(true)}
          onUpload={handleUpload}
          onToggleSidebar={toggleSidebar}
          onSelectAll={handleSelectAll}
          onClearSelection={clearSelection}
          onDeleteSelected={() => setBulkDeleteDialog(true)}
          onDownloadSelected={handleDownloadSelected}
        />

        {/* Content */}
        <div className="flex-1 flex overflow-hidden">
          {/* File list */}
          <div className={cn('overflow-auto', selectedFile ? 'flex-1 md:w-1/2 xl:w-3/5 border-r' : 'flex-1')}>
            {error ? (
              <EmptyState
                icon={AlertCircle}
                title="Error loading directory"
                description={error}
                action={<Button onClick={() => loadDirectory(queryPath)}>Try Again</Button>}
                className="h-full"
              />
            ) : loading ? (
              <div className="h-full flex items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : sortedAndFilteredEntries.length === 0 ? (
              <EmptyState
                icon={searchQuery ? Search : Folder}
                title={searchQuery ? 'No files match your search' : 'Empty directory'}
                description={searchQuery ? 'Try a different search term' : 'Create a new file or folder to get started'}
                className="h-full"
              />
            ) : (
              <div
                className={cn(
                  'p-3',
                  viewMode === 'grid' && 'grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-2'
                )}
              >
                {sortedAndFilteredEntries.map((entry, index) => (
                  <FileEntryItem
                    key={entry.path}
                    entry={entry}
                    viewMode={viewMode}
                    isSelected={selectedFile?.path === entry.path}
                    isChecked={isSelected(entry.path)}
                    isFocused={index === focusedIndex}
                    showCheckbox={selectedPaths.size > 0}
                    isPinned={entry.type === 'directory' ? isPinned(entry.path) : undefined}
                    vsCodeUrl={server.url + '/vscode'}
                    onSelect={handleEntrySelect}
                    onNavigate={navigateTo}
                    onDownload={handleDownload}
                    onDelete={setDeleteDialog}
                    onRename={handleRenameEntry}
                    onCopyPath={copyPath}
                    onEdit={handleEdit}
                    onToggleCheck={handleToggleCheck}
                    onTogglePin={entry.type === 'directory' ? handleTogglePinEntry : undefined}
                  />
                ))}
              </div>
            )}
          </div>

          {/* File preview/editor */}
          {selectedFile && (
            <FilePreview
              file={selectedFile}
              content={fileContent}
              editedContent={editedContent}
              isLoading={fileLoading}
              isEditing={isEditing}
              isSaving={saving}
              hasUnsavedChanges={hasUnsavedChanges}
              downloadUrl={server.getClient().files.getDownloadUrl(selectedFile.path)}
              onEditedContentChange={setEditedContent}
              onStartEdit={() => setIsEditing(true)}
              onCancelEdit={cancelEdit}
              onSave={handleSave}
              onClose={closePreview}
              onLoadContent={() => loadFileContent(selectedFile)}
            />
          )}
        </div>

        {/* Status bar */}
        <div className="border-t px-4 py-2 text-xs text-muted-foreground flex items-center gap-4 bg-muted/30">
          <span>{sortedAndFilteredEntries.length} items</span>
          {selectedPaths.size > 0 && <span className="text-primary">{selectedPaths.size} selected</span>}
          <span className="flex-1" />
          <span className="truncate max-w-md hidden sm:block">{currentPath}</span>
        </div>
      </DropZone>

      {/* Dialogs */}
      <NewFolderDialog
        open={newFolderDialog}
        name={newName}
        onOpenChange={setNewFolderDialog}
        onNameChange={setNewName}
        onCreate={handleCreateFolder}
      />

      <NewFileDialog
        open={newFileDialog}
        name={newName}
        onOpenChange={setNewFileDialog}
        onNameChange={setNewName}
        onCreate={handleCreateFile}
      />

      <DeleteDialog entry={deleteDialog} onClose={() => setDeleteDialog(null)} onDelete={handleDelete} />

      <RenameDialog
        entry={renameDialog}
        name={newName}
        onClose={() => {
          setRenameDialog(null)
          setNewName('')
        }}
        onNameChange={setNewName}
        onRename={handleRename}
      />

      <BulkDeleteDialog
        count={selectedPaths.size}
        open={bulkDeleteDialog}
        onClose={() => setBulkDeleteDialog(false)}
        onDelete={handleBulkDelete}
      />
    </div>
  )
}
