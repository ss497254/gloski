import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { useSearchParams, Navigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { useServer } from '@/hooks'
import type { FileEntry, ListResponse } from '@/lib/types'
import { toast } from 'sonner'
import { Folder, Search, Loader2, AlertCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { EmptyState } from '@/components/common'
import { joinPath, getParentPath } from '@/lib/file-utils'
import {
  FileEntryItem,
  FilePreview,
  FilesToolbar,
  NewFolderDialog,
  NewFileDialog,
  DeleteDialog,
  type ViewMode,
  type SortBy,
  type SortOrder,
} from '@/components/files'

export function FilesPage() {
  const { server, api } = useServer()
  const [searchParams, setSearchParams] = useSearchParams()
  const currentPath = searchParams.get('path') || '/'

  // Data state
  const [data, setData] = useState<ListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // File preview state
  const [selectedFile, setSelectedFile] = useState<FileEntry | null>(null)
  const [fileContent, setFileContent] = useState<string | null>(null)
  const [fileLoading, setFileLoading] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [editedContent, setEditedContent] = useState('')
  const [saving, setSaving] = useState(false)

  // UI state
  const [viewMode, setViewMode] = useState<ViewMode>('list')
  const [sortBy, setSortBy] = useState<SortBy>('name')
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc')
  const [searchQuery, setSearchQuery] = useState('')

  // Dialog state
  const [newFolderDialog, setNewFolderDialog] = useState(false)
  const [newFileDialog, setNewFileDialog] = useState(false)
  const [deleteDialog, setDeleteDialog] = useState<FileEntry | null>(null)
  const [newName, setNewName] = useState('')

  // Upload state
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Redirect if no server
  if (!server || !api) {
    return <Navigate to="/" replace />
  }

  // ─────────────────────────────────────────────────────────────────────────────
  // Data Loading
  // ─────────────────────────────────────────────────────────────────────────────

  const loadDirectory = useCallback(
    async (path: string) => {
      if (!api) return
      setLoading(true)
      setError('')
      try {
        const result = await api.listFiles(path)
        setData(result)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load directory')
      } finally {
        setLoading(false)
      }
    },
    [api]
  )

  useEffect(() => {
    loadDirectory(currentPath)
  }, [currentPath, loadDirectory])

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

  // ─────────────────────────────────────────────────────────────────────────────
  // File Operations
  // ─────────────────────────────────────────────────────────────────────────────

  const handleEntrySelect = useCallback(
    async (entry: FileEntry) => {
      if (!api) return
      if (entry.type === 'directory') {
        navigateTo(entry.path)
      } else {
        setSelectedFile(entry)
        setFileLoading(true)
        setIsEditing(false)
        try {
          const result = await api.readFile(entry.path)
          setFileContent(result.content)
          setEditedContent(result.content)
        } catch (err) {
          setFileContent(
            `Error loading file: ${err instanceof Error ? err.message : 'Unknown error'}`
          )
        } finally {
          setFileLoading(false)
        }
      }
    },
    [api, navigateTo]
  )

  const handleEdit = useCallback(
    (entry: FileEntry) => {
      handleEntrySelect(entry)
      setTimeout(() => setIsEditing(true), 100)
    },
    [handleEntrySelect]
  )

  const handleSave = useCallback(async () => {
    if (!api || !selectedFile) return
    setSaving(true)
    try {
      await api.writeFile(selectedFile.path, editedContent)
      setFileContent(editedContent)
      setIsEditing(false)
      toast.success('File saved successfully')
    } catch {
      toast.error('Failed to save file')
    } finally {
      setSaving(false)
    }
  }, [api, selectedFile, editedContent])

  const handleCreateFolder = useCallback(async () => {
    if (!api || !newName.trim()) return
    try {
      await api.mkdir(joinPath(currentPath, newName))
      toast.success('Folder created')
      setNewFolderDialog(false)
      setNewName('')
      loadDirectory(currentPath)
    } catch {
      toast.error('Failed to create folder')
    }
  }, [api, newName, currentPath, loadDirectory])

  const handleCreateFile = useCallback(async () => {
    if (!api || !newName.trim()) return
    try {
      await api.writeFile(joinPath(currentPath, newName), '')
      toast.success('File created')
      setNewFileDialog(false)
      setNewName('')
      loadDirectory(currentPath)
    } catch {
      toast.error('Failed to create file')
    }
  }, [api, newName, currentPath, loadDirectory])

  const handleDelete = useCallback(async () => {
    if (!api || !deleteDialog) return
    try {
      await api.deleteFile(deleteDialog.path)
      toast.success(
        `${deleteDialog.type === 'directory' ? 'Folder' : 'File'} deleted`
      )
      setDeleteDialog(null)
      if (selectedFile?.path === deleteDialog.path) {
        setSelectedFile(null)
        setFileContent(null)
      }
      loadDirectory(currentPath)
    } catch {
      toast.error('Failed to delete')
    }
  }, [api, deleteDialog, selectedFile?.path, currentPath, loadDirectory])

  const handleUpload = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!api) return
      const files = e.target.files
      if (!files || files.length === 0) return

      setUploading(true)
      let successCount = 0
      let failCount = 0

      for (const file of Array.from(files)) {
        try {
          await api.uploadFile(currentPath, file)
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

      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    },
    [api, currentPath, loadDirectory]
  )

  const handleDownload = useCallback(
    (entry: FileEntry) => {
      if (!api) return
      const url = api.getDownloadUrl(entry.path)
      const link = document.createElement('a')
      link.href = url
      link.download = entry.name
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
    },
    [api]
  )

  const copyPath = useCallback((path: string) => {
    navigator.clipboard.writeText(path)
    toast.success('Path copied to clipboard')
  }, [])

  const closePreview = useCallback(() => {
    setSelectedFile(null)
    setFileContent(null)
    setIsEditing(false)
  }, [])

  const cancelEdit = useCallback(() => {
    setIsEditing(false)
    setEditedContent(fileContent || '')
  }, [fileContent])

  // ─────────────────────────────────────────────────────────────────────────────
  // Computed Values
  // ─────────────────────────────────────────────────────────────────────────────

  const sortedAndFilteredEntries = useMemo(() => {
    if (!data?.entries) return []

    let entries = [...data.entries]

    // Filter
    if (searchQuery) {
      entries = entries.filter((e) =>
        e.name.toLowerCase().includes(searchQuery.toLowerCase())
      )
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
          comparison =
            new Date(a.modified).getTime() - new Date(b.modified).getTime()
          break
      }

      return sortOrder === 'asc' ? comparison : -comparison
    })

    return entries
  }, [data?.entries, searchQuery, sortBy, sortOrder])

  // ─────────────────────────────────────────────────────────────────────────────
  // Render
  // ─────────────────────────────────────────────────────────────────────────────

  return (
    <div className="h-full flex flex-col">
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
        onNavigate={navigateTo}
        onGoUp={goUp}
        onRefresh={() => loadDirectory(currentPath)}
        onViewModeChange={setViewMode}
        onSortByChange={setSortBy}
        onSortOrderToggle={() => setSortOrder((o) => (o === 'asc' ? 'desc' : 'asc'))}
        onSearchChange={setSearchQuery}
        onNewFolder={() => setNewFolderDialog(true)}
        onNewFile={() => setNewFileDialog(true)}
        onUpload={handleUpload}
      />

      {/* Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* File list */}
        <div
          className={cn('overflow-auto', selectedFile ? 'w-1/2 border-r' : 'flex-1')}
        >
          {error ? (
            <EmptyState
              icon={AlertCircle}
              title="Error loading directory"
              description={error}
              action={
                <Button onClick={() => loadDirectory(currentPath)}>Try Again</Button>
              }
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
              description={
                searchQuery
                  ? 'Try a different search term'
                  : 'Create a new file or folder to get started'
              }
              className="h-full"
            />
          ) : (
            <div
              className={cn(
                'p-3',
                viewMode === 'grid' &&
                  'grid grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-2'
              )}
            >
              {sortedAndFilteredEntries.map((entry) => (
                <FileEntryItem
                  key={entry.path}
                  entry={entry}
                  viewMode={viewMode}
                  isSelected={selectedFile?.path === entry.path}
                  onSelect={handleEntrySelect}
                  onNavigate={navigateTo}
                  onDownload={handleDownload}
                  onDelete={setDeleteDialog}
                  onCopyPath={copyPath}
                  onEdit={handleEdit}
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
            onEditedContentChange={setEditedContent}
            onStartEdit={() => setIsEditing(true)}
            onCancelEdit={cancelEdit}
            onSave={handleSave}
            onClose={closePreview}
          />
        )}
      </div>

      {/* Status bar */}
      <div className="border-t px-4 py-2 text-xs text-muted-foreground flex items-center gap-4 bg-muted/30">
        <span>{sortedAndFilteredEntries.length} items</span>
        <span className="flex-1" />
        <span className="truncate max-w-md">{currentPath}</span>
      </div>

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

      <DeleteDialog
        entry={deleteDialog}
        onClose={() => setDeleteDialog(null)}
        onDelete={handleDelete}
      />
    </div>
  )
}
