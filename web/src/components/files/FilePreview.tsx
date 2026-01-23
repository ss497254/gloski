import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { FileText, Pencil, Save, X, Loader2 } from 'lucide-react'
import { formatSize, formatDate } from '@/lib/file-utils'
import type { FileEntry } from '@/lib/types'

interface FilePreviewProps {
  file: FileEntry
  content: string | null
  editedContent: string
  isLoading: boolean
  isEditing: boolean
  isSaving: boolean
  onEditedContentChange: (content: string) => void
  onStartEdit: () => void
  onCancelEdit: () => void
  onSave: () => void
  onClose: () => void
}

export function FilePreview({
  file,
  content,
  editedContent,
  isLoading,
  isEditing,
  isSaving,
  onEditedContentChange,
  onStartEdit,
  onCancelEdit,
  onSave,
  onClose,
}: FilePreviewProps) {
  return (
    <div className="w-1/2 flex flex-col bg-muted/20">
      {/* Header */}
      <div className="p-4 border-b flex items-center gap-3 bg-background">
        <FileText className="h-5 w-5 shrink-0 text-muted-foreground" />
        <div className="flex-1 min-w-0">
          <p className="font-medium truncate">{file.name}</p>
          <p className="text-xs text-muted-foreground">
            {formatSize(file.size)} &bull; {formatDate(file.modified)}
          </p>
        </div>
        {!isEditing ? (
          <Button variant="outline" size="sm" onClick={onStartEdit}>
            <Pencil className="h-4 w-4 mr-2" />
            Edit
          </Button>
        ) : (
          <>
            <Button variant="ghost" size="sm" onClick={onCancelEdit}>
              Cancel
            </Button>
            <Button size="sm" onClick={onSave} disabled={isSaving}>
              {isSaving ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Save className="h-4 w-4 mr-2" />
              )}
              Save
            </Button>
          </>
        )}
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Content */}
      <ScrollArea className="flex-1">
        {isLoading ? (
          <div className="h-full flex items-center justify-center">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : isEditing ? (
          <textarea
            value={editedContent}
            onChange={(e) => onEditedContentChange(e.target.value)}
            className="w-full h-full min-h-[500px] p-4 font-mono text-sm bg-transparent resize-none focus:outline-none"
            spellCheck={false}
          />
        ) : (
          <pre className="p-4 text-sm font-mono whitespace-pre-wrap break-all">
            {content}
          </pre>
        )}
      </ScrollArea>
    </div>
  )
}
