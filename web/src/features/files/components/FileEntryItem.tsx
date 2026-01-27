import type { FileEntry } from '@/shared/lib/types'
import { cn } from '@/shared/lib/utils'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from '@/ui/context-menu'
import {
  CheckSquare,
  Code,
  Copy,
  Download,
  Eye,
  Folder,
  Pencil,
  Pin,
  PinOff,
  Square,
  Terminal,
  Trash2,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import { getFileColor, getFileIcon } from '../lib/file-icons'
import { formatDate, formatSize } from '../lib/file-utils'

export type ViewMode = 'list' | 'grid'

interface FileEntryItemProps {
  entry: FileEntry
  viewMode: ViewMode
  isSelected: boolean
  isChecked: boolean
  isFocused: boolean
  showCheckbox: boolean
  isPinned?: boolean
  vsCodeUrl: string
  onSelect: (entry: FileEntry) => void
  onNavigate: (path: string) => void
  onDownload: (entry: FileEntry) => void
  onDelete: (entry: FileEntry) => void
  onRename: (entry: FileEntry) => void
  onCopyPath: (path: string) => void
  onEdit: (entry: FileEntry) => void
  onToggleCheck: (entry: FileEntry, shiftKey: boolean) => void
  onTogglePin?: (entry: FileEntry) => void
}

export function FileEntryItem({
  entry,
  viewMode,
  isSelected,
  isChecked,
  isFocused,
  showCheckbox,
  isPinned,
  vsCodeUrl,
  onSelect,
  onNavigate,
  onDownload,
  onDelete,
  onRename,
  onCopyPath,
  onEdit,
  onToggleCheck,
  onTogglePin,
}: FileEntryItemProps) {
  const Icon = getFileIcon(entry.name, entry.type)
  const color = getFileColor(entry.name, entry.type)

  const handleClick = (e: React.MouseEvent) => {
    if (e.ctrlKey || e.metaKey) {
      onToggleCheck(entry, false)
    } else if (e.shiftKey) {
      onToggleCheck(entry, true)
    } else {
      onSelect(entry)
    }
  }

  const handleDoubleClick = () => {
    if (entry.type === 'directory') {
      onNavigate(entry.path)
    }
  }

  const handleCheckboxClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    onToggleCheck(entry, e.shiftKey)
  }

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <div
          className={cn(
            'group flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer transition-all',
            'hover:bg-accent/50',
            isSelected && 'bg-accent',
            isChecked && 'bg-primary/10',
            isFocused && 'ring-2 ring-primary ring-inset',
            viewMode === 'grid' && 'flex-col p-4 text-center hover:bg-accent/50'
          )}
          onClick={handleClick}
          onDoubleClick={handleDoubleClick}
        >
          {/* Checkbox */}
          {viewMode === 'list' && (
            <button
              onClick={handleCheckboxClick}
              className={cn(
                'shrink-0 transition-opacity',
                showCheckbox || isChecked ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'
              )}
            >
              {isChecked ? (
                <CheckSquare className="h-4 w-4 text-primary" />
              ) : (
                <Square className="h-4 w-4 text-muted-foreground" />
              )}
            </button>
          )}

          {/* eslint-disable-next-line react-hooks/static-components */}
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
              <p className="text-xs text-muted-foreground mt-1">{formatSize(entry.size)}</p>
            )}
          </div>
          {viewMode === 'list' && (
            <>
              <span className="text-xs text-muted-foreground w-20 text-right shrink-0 hidden sm:block">
                {entry.type === 'file' ? formatSize(entry.size) : '--'}
              </span>
              <span className="text-xs text-muted-foreground w-24 text-right shrink-0 hidden md:block">
                {formatDate(entry.modified)}
              </span>
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
              <Link to={`/servers/[id]/terminal?cwd=${encodeURIComponent(entry.path)}`}>
                <Terminal className="h-4 w-4 mr-2" />
                Open in Terminal
              </Link>
            </ContextMenuItem>
            <ContextMenuItem
              onClick={() => window.open(`${vsCodeUrl}/vscode?folder=${encodeURIComponent(entry.path)}`, '_blank')}
            >
              <Code className="h-4 w-4 mr-2" />
              Open in VS Code
            </ContextMenuItem>
            <ContextMenuSeparator />
            {onTogglePin && (
              <ContextMenuItem onClick={() => onTogglePin(entry)}>
                {isPinned ? (
                  <>
                    <PinOff className="h-4 w-4 mr-2" />
                    Unpin Folder
                  </>
                ) : (
                  <>
                    <Pin className="h-4 w-4 mr-2" />
                    Pin Folder
                  </>
                )}
              </ContextMenuItem>
            )}
            <ContextMenuSeparator />
          </>
        )}
        <ContextMenuItem onClick={() => onRename(entry)}>
          <Pencil className="h-4 w-4 mr-2" />
          Rename
        </ContextMenuItem>
        <ContextMenuItem onClick={() => onCopyPath(entry.path)}>
          <Copy className="h-4 w-4 mr-2" />
          Copy Path
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem className="text-destructive focus:text-destructive" onClick={() => onDelete(entry)}>
          <Trash2 className="h-4 w-4 mr-2" />
          Delete
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  )
}
