import type { FileEntry } from '@/shared/lib/types'
import { Button } from '@/ui/button'
import { AlertTriangle, Download, FileText, Loader2, Pencil, Save, X } from 'lucide-react'
import { useState } from 'react'
import { formatFileSize, getFileType, isTextFile, shouldWarnLargeFile } from '../lib/file-types'
import { formatDate, formatSize } from '../lib/file-utils'
import { AudioPreview, BinaryPreview, ImagePreview, PdfPreview, TextPreview, VideoPreview } from './previews'

interface FilePreviewProps {
  file: FileEntry
  content: string | null
  editedContent: string
  isLoading: boolean
  isEditing: boolean
  isSaving: boolean
  hasUnsavedChanges: boolean
  downloadUrl: string
  onEditedContentChange: (content: string) => void
  onStartEdit: () => void
  onCancelEdit: () => void
  onSave: () => void
  onClose: () => void
  onLoadContent: () => void
}

export function FilePreview({
  file,
  content,
  editedContent,
  isLoading,
  isEditing,
  isSaving,
  hasUnsavedChanges,
  downloadUrl,
  onEditedContentChange,
  onStartEdit,
  onCancelEdit,
  onSave,
  onClose,
  onLoadContent,
}: FilePreviewProps) {
  const [dismissedWarning, setDismissedWarning] = useState(false)

  const fileType = getFileType(file.name)
  const showLargeFileWarning =
    shouldWarnLargeFile(file.size) && !dismissedWarning && content === null && isTextFile(file.name)
  const canEdit = isTextFile(file.name)

  const handleClose = () => {
    if (hasUnsavedChanges) {
      if (window.confirm('You have unsaved changes. Are you sure you want to close?')) {
        onClose()
      }
    } else {
      onClose()
    }
  }

  const handleDownload = () => {
    window.open(downloadUrl, '_blank')
  }

  const renderPreviewContent = () => {
    // Loading state
    if (isLoading) {
      return (
        <div className="h-full flex items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )
    }

    // Large file warning for text files
    if (showLargeFileWarning) {
      return (
        <div className="h-full flex flex-col items-center justify-center gap-4 p-8 text-center">
          <AlertTriangle className="h-12 w-12 text-yellow-500" />
          <div className="space-y-1">
            <h3 className="font-medium">Large File Warning</h3>
            <p className="text-sm text-muted-foreground">
              This file is {formatFileSize(file.size)}. Opening large files may slow down your browser.
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleDownload}>
              <Download className="h-4 w-4 mr-2" />
              Download Instead
            </Button>
            <Button
              onClick={() => {
                setDismissedWarning(true)
                onLoadContent()
              }}
            >
              Open Anyway
            </Button>
          </div>
        </div>
      )
    }

    // Route to appropriate preview based on file type
    switch (fileType) {
      case 'image':
        return <ImagePreview src={downloadUrl} alt={file.name} />

      case 'video':
        return <VideoPreview src={downloadUrl} filename={file.name} />

      case 'audio':
        return <AudioPreview src={downloadUrl} filename={file.name} />

      case 'pdf':
        return <PdfPreview src={downloadUrl} filename={file.name} />

      case 'code':
      case 'text':
        if (content === null) {
          return (
            <div className="h-full flex items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          )
        }
        return (
          <TextPreview
            content={isEditing ? editedContent : content}
            onChange={onEditedContentChange}
            readOnly={!isEditing}
            filename={file.name}
          />
        )

      case 'binary':
      default:
        return <BinaryPreview filename={file.name} size={file.size} onDownload={handleDownload} />
    }
  }

  return (
    <div className="w-full md:w-1/2 xl:w-2/5 flex flex-col bg-muted/20 border-l">
      {/* Header */}
      <div className="p-3 border-b flex items-center gap-3 bg-background">
        <FileText className="h-5 w-5 shrink-0 text-muted-foreground" />
        <div className="flex-1 min-w-0">
          <p className="font-medium truncate">
            {file.name}
            {hasUnsavedChanges && <span className="text-yellow-500 ml-1">*</span>}
          </p>
          <p className="text-xs text-muted-foreground">
            {formatSize(file.size)} &bull; {formatDate(file.modified)}
          </p>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1">
          {canEdit && !isEditing && (
            <Button variant="outline" size="sm" onClick={onStartEdit}>
              <Pencil className="h-4 w-4 mr-2" />
              Edit
            </Button>
          )}

          {isEditing && (
            <>
              <Button variant="ghost" size="sm" onClick={onCancelEdit}>
                Cancel
              </Button>
              <Button size="sm" onClick={onSave} disabled={isSaving}>
                {isSaving ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Save className="h-4 w-4 mr-2" />}
                Save
              </Button>
            </>
          )}

          {!isEditing && fileType !== 'binary' && (
            <Button variant="ghost" size="icon" onClick={handleDownload} title="Download">
              <Download className="h-4 w-4" />
            </Button>
          )}

          <Button variant="ghost" size="icon" onClick={handleClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Preview Content */}
      <div className="flex-1 overflow-hidden">{renderPreviewContent()}</div>
    </div>
  )
}
