import type { RefObject } from 'react'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import {
  Home,
  ChevronRight,
  ArrowLeft,
  RefreshCw,
  FolderPlus,
  FilePlus,
  Upload,
  LayoutGrid,
  List,
  Search,
  X,
  SortAsc,
  SortDesc,
  FileType,
  HardDrive,
  Clock,
  ChevronDown,
  Loader2,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { getPathParts } from '../lib/file-utils'
import type { ViewMode } from './FileEntryItem'

export type SortBy = 'name' | 'size' | 'modified'
export type SortOrder = 'asc' | 'desc'

interface FilesToolbarProps {
  currentPath: string
  viewMode: ViewMode
  sortBy: SortBy
  sortOrder: SortOrder
  searchQuery: string
  loading: boolean
  uploading: boolean
  fileInputRef: RefObject<HTMLInputElement | null>
  onNavigate: (path: string) => void
  onGoUp: () => void
  onRefresh: () => void
  onViewModeChange: (mode: ViewMode) => void
  onSortByChange: (sortBy: SortBy) => void
  onSortOrderToggle: () => void
  onSearchChange: (query: string) => void
  onNewFolder: () => void
  onNewFile: () => void
  onUpload: (e: React.ChangeEvent<HTMLInputElement>) => void
}

export function FilesToolbar({
  currentPath,
  viewMode,
  sortBy,
  sortOrder,
  searchQuery,
  loading,
  uploading,
  fileInputRef,
  onNavigate,
  onGoUp,
  onRefresh,
  onViewModeChange,
  onSortByChange,
  onSortOrderToggle,
  onSearchChange,
  onNewFolder,
  onNewFile,
  onUpload,
}: FilesToolbarProps) {
  const pathParts = getPathParts(currentPath)

  return (
    <div className="sticky top-0 z-10 bg-background border-b">
      {/* Navigation Bar */}
      <div className="flex items-center gap-2 px-4 py-3">
        {/* Back button */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onGoUp}
          disabled={currentPath === '/'}
          className="shrink-0"
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>

        {/* Breadcrumb */}
        <div className="flex items-center gap-1 flex-1 min-w-0 overflow-x-auto scrollbar-none">
          <Button
            variant="ghost"
            size="sm"
            className="shrink-0"
            onClick={() => onNavigate('/')}
          >
            <Home className="h-4 w-4" />
          </Button>
          {pathParts.map((part, index) => (
            <div key={index} className="flex items-center shrink-0">
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onNavigate('/' + pathParts.slice(0, index + 1).join('/'))}
                className="truncate max-w-40"
              >
                {part}
              </Button>
            </div>
          ))}
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 shrink-0">
          <Button variant="ghost" size="icon" onClick={onNewFolder} title="New Folder">
            <FolderPlus className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" onClick={onNewFile} title="New File">
            <FilePlus className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => fileInputRef.current?.click()}
            title="Upload Files"
            disabled={uploading}
          >
            {uploading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Upload className="h-4 w-4" />
            )}
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={onUpload}
          />
          <div className="w-px h-6 bg-border mx-1" />
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onViewModeChange(viewMode === 'list' ? 'grid' : 'list')}
            title={viewMode === 'list' ? 'Grid view' : 'List view'}
          >
            {viewMode === 'list' ? (
              <LayoutGrid className="h-4 w-4" />
            ) : (
              <List className="h-4 w-4" />
            )}
          </Button>
          <Button variant="ghost" size="icon" onClick={onRefresh} title="Refresh">
            <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
          </Button>
        </div>
      </div>

      {/* Search and Sort bar */}
      <div className="flex items-center gap-3 px-4 pb-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Filter files..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="pl-9 h-9"
          />
          {searchQuery && (
            <Button
              variant="ghost"
              size="icon"
              className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7"
              onClick={() => onSearchChange('')}
            >
              <X className="h-3 w-3" />
            </Button>
          )}
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="gap-2">
              {sortOrder === 'asc' ? (
                <SortAsc className="h-4 w-4" />
              ) : (
                <SortDesc className="h-4 w-4" />
              )}
              Sort by {sortBy}
              <ChevronDown className="h-3 w-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => onSortByChange('name')}>
              <FileType className="h-4 w-4 mr-2" />
              Name
              {sortBy === 'name' && <span className="ml-auto">✓</span>}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onSortByChange('size')}>
              <HardDrive className="h-4 w-4 mr-2" />
              Size
              {sortBy === 'size' && <span className="ml-auto">✓</span>}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onSortByChange('modified')}>
              <Clock className="h-4 w-4 mr-2" />
              Modified
              {sortBy === 'modified' && <span className="ml-auto">✓</span>}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={onSortOrderToggle}>
              {sortOrder === 'asc' ? (
                <>
                  <SortDesc className="h-4 w-4 mr-2" />
                  Descending
                </>
              ) : (
                <>
                  <SortAsc className="h-4 w-4 mr-2" />
                  Ascending
                </>
              )}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}
