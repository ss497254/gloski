import { useServer } from '@/features/servers'
import { EmptyState, FilesListSkeleton } from '@/shared/components'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import { AlertCircle, Folder, Search } from 'lucide-react'
import {
  BulkDeleteDialog,
  DeleteDialog,
  DropZone,
  FileEntryItem,
  FilePreview,
  FilesSidebar,
  FilesToolbar,
  NewFileDialog,
  NewFolderDialog,
  RenameDialog,
  ResizablePreview,
} from '../components'
import { FilesProvider, useFiles } from '../context'
import { useFilesKeyboard } from '../hooks/useFilesKeyboard'

/**
 * Inner component that consumes the FilesContext.
 * Handles the layout and rendering of the files feature.
 */
function FilesPageContent() {
  const { server } = useServer()

  const {
    // Navigation
    currentPath,
    navigateTo,
    refresh,

    // Data
    loading,
    error,
    homeDir,
    sortedAndFilteredEntries,

    // Preview
    selectedFile,

    // UI state
    searchQuery,
    focusedIndex,

    // Store values
    selectedPaths,
    viewMode,
    sidebarOpen,

    // Store actions
    setSidebarOpen,
    isSelected,
    isPinned,

    // Handlers
    handleEntrySelect,
    handleEdit,
    handleDownload,
    setDeleteDialog,
    handleRenameEntry,
    copyPath,
    handleToggleCheck,
    handleTogglePinEntry,
    handleSidebarUnpin,
    handleDropUpload,

    // Upload
    uploading,

    // Dialogs
    newFolderDialog,
    setNewFolderDialog,
    newFileDialog,
    setNewFileDialog,
    deleteDialog,
    renameDialog,
    setRenameDialog,
    bulkDeleteDialog,
    setBulkDeleteDialog,
    newName,
    setNewName,

    // Dialog handlers
    handleCreateFolder,
    handleCreateFile,
    handleDelete,
    handleRename,
    handleBulkDelete,
  } = useFiles()

  // Keyboard navigation
  useFilesKeyboard()

  return (
    <div className="h-full flex">
      {/* Sidebar */}
      <FilesSidebar
        currentPath={currentPath}
        homeDir={homeDir}
        onNavigate={navigateTo}
        onUnpin={handleSidebarUnpin}
        open={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
      />

      {/* Main Content */}
      <DropZone onDrop={handleDropUpload} disabled={uploading} className="flex-1 flex flex-col overflow-hidden">
        {/* Toolbar */}
        <FilesToolbar />

        {/* Content */}
        <div className="flex-1 flex overflow-hidden">
          {/* File list */}
          <div className={cn('overflow-auto flex-1', selectedFile && 'border-r')}>
            {error ? (
              <EmptyState
                icon={AlertCircle}
                title="Error loading directory"
                description={error}
                action={<Button onClick={refresh}>Try Again</Button>}
                className="h-full"
              />
            ) : loading ? (
              <FilesListSkeleton />
            ) : sortedAndFilteredEntries.length === 0 ? (
              <EmptyState
                icon={searchQuery ? Search : Folder}
                title={searchQuery ? 'No files match your search' : 'Empty directory'}
                description={searchQuery ? 'Try a different search term' : 'Create a new file or folder to get started'}
                className="h-full"
              />
            ) : (
              <div
                className={cn(
                  'p-3',
                  viewMode === 'grid' && 'grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-2'
                )}
              >
                {sortedAndFilteredEntries.map((entry, index) => (
                  <FileEntryItem
                    key={entry.path}
                    entry={entry}
                    viewMode={viewMode}
                    isSelected={selectedFile?.path === entry.path}
                    isChecked={isSelected(entry.path)}
                    isFocused={index === focusedIndex}
                    showCheckbox={selectedPaths.size > 0}
                    isPinned={entry.type === 'directory' ? isPinned(entry.path) : undefined}
                    vsCodeUrl={server.url + '/vscode'}
                    onSelect={handleEntrySelect}
                    onNavigate={navigateTo}
                    onDownload={handleDownload}
                    onDelete={setDeleteDialog}
                    onRename={handleRenameEntry}
                    onCopyPath={copyPath}
                    onEdit={handleEdit}
                    onToggleCheck={handleToggleCheck}
                    onTogglePin={entry.type === 'directory' ? handleTogglePinEntry : undefined}
                  />
                ))}
              </div>
            )}
          </div>

          {/* File preview/editor with resize handle */}
          {selectedFile && (
            <ResizablePreview defaultWidth={40} minWidth={30} maxWidth={70}>
              <FilePreview />
            </ResizablePreview>
          )}
        </div>

        {/* Status bar */}
        <div className="border-t px-4 py-2 text-xs text-muted-foreground flex items-center gap-4 bg-muted/30">
          <span>{sortedAndFilteredEntries.length} items</span>
          {selectedPaths.size > 0 && <span className="text-primary">{selectedPaths.size} selected</span>}
          <span className="flex-1" />
          <span className="truncate max-w-md hidden sm:block">{currentPath}</span>
        </div>
      </DropZone>

      {/* Dialogs */}
      <NewFolderDialog
        open={newFolderDialog}
        name={newName}
        onOpenChange={setNewFolderDialog}
        onNameChange={setNewName}
        onCreate={handleCreateFolder}
      />

      <NewFileDialog
        open={newFileDialog}
        name={newName}
        onOpenChange={setNewFileDialog}
        onNameChange={setNewName}
        onCreate={handleCreateFile}
      />

      <DeleteDialog entry={deleteDialog} onClose={() => setDeleteDialog(null)} onDelete={handleDelete} />

      <RenameDialog
        entry={renameDialog}
        name={newName}
        onClose={() => {
          setRenameDialog(null)
          setNewName('')
        }}
        onNameChange={setNewName}
        onRename={handleRename}
      />

      <BulkDeleteDialog
        count={selectedPaths.size}
        open={bulkDeleteDialog}
        onClose={() => setBulkDeleteDialog(false)}
        onDelete={handleBulkDelete}
      />
    </div>
  )
}

/**
 * FilesPage wraps the content with FilesProvider.
 * This ensures all state management is handled by context.
 */
export function FilesPage() {
  return (
    <FilesProvider>
      <FilesPageContent />
    </FilesProvider>
  )
}
