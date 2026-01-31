import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import { Input } from '@/ui/input'
import {
  ArrowLeft,
  CheckSquare,
  ChevronDown,
  ChevronRight,
  Clock,
  Download,
  FilePlus,
  FileType,
  FolderPlus,
  HardDrive,
  Home,
  LayoutGrid,
  List,
  Loader2,
  Menu,
  RefreshCw,
  Search,
  SortAsc,
  SortDesc,
  Trash2,
  Upload,
  X,
} from 'lucide-react'
import { useFiles } from '../context'
import { getPathParts } from '../lib/file-utils'

export type ViewMode = 'list' | 'grid'
export type SortBy = 'name' | 'size' | 'modified'
export type SortOrder = 'asc' | 'desc'

export function FilesToolbar() {
  const {
    currentPath,
    viewMode,
    sortBy,
    sortOrder,
    searchQuery,
    loading,
    uploading,
    fileInputRef,
    selectedPaths,
    sortedAndFilteredEntries,
    navigateTo,
    goUp,
    refresh,
    setViewMode,
    setSortBy,
    toggleSortOrder,
    setSearchQuery,
    setNewFolderDialog,
    setNewFileDialog,
    handleUpload,
    toggleSidebar,
    handleSelectAll,
    clearSelection,
    setBulkDeleteDialog,
    handleDownloadSelected,
  } = useFiles()

  const pathParts = getPathParts(currentPath)
  const selectedCount = selectedPaths.size
  const totalCount = sortedAndFilteredEntries.length
  const hasSelection = selectedCount > 0

  return (
    <div className="sticky top-0 z-10 bg-background border-b">
      {/* Bulk Action Bar - shown when items are selected */}
      {hasSelection ? (
        <div className="flex items-center gap-3 px-4 py-3 bg-accent/50">
          <Button variant="ghost" size="sm" onClick={clearSelection} className="gap-2">
            <X className="h-4 w-4" />
            Clear
          </Button>

          <span className="text-sm font-medium">{selectedCount} selected</span>

          <div className="flex-1" />

          <Button
            variant="outline"
            size="sm"
            onClick={selectedCount === totalCount ? clearSelection : handleSelectAll}
            className="gap-2"
          >
            <CheckSquare className="h-4 w-4" />
            {selectedCount === totalCount ? 'Deselect All' : 'Select All'}
          </Button>

          <Button variant="outline" size="sm" onClick={handleDownloadSelected} className="gap-2">
            <Download className="h-4 w-4" />
            Download
          </Button>

          <Button variant="destructive" size="sm" onClick={() => setBulkDeleteDialog(true)} className="gap-2">
            <Trash2 className="h-4 w-4" />
            Delete
          </Button>
        </div>
      ) : (
        <>
          {/* Navigation Bar */}
          <div className="flex items-center gap-2 px-4 py-3">
            {/* Sidebar toggle (mobile) */}
            <Button variant="ghost" size="icon" onClick={toggleSidebar} className="shrink-0 md:hidden">
              <Menu className="h-4 w-4" />
            </Button>

            {/* Back button */}
            <Button variant="ghost" size="icon" onClick={goUp} disabled={currentPath === '/'} className="shrink-0">
              <ArrowLeft className="h-4 w-4" />
            </Button>

            {/* Breadcrumb */}
            <div className="flex items-center gap-1 flex-1 min-w-0 overflow-x-auto scrollbar-none">
              <Button variant="ghost" size="sm" className="shrink-0" onClick={() => navigateTo('/')}>
                <Home className="h-4 w-4" />
              </Button>
              {pathParts.map((part, index) => (
                <div key={index} className="flex items-center shrink-0">
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => navigateTo(pathParts.slice(0, index + 1).join('/'))}
                    className="truncate max-w-40"
                  >
                    {part}
                  </Button>
                </div>
              ))}
            </div>

            {/* Actions */}
            <div className="flex items-center gap-1 shrink-0">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setNewFolderDialog(true)}
                title="New Folder"
                className="hidden sm:flex"
              >
                <FolderPlus className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setNewFileDialog(true)}
                title="New File"
                className="hidden sm:flex"
              >
                <FilePlus className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => fileInputRef.current?.click()}
                title="Upload Files"
                disabled={uploading}
              >
                {uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
              </Button>
              <input ref={fileInputRef} type="file" multiple className="hidden" onChange={handleUpload} />
              <div className="w-px h-6 bg-border mx-1" />
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setViewMode(viewMode === 'list' ? 'grid' : 'list')}
                title={viewMode === 'list' ? 'Grid view' : 'List view'}
              >
                {viewMode === 'list' ? <LayoutGrid className="h-4 w-4" /> : <List className="h-4 w-4" />}
              </Button>
              <Button variant="ghost" size="icon" onClick={refresh} title="Refresh">
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
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 h-9"
              />
              {searchQuery && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7"
                  onClick={() => setSearchQuery('')}
                >
                  <X className="h-3 w-3" />
                </Button>
              )}
            </div>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-2">
                  {sortOrder === 'asc' ? <SortAsc className="h-4 w-4" /> : <SortDesc className="h-4 w-4" />}
                  <span className="hidden sm:inline">Sort by</span> {sortBy}
                  <ChevronDown className="h-3 w-3" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setSortBy('name')}>
                  <FileType className="h-4 w-4 mr-2" />
                  Name
                  {sortBy === 'name' && <span className="ml-auto">✓</span>}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSortBy('size')}>
                  <HardDrive className="h-4 w-4 mr-2" />
                  Size
                  {sortBy === 'size' && <span className="ml-auto">✓</span>}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSortBy('modified')}>
                  <Clock className="h-4 w-4 mr-2" />
                  Modified
                  {sortBy === 'modified' && <span className="ml-auto">✓</span>}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={toggleSortOrder}>
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
        </>
      )}
    </div>
  )
}
