import { cn } from '@/shared/lib/utils'
import { Button } from '@/ui/button'
import { Dialog, DialogContent, DialogTitle } from '@/ui/dialog'
import { ScrollArea } from '@/ui/scroll-area'
import { FolderDown, FolderOpen, FolderRoot, Home, Pin, PinOff, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { type PinnedFolder, useFilesStore } from '../stores/files'

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
  const pinnedFolders = useFilesStore((state) => state.pinnedFolders)
  const [isMobile, setIsMobile] = useState(() => window.matchMedia('(max-width: 767px)').matches)
  const downloadDir = homeDir + '/Downloads'

  useEffect(() => {
    const media = window.matchMedia('(max-width: 767px)')
    const update = () => setIsMobile(media.matches)
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])

  return (
    <>
      <aside className={cn('hidden shrink-0 flex-col border-r bg-background md:flex md:w-56', className)}>
        <SidebarItems
          currentPath={currentPath}
          homeDir={homeDir}
          downloadDir={downloadDir}
          pinnedFolders={pinnedFolders}
          onNavigate={onNavigate}
          onUnpin={onUnpin}
          onClose={onClose}
        />
      </aside>

      <Dialog open={open && isMobile} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
        <DialogContent
          showCloseButton={false}
          aria-describedby={undefined}
          className="fixed inset-y-0 left-0 top-0 flex h-dvh w-64 max-w-[85vw] translate-x-0 translate-y-0 flex-col gap-0 rounded-none border-y-0 border-l-0 p-0 md:hidden"
        >
          <div className="flex items-center justify-between border-b p-3">
            <DialogTitle className="text-sm">Quick Access</DialogTitle>
            <Button variant="ghost" size="icon" onClick={onClose} aria-label="Close file navigation">
              <X className="h-4 w-4" />
            </Button>
          </div>
          <SidebarItems
            currentPath={currentPath}
            homeDir={homeDir}
            downloadDir={downloadDir}
            pinnedFolders={pinnedFolders}
            onNavigate={onNavigate}
            onUnpin={onUnpin}
            onClose={onClose}
          />
        </DialogContent>
      </Dialog>
    </>
  )
}

function SidebarItems({
  currentPath,
  homeDir,
  downloadDir,
  pinnedFolders,
  onNavigate,
  onUnpin,
  onClose,
}: {
  currentPath: string
  homeDir: string
  downloadDir: string
  pinnedFolders: PinnedFolder[]
  onNavigate: (path: string) => void
  onUnpin: (id: string) => void
  onClose: () => void
}) {
  const navigateAndClose = (path: string) => {
    onNavigate(path)
    onClose()
  }

  return (
    <ScrollArea className="min-h-0 flex-1">
      <div className="p-2">
        <div className="mb-4">
          <h3 className="px-2 py-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Quick Access
          </h3>
          <NavItem icon={FolderRoot} label="Root" path="/" currentPath={currentPath} onClick={() => navigateAndClose('/')} />
          <NavItem
            icon={Home}
            label="Home"
            path={homeDir}
            currentPath={currentPath}
            onClick={() => navigateAndClose(homeDir)}
          />
          <NavItem
            icon={FolderDown}
            label="Downloads"
            path={downloadDir}
            currentPath={currentPath}
            onClick={() => navigateAndClose(downloadDir)}
          />
        </div>

        {pinnedFolders.length > 0 ? (
          <div>
            <h3 className="flex items-center gap-1 px-2 py-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              <Pin className="h-3 w-3" />
              Pinned
            </h3>
            {pinnedFolders.map((folder) => (
              <PinnedFolderItem
                key={folder.id}
                folder={folder}
                currentPath={currentPath}
                onClick={() => navigateAndClose(folder.path)}
                onUnpin={() => onUnpin(folder.id)}
              />
            ))}
          </div>
        ) : (
          <div className="px-2 py-4 text-center">
            <p className="text-xs text-muted-foreground">No pinned folders yet.</p>
            <p className="mt-1 text-xs text-muted-foreground">Use the pin button in the toolbar to pin folders.</p>
          </div>
        )}
      </div>
    </ScrollArea>
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
        className="rounded p-1 opacity-0 transition-opacity hover:bg-muted focus-visible:opacity-100 group-hover:opacity-100"
        aria-label={`Unpin ${folder.name}`}
      >
        <PinOff className="h-3 w-3" />
      </button>
    </div>
  )
}
