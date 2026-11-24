import { pinnedFolderId as toPinnedFolderId, unwrap, type FileEntry } from '@/shared/api'
import type { DeviceClient } from '@/shared/lib/device'
import { useCallback, useRef, useState, type ChangeEvent } from 'react'
import { toast } from 'sonner'
import { getParentPath, joinPath } from '../lib/file-utils'
import { useFilesStore } from '../stores/files'

interface UseFileOperationsOptions {
  device: DeviceClient
  currentPath: string
  entries: FileEntry[]
  selectedFile: FileEntry | null
  clearPreview: () => void
  loadDirectory: (path: string) => Promise<void>
}

/** Groups file mutations, upload state and their dialogs behind one feature hook. */
export function useFileOperations({
  device,
  currentPath,
  entries,
  selectedFile,
  clearPreview,
  loadDirectory,
}: UseFileOperationsOptions) {
  const setPinnedFolders = useFilesStore((s) => s.setPinnedFolders)
  const addPinnedFolder = useFilesStore((s) => s.addPinnedFolder)
  const removePinnedFolder = useFilesStore((s) => s.removePinnedFolder)
  const isPinned = useFilesStore((s) => s.isPinned)
  const getPinnedFolderByPath = useFilesStore((s) => s.getPinnedFolderByPath)
  const selectedPaths = useFilesStore((s) => s.selectedPaths)
  const clearSelection = useFilesStore((s) => s.clearSelection)

  const [newFolderDialog, setNewFolderDialog] = useState(false)
  const [newFileDialog, setNewFileDialog] = useState(false)
  const [deleteDialog, setDeleteDialog] = useState<FileEntry | null>(null)
  const [renameDialog, setRenameDialog] = useState<FileEntry | null>(null)
  const [bulkDeleteDialog, setBulkDeleteDialog] = useState(false)
  const [newName, setNewName] = useState('')
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const selectedFilePath = selectedFile?.path

  const handleCreateFolder = useCallback(async () => {
    if (!newName.trim()) return
    try {
      await device.getClient().files.mkdir(joinPath(currentPath, newName))
      toast.success('Folder created')
      setNewFolderDialog(false)
      setNewName('')
      void loadDirectory(currentPath)
    } catch {
      toast.error('Failed to create folder')
    }
  }, [device, newName, currentPath, loadDirectory])

  const handleCreateFile = useCallback(async () => {
    if (!newName.trim()) return
    try {
      await device.getClient().files.write(joinPath(currentPath, newName), '')
      toast.success('File created')
      setNewFileDialog(false)
      setNewName('')
      void loadDirectory(currentPath)
    } catch {
      toast.error('Failed to create file')
    }
  }, [device, newName, currentPath, loadDirectory])

  const handleDelete = useCallback(async () => {
    if (!deleteDialog) return
    try {
      await device.getClient().files.delete(deleteDialog.path)
      toast.success(`${deleteDialog.type === 'directory' ? 'Folder' : 'File'} deleted`)
      setDeleteDialog(null)
      if (selectedFilePath === deleteDialog.path) clearPreview()
      void loadDirectory(currentPath)
    } catch {
      toast.error('Failed to delete')
    }
  }, [device, deleteDialog, selectedFilePath, currentPath, loadDirectory, clearPreview])

  const handleRename = useCallback(async () => {
    if (!renameDialog || !newName.trim()) return
    const newPath = joinPath(getParentPath(renameDialog.path), newName)
    try {
      await device.getClient().files.rename(renameDialog.path, newPath)
      toast.success('Renamed successfully')
      setRenameDialog(null)
      setNewName('')
      if (selectedFilePath === renameDialog.path) clearPreview()
      void loadDirectory(currentPath)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to rename')
    }
  }, [device, renameDialog, newName, selectedFilePath, currentPath, loadDirectory, clearPreview])

  const handleBulkDelete = useCallback(async () => {
    if (selectedPaths.size === 0) return
    const results = await Promise.allSettled(
      Array.from(selectedPaths).map((path) => device.getClient().files.delete(path))
    )
    const successCount = results.filter((result) => result.status === 'fulfilled').length
    const failCount = results.length - successCount

    setBulkDeleteDialog(false)
    clearSelection()
    if (successCount > 0) {
      toast.success(`Deleted ${successCount} item(s)`)
      void loadDirectory(currentPath)
    }
    if (failCount > 0) toast.error(`Failed to delete ${failCount} item(s)`)
  }, [device, selectedPaths, currentPath, loadDirectory, clearSelection])

  const handleDropUpload = useCallback(
    async (files: File[]) => {
      if (files.length === 0) return
      setUploading(true)
      let successCount = 0
      let failCount = 0
      try {
        for (const file of files) {
          try {
            await device.getClient().files.upload(currentPath, file)
            successCount++
          } catch (err) {
            failCount++
            console.error(`Failed to upload ${file.name}:`, err)
          }
        }
      } finally {
        setUploading(false)
      }

      if (successCount > 0) {
        toast.success(`Uploaded ${successCount} file(s)`)
        void loadDirectory(currentPath)
      }
      if (failCount > 0) toast.error(`Failed to upload ${failCount} file(s)`)
    },
    [device, currentPath, loadDirectory]
  )

  const handleUpload = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const files = event.target.files
      if (!files?.length) return
      void handleDropUpload(Array.from(files))
      if (fileInputRef.current) fileInputRef.current.value = ''
    },
    [handleDropUpload]
  )

  const handleDownload = useCallback(
    (entry: FileEntry) => {
      const link = document.createElement('a')
      link.href = device.getClient().files.getDownloadUrl(entry.path)
      link.download = entry.name
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
    },
    [device]
  )

  const handleDownloadSelected = useCallback(() => {
    entries.filter((entry) => selectedPaths.has(entry.path) && entry.type === 'file').forEach(handleDownload)
  }, [entries, selectedPaths, handleDownload])

  const copyPath = useCallback((path: string) => {
    void navigator.clipboard.writeText(path).then(
      () => toast.success('Path copied to clipboard'),
      () => toast.error('Could not copy path')
    )
  }, [])

  const handleTogglePinEntry = useCallback(
    async (entry: FileEntry) => {
      if (entry.type !== 'directory') return
      if (isPinned(entry.path)) {
        const folder = getPinnedFolderByPath(entry.path)
        if (!folder) return
        try {
          await device.getClient().files.pinned.unpin(folder.id)
          removePinnedFolder(folder.id)
          toast.success('Folder unpinned')
        } catch {
          toast.error('Failed to unpin folder')
        }
      } else {
        try {
          const folder = unwrap(await device.getClient().files.pinned.pin(entry.path, entry.name))
          addPinnedFolder(folder)
          toast.success('Folder pinned')
        } catch {
          toast.error('Failed to pin folder')
        }
      }
    },
    [device, isPinned, getPinnedFolderByPath, removePinnedFolder, addPinnedFolder]
  )

  const handleSidebarUnpin = useCallback(
    async (id: string) => {
      try {
        await device.getClient().files.pinned.unpin(toPinnedFolderId(id))
        removePinnedFolder(id)
        toast.success('Folder unpinned')
      } catch {
        toast.error('Failed to unpin folder')
      }
    },
    [device, removePinnedFolder]
  )

  const handleRenameEntry = useCallback((entry: FileEntry) => {
    setRenameDialog(entry)
    setNewName(entry.name)
  }, [])

  return {
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
    isDialogOpen: !!newFolderDialog || !!newFileDialog || !!deleteDialog || !!renameDialog || bulkDeleteDialog,
    setPinnedFolders,
  }
}
