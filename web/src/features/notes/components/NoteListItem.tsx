import { Button } from '@/ui/button'
import { Badge } from '@/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import { Pin, Trash2, MoreVertical } from 'lucide-react'
import { cn, formatRelativeTime } from '@/shared/lib/utils'
import type { Note } from '../stores/notes'

interface NoteListItemProps {
  note: Note
  isSelected: boolean
  onSelect: () => void
  onTogglePin: () => void
  onDelete: () => void
}

export function NoteListItem({
  note,
  isSelected,
  onSelect,
  onTogglePin,
  onDelete,
}: NoteListItemProps) {
  return (
    <button
      onClick={onSelect}
      className={cn(
        'w-full text-left p-3 rounded-lg transition-colors group',
        isSelected ? 'bg-accent' : 'hover:bg-muted/50'
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            {note.pinned && (
              <Pin className="h-3 w-3 text-primary shrink-0 fill-current" />
            )}
            <span className="font-medium truncate">{note.title}</span>
          </div>
          <p className="text-sm text-muted-foreground line-clamp-2 mt-1">
            {note.content.slice(0, 100)}
          </p>
          <div className="flex items-center gap-2 mt-2">
            <span className="text-xs text-muted-foreground">
              {formatRelativeTime(note.updatedAt)}
            </span>
            {note.folder && (
              <Badge variant="secondary" className="text-xs h-5">
                {note.folder}
              </Badge>
            )}
          </div>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 opacity-0 group-hover:opacity-100 shrink-0"
              onClick={(e) => e.stopPropagation()}
            >
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onTogglePin}>
              <Pin className="h-4 w-4 mr-2" />
              {note.pinned ? 'Unpin' : 'Pin'}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={onDelete} className="text-destructive">
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </button>
  )
}
