import { Input } from '@/ui/input'
import { Badge } from '@/ui/badge'
import { formatRelativeTime } from '@/shared/lib/utils'
import type { Note } from '../stores/notes'

interface NoteEditorProps {
  note: Note
  title: string
  content: string
  onTitleChange: (title: string) => void
  onContentChange: (content: string) => void
}

export function NoteEditor({
  note,
  title,
  content,
  onTitleChange,
  onContentChange,
}: NoteEditorProps) {
  return (
    <>
      <div className="p-4 border-b">
        <Input
          value={title}
          onChange={(e) => onTitleChange(e.target.value)}
          className="text-xl font-semibold border-0 p-0 h-auto focus-visible:ring-0"
          placeholder="Note title"
        />
        <div className="flex items-center gap-2 mt-2 text-xs text-muted-foreground">
          <span>Last edited {formatRelativeTime(note.updatedAt)}</span>
          {note.folder && (
            <Badge variant="secondary" className="text-xs">
              {note.folder}
            </Badge>
          )}
        </div>
      </div>

      <div className="flex-1 p-4">
        <textarea
          value={content}
          onChange={(e) => onContentChange(e.target.value)}
          className="w-full h-full resize-none bg-transparent border-0 focus:outline-none font-mono text-sm"
          placeholder="Start writing..."
        />
      </div>
    </>
  )
}
