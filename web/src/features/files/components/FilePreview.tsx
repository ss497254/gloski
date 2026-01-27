import { useServer } from '@/features/servers'
import { Button } from '@/ui/button'
import { AlertTriangle, Download, FileText, Loader2, Pencil, Save, X } from 'lucide-react'
import { useState } from 'react'
import { useFiles } from '../context'
import { formatFileSize, getFileType, isTextFile, shouldWarnLargeFile } from '../lib/file-types'
import { formatDate, formatSize } from '../lib/file-utils'
import { AudioPreview, BinaryPreview, ImagePreview, PdfPreview, TextPreview, VideoPreview } from './previews'

export function FilePreview() {
  const { server } = useServer()
  const {
    selectedFile,
    fileContent,
    editedContent,
    fileLoading,
    isEditing,
    saving,
    hasUnsavedChanges,
    setEditedContent,
    handleSave,
    closePreview,
    cancelEdit,
    startEdit,
    loadFileContent,
  } = useFiles()

  const [dismissedWarning, setDismissedWarning] = useState(false)

  // Don't render if no file is selected
  if (!selectedFile) return null

  const downloadUrl = server.getClient().files.getDownloadUrl(selectedFile.path)
  const fileType = getFileType(selectedFile.name)
  const showLargeFileWarning =
    shouldWarnLargeFile(selectedFile.size) && !dismissedWarning && fileContent === null && isTextFile(selectedFile.name)
  const canEdit = isTextFile(selectedFile.name)

  const handleClose = () => {
    if (hasUnsavedChanges) {
      if (window.confirm('You have unsaved changes. Are you sure you want to close?')) {
        closePreview()
      }
    } else {
      closePreview()
    }
  }

  const handleDownload = () => {
    window.open(downloadUrl, '_blank')
  }



  const renderPreviewContent = () => {
    // Loading state
    if (fileLoading) {
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
              This file is {formatFileSize(selectedFile.size)}. Opening large files may slow down your browser.
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
                loadFileContent(selectedFile)
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
        return <ImagePreview src={downloadUrl} alt={selectedFile.name} />

      case 'video':
        return <VideoPreview src={downloadUrl} filename={selectedFile.name} />

      case 'audio':
        return <AudioPreview src={downloadUrl} filename={selectedFile.name} />

      case 'pdf':
        return <PdfPreview src={downloadUrl} filename={selectedFile.name} />

      case 'code':
      case 'text':
        if (fileContent === null) {
          return (
            <div className="h-full flex items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          )
        }
        return (
          <TextPreview
            content={isEditing ? editedContent : fileContent}
            onChange={setEditedContent}
            readOnly={!isEditing}
            filename={selectedFile.name}
          />
        )

      case 'binary':
      default:
        return <BinaryPreview filename={selectedFile.name} size={selectedFile.size} onDownload={handleDownload} />
    }
  }

  return (
    <div className="w-full md:w-1/2 xl:w-2/5 flex flex-col bg-muted/20 border-l">
      {/* Header */}
      <div className="p-3 border-b flex items-center gap-3 bg-background">
        <FileText className="h-5 w-5 shrink-0 text-muted-foreground" />
        <div className="flex-1 min-w-0">
          <p className="font-medium truncate">
            {selectedFile.name}
            {hasUnsavedChanges && <span className="text-yellow-500 ml-1">*</span>}
          </p>
          <p className="text-xs text-muted-foreground">
            {formatSize(selectedFile.size)} &bull; {formatDate(selectedFile.modified)}
          </p>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1">
          {canEdit && !isEditing && (
            <Button variant="outline" size="sm" onClick={startEdit}>
              <Pencil className="h-4 w-4 mr-2" />
              Edit
            </Button>
          )}

          {isEditing && (
            <>
              <Button variant="ghost" size="sm" onClick={cancelEdit}>
                Cancel
              </Button>
              <Button size="sm" onClick={handleSave} disabled={saving}>
                {saving ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Save className="h-4 w-4 mr-2" />}
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
