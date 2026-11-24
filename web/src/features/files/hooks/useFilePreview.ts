import { unwrap, type FileEntry } from '@/shared/api'
import type { DeviceClient } from '@/shared/lib/device'
import { useCallback, useRef, useState } from 'react'
import { toast } from 'sonner'
import { getFileType } from '../lib/file-types'

/** Owns preview/editor state and ensures reads only apply to the latest file. */
export function useFilePreview(device: DeviceClient, navigateTo: (path: string) => void) {
  const [selectedFile, setSelectedFile] = useState<FileEntry | null>(null)
  const [fileContent, setFileContent] = useState<string | null>(null)
  const [fileLoading, setFileLoading] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [editedContent, setEditedContent] = useState('')
  const [saving, setSaving] = useState(false)
  const requestRef = useRef(0)
  const selectedPathRef = useRef<string | null>(null)
  const hasUnsavedChanges = isEditing && editedContent !== fileContent

  const clearPreview = useCallback(() => {
    requestRef.current++
    selectedPathRef.current = null
    setSelectedFile(null)
    setFileContent(null)
    setFileLoading(false)
    setIsEditing(false)
  }, [])

  const loadFileContent = useCallback(
    async (entry: FileEntry) => {
      const request = ++requestRef.current
      setFileLoading(true)
      try {
        const result = unwrap(await device.getClient().files.read(entry.path))
        if (request !== requestRef.current) return
        setFileContent(result.content)
        setEditedContent(result.content)
      } catch (err) {
        if (request === requestRef.current) {
          setFileContent(`Error loading file: ${err instanceof Error ? err.message : 'Unknown error'}`)
        }
      } finally {
        if (request === requestRef.current) setFileLoading(false)
      }
    },
    [device]
  )

  const handleEntrySelect = useCallback(
    async (entry: FileEntry) => {
      if (entry.type === 'directory') {
        clearPreview()
        navigateTo(entry.path)
        return
      }

      requestRef.current++
      selectedPathRef.current = entry.path
      setSelectedFile(entry)
      setFileContent(null)
      setEditedContent('')
      setIsEditing(false)

      const fileType = getFileType(entry.name)
      if (fileType === 'code' || fileType === 'text') {
        await loadFileContent(entry)
      } else {
        requestRef.current++
        setFileLoading(false)
      }
    },
    [clearPreview, loadFileContent, navigateTo]
  )

  const handleEdit = useCallback(
    async (entry: FileEntry) => {
      await handleEntrySelect(entry)
      if (
        entry.type === 'file' &&
        selectedPathRef.current === entry.path &&
        (getFileType(entry.name) === 'code' || getFileType(entry.name) === 'text')
      ) {
        setIsEditing(true)
      }
    },
    [handleEntrySelect]
  )

  const handleSave = useCallback(async () => {
    if (!selectedFile) return
    setSaving(true)
    try {
      await device.getClient().files.write(selectedFile.path, editedContent)
      setFileContent(editedContent)
      setIsEditing(false)
      toast.success('File saved successfully')
    } catch {
      toast.error('Failed to save file')
    } finally {
      setSaving(false)
    }
  }, [device, editedContent, selectedFile])

  const closePreview = useCallback(() => {
    if (hasUnsavedChanges && !window.confirm('You have unsaved changes. Are you sure you want to close?')) return
    clearPreview()
  }, [clearPreview, hasUnsavedChanges])

  const cancelEdit = useCallback(() => {
    setIsEditing(false)
    setEditedContent(fileContent || '')
  }, [fileContent])

  const startEdit = useCallback(() => setIsEditing(true), [])

  return {
    selectedFile,
    setSelectedFile,
    fileContent,
    setFileContent,
    fileLoading,
    setFileLoading,
    isEditing,
    setIsEditing,
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
  }
}
