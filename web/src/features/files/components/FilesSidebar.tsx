import { Home, FolderOpen, Pin, PinOff, X, FolderRoot, FolderDown } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import { ScrollArea } from '@/ui/scroll-area'
import { useFilesStore, type PinnedFolder } from '../stores/files'

interface FilesSidebarProps {
  currentPath: string
  homeDir?: string
  onNavigate: (path: string) => void
  onUnpin: (id: string) => void
  open: boolean
  onClose: () => void
  className?: string
}

export function FilesSidebar({
  currentPath,
  homeDir = '/home',
  onNavigate,
  onUnpin,
  open,
  onClose,
  className,
}: FilesSidebarProps) {
  const { pinnedFolders } = useFilesStore()
  const downloadDir = homeDir + '/Downloads'

  return (
    <>
      {/* Mobile overlay */}
      {open && <div className="fixed inset-0 bg-black/50 z-40 md:hidden" onClick={onClose} />}

      {/* Sidebar */}
      <aside
        className={cn(
          'bg-background border-r flex flex-col z-50',
          // Mobile: slide-in drawer
          'fixed inset-y-0 left-0 w-64 transform transition-transform duration-200 ease-in-out md:relative md:translate-x-0',
          open ? 'translate-x-0' : '-translate-x-full',
          // Desktop: always visible when open
          'md:w-56 md:shrink-0',
          className
        )}
      >
        {/* Header (mobile only) */}
        <div className="flex items-center justify-between p-3 border-b md:hidden">
          <span className="font-semibold text-sm">Quick Access</span>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <ScrollArea className="flex-1">
          <div className="p-2">
            {/* Quick Access Section */}
            <div className="mb-4">
              <h3 className="px-2 py-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Quick Access
              </h3>

              <NavItem
                icon={FolderRoot}
                label="Root"
                path="/"
                currentPath={currentPath}
                onClick={() => {
                  onNavigate('/')
                  onClose()
                }}
              />

              <NavItem
                icon={Home}
                label="Home"
                path={homeDir}
                currentPath={currentPath}
                onClick={() => {
                  onNavigate(homeDir)
                  onClose()
                }}
              />

              <NavItem
                icon={FolderDown}
                label="Downloads"
                path={downloadDir}
                currentPath={currentPath}
                onClick={() => {
                  onNavigate(downloadDir)
                  onClose()
                }}
              />
            </div>

            {/* Pinned Folders Section */}
            {pinnedFolders.length > 0 && (
              <div>
                <h3 className="px-2 py-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1">
                  <Pin className="h-3 w-3" />
                  Pinned
                </h3>

                {pinnedFolders.map((folder) => (
                  <PinnedFolderItem
                    key={folder.id}
                    folder={folder}
                    currentPath={currentPath}
                    onClick={() => {
                      onNavigate(folder.path)
                      onClose()
                    }}
                    onUnpin={() => onUnpin(folder.id)}
                  />
                ))}
              </div>
            )}

            {pinnedFolders.length === 0 && (
              <div className="px-2 py-4 text-center">
                <p className="text-xs text-muted-foreground">No pinned folders yet.</p>
                <p className="text-xs text-muted-foreground mt-1">Use the pin button in the toolbar to pin folders.</p>
              </div>
            )}
          </div>
        </ScrollArea>
      </aside>
    </>
  )
}

interface NavItemProps {
  icon: React.ComponentType<{ className?: string }>
  label: string
  path: string
  currentPath: string
  onClick: () => void
}

function NavItem({ icon: Icon, label, path, currentPath, onClick }: NavItemProps) {
  const isActive = currentPath === path

  return (
    <button
      onClick={onClick}
      className={cn(
        'w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-md transition-colors',
        'hover:bg-accent hover:text-accent-foreground',
        isActive && 'bg-accent text-accent-foreground font-medium'
      )}
    >
      <Icon className="h-4 w-4 shrink-0" />
      <span className="truncate">{label}</span>
    </button>
  )
}

interface PinnedFolderItemProps {
  folder: PinnedFolder
  currentPath: string
  onClick: () => void
  onUnpin: () => void
}

function PinnedFolderItem({ folder, currentPath, onClick, onUnpin }: PinnedFolderItemProps) {
  const isActive = currentPath === folder.path

  return (
    <div
      className={cn(
        'group flex items-center gap-2 px-2 py-1.5 text-sm rounded-md transition-colors',
        'hover:bg-accent hover:text-accent-foreground',
        isActive && 'bg-accent text-accent-foreground font-medium'
      )}
    >
      <button onClick={onClick} className="flex-1 flex items-center gap-2 text-left min-w-0">
        <FolderOpen className="h-4 w-4 shrink-0" />
        <span className="truncate">{folder.name}</span>
      </button>

      <button
        onClick={(e) => {
          e.stopPropagation()
          onUnpin()
        }}
        className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-muted transition-opacity"
        title="Unpin folder"
      >
        <PinOff className="h-3 w-3" />
      </button>
    </div>
  )
}
