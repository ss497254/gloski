import { Input } from '@/ui/input'
import { Badge } from '@/ui/badge'
import { CodeEditor } from '@/shared/components/code-editor'
import { formatRelativeTime } from '@/shared/lib/utils'
import type { Note } from '../stores/notes'

interface NoteEditorProps {
  note: Note
  title: string
  content: string
  onTitleChange: (title: string) => void
  onContentChange: (content: string) => void
}

export function NoteEditor({ note, title, content, onTitleChange, onContentChange }: NoteEditorProps) {
  return (
    <>
      <div className="p-4 border-b">
        <Input
          value={title}
          onChange={(e) => onTitleChange(e.target.value)}
          className="text-xl font-semibold border-0 p-0 h-auto focus-visible:ring-0 bg-transparent!"
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

      <div className="flex-1 min-h-0">
        <CodeEditor
          value={content}
          onChange={onContentChange}
          filename="note.md"
          lineNumbers={false}
          placeholder="Start writing..."
          className="h-full border-0 rounded-none"
        />
      </div>
    </>
  )
}
