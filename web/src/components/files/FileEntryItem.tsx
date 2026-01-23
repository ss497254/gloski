import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from '@/components/ui/context-menu'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Folder,
  Trash2,
  Pencil,
  Terminal,
  Eye,
  Download,
  MoreHorizontal,
  Copy,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { getFileIcon, getFileColor } from '@/lib/file-icons'
import { formatSize, formatDate } from '@/lib/file-utils'
import type { FileEntry } from '@/lib/types'

export type ViewMode = 'list' | 'grid'

interface FileEntryItemProps {
  entry: FileEntry
  viewMode: ViewMode
  isSelected: boolean
  onSelect: (entry: FileEntry) => void
  onNavigate: (path: string) => void
  onDownload: (entry: FileEntry) => void
  onDelete: (entry: FileEntry) => void
  onCopyPath: (path: string) => void
  onEdit: (entry: FileEntry) => void
}

export function FileEntryItem({
  entry,
  viewMode,
  isSelected,
  onSelect,
  onNavigate,
  onDownload,
  onDelete,
  onCopyPath,
  onEdit,
}: FileEntryItemProps) {
  const Icon = getFileIcon(entry.name, entry.type)
  const color = getFileColor(entry.name, entry.type)

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <div
          className={cn(
            'group flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer transition-all',
            'hover:bg-accent/50',
            isSelected && 'bg-accent',
            viewMode === 'grid' && 'flex-col p-4 text-center hover:bg-accent/50'
          )}
          onClick={() => onSelect(entry)}
          onDoubleClick={() => entry.type === 'directory' && onNavigate(entry.path)}
        >
          <Icon
            className={cn(
              'shrink-0 transition-transform group-hover:scale-110',
              color,
              viewMode === 'grid' ? 'h-12 w-12' : 'h-5 w-5'
            )}
          />
          <div className={cn('min-w-0 flex-1', viewMode === 'grid' && 'text-center')}>
            <p className="truncate text-sm font-medium">{entry.name}</p>
            {viewMode === 'grid' && entry.type === 'file' && (
              <p className="text-xs text-muted-foreground mt-1">
                {formatSize(entry.size)}
              </p>
            )}
          </div>
          {viewMode === 'list' && (
            <>
              <span className="text-xs text-muted-foreground w-20 text-right shrink-0">
                {entry.type === 'file' ? formatSize(entry.size) : '--'}
              </span>
              <span className="text-xs text-muted-foreground w-24 text-right shrink-0">
                {formatDate(entry.modified)}
              </span>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 opacity-0 group-hover:opacity-100 shrink-0"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  {entry.type === 'file' && (
                    <>
                      <DropdownMenuItem onClick={() => onSelect(entry)}>
                        <Eye className="h-4 w-4 mr-2" />
                        View
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => onDownload(entry)}>
                        <Download className="h-4 w-4 mr-2" />
                        Download
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                    </>
                  )}
                  <DropdownMenuItem onClick={() => onCopyPath(entry.path)}>
                    <Copy className="h-4 w-4 mr-2" />
                    Copy Path
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    className="text-destructive focus:text-destructive"
                    onClick={() => onDelete(entry)}
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          )}
        </div>
      </ContextMenuTrigger>
      <ContextMenuContent>
        {entry.type === 'file' && (
          <>
            <ContextMenuItem onClick={() => onSelect(entry)}>
              <Eye className="h-4 w-4 mr-2" />
              View
            </ContextMenuItem>
            <ContextMenuItem onClick={() => onEdit(entry)}>
              <Pencil className="h-4 w-4 mr-2" />
              Edit
            </ContextMenuItem>
            <ContextMenuItem onClick={() => onDownload(entry)}>
              <Download className="h-4 w-4 mr-2" />
              Download
            </ContextMenuItem>
            <ContextMenuSeparator />
          </>
        )}
        {entry.type === 'directory' && (
          <>
            <ContextMenuItem onClick={() => onNavigate(entry.path)}>
              <Folder className="h-4 w-4 mr-2" />
              Open
            </ContextMenuItem>
            <ContextMenuItem asChild>
              <Link to={`/terminal?cwd=${encodeURIComponent(entry.path)}`}>
                <Terminal className="h-4 w-4 mr-2" />
                Open in Terminal
              </Link>
            </ContextMenuItem>
            <ContextMenuSeparator />
          </>
        )}
        <ContextMenuItem onClick={() => onCopyPath(entry.path)}>
          <Copy className="h-4 w-4 mr-2" />
          Copy Path
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem
          className="text-destructive focus:text-destructive"
          onClick={() => onDelete(entry)}
        >
          <Trash2 className="h-4 w-4 mr-2" />
          Delete
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  )
}
