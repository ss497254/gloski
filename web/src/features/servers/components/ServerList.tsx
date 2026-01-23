import { Link, useParams, useNavigate } from 'react-router-dom'
import { Button } from '@/ui/button'
import { useServersStore, getSortedServers, type Server, type ServerStatus } from '../stores/servers'
import { cn } from '@/shared/lib/utils'
import {
  Server as ServerIcon,
  Plus,
  Trash2,
  MoreVertical,
  Wifi,
  WifiOff,
  ShieldAlert,
  ExternalLink,
  Settings2,
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/ui/tooltip'
import { useState } from 'react'

const statusConfig: Record<
  ServerStatus,
  { color: string; pulseColor?: string; label: string; icon: typeof Wifi }
> = {
  online: {
    color: 'bg-emerald-500',
    label: 'Online',
    icon: Wifi,
  },
  connecting: {
    color: 'bg-yellow-500',
    pulseColor: 'bg-yellow-400',
    label: 'Connecting...',
    icon: Wifi,
  },
  unauthorized: {
    color: 'bg-orange-500',
    label: 'Authentication failed',
    icon: ShieldAlert,
  },
  offline: {
    color: 'bg-red-500',
    label: 'Offline',
    icon: WifiOff,
  },
}

interface ServerItemProps {
  server: Server
  isActive: boolean
  onDelete: () => void
}

function ServerItem({ server, isActive, onDelete }: ServerItemProps) {
  const navigate = useNavigate()
  const status = statusConfig[server.status]

  return (
    <div
      className={cn(
        'group relative flex items-center gap-2.5 px-2.5 py-2 rounded-lg transition-all duration-150',
        isActive
          ? 'bg-accent shadow-sm'
          : 'hover:bg-accent/60'
      )}
    >
      <Link
        to={`/servers/${server.id}`}
        className="flex items-center gap-2.5 flex-1 min-w-0"
      >
        {/* Server Icon with Status */}
        <div className="relative flex-shrink-0">
          <div className={cn(
            'h-8 w-8 rounded-md flex items-center justify-center transition-colors',
            isActive ? 'bg-primary/10' : 'bg-muted/50 group-hover:bg-muted'
          )}>
            <ServerIcon className={cn(
              'h-4 w-4 transition-colors',
              isActive ? 'text-primary' : 'text-muted-foreground'
            )} />
          </div>
          {/* Status Indicator */}
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="absolute -bottom-0.5 -right-0.5 block">
                <span
                  className={cn(
                    'block h-2.5 w-2.5 rounded-full border-2 border-card',
                    status.color
                  )}
                />
                {status.pulseColor && (
                  <span
                    className={cn(
                      'absolute inset-0 rounded-full animate-ping',
                      status.pulseColor
                    )}
                    style={{ animationDuration: '1.5s' }}
                  />
                )}
              </span>
            </TooltipTrigger>
            <TooltipContent side="right" sideOffset={8}>
              <div className="flex items-center gap-1.5">
                <status.icon className="h-3 w-3" />
                {status.label}
              </div>
            </TooltipContent>
          </Tooltip>
        </div>

        {/* Server Name */}
        <div className="flex-1 min-w-0">
          <span className={cn(
            'block truncate text-sm font-medium transition-colors',
            isActive ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'
          )}>
            {server.name}
          </span>
        </div>
      </Link>

      {/* Actions Menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className={cn(
              'h-7 w-7 flex-shrink-0 transition-opacity',
              isActive ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'
            )}
          >
            <MoreVertical className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuItem onClick={() => navigate(`/servers/${server.id}`)}>
            <Settings2 className="h-4 w-4 mr-2" />
            View Details
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => window.open(server.url, '_blank')}>
            <ExternalLink className="h-4 w-4 mr-2" />
            Open URL
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className="text-destructive focus:text-destructive focus:bg-destructive/10"
            onClick={onDelete}
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Remove Server
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

export function ServerList() {
  const { serverId } = useParams()
  const { servers, removeServer } = useServersStore()
  const [deleteServer, setDeleteServer] = useState<Server | null>(null)

  const sortedServers = getSortedServers(servers)

  const handleDelete = () => {
    if (deleteServer) {
      removeServer(deleteServer.id)
      setDeleteServer(null)
    }
  }

  const onlineCount = servers.filter((s) => s.status === 'online').length

  return (
    <div className="space-y-1">
      {/* Header */}
      <div className="flex items-center justify-between px-2.5 py-1.5">
        <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Servers
          {servers.length > 0 && (
            <span className="ml-1.5 text-[10px] font-normal normal-case">
              ({onlineCount}/{servers.length})
            </span>
          )}
        </span>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 hover:bg-primary/10 hover:text-primary"
              asChild
            >
              <Link to="/add-server">
                <Plus className="h-3.5 w-3.5" />
              </Link>
            </Button>
          </TooltipTrigger>
          <TooltipContent side="right">Add server</TooltipContent>
        </Tooltip>
      </div>

      {/* Server List */}
      <div className="space-y-0.5">
        {sortedServers.length === 0 ? (
          <div className="px-2.5 py-6 text-center">
            <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center mx-auto mb-2">
              <ServerIcon className="h-5 w-5 text-muted-foreground" />
            </div>
            <p className="text-xs text-muted-foreground mb-2">
              No servers yet
            </p>
            <Button variant="outline" size="sm" asChild>
              <Link to="/add-server">
                <Plus className="h-3 w-3 mr-1.5" />
                Add Server
              </Link>
            </Button>
          </div>
        ) : (
          sortedServers.map((server) => (
            <ServerItem
              key={server.id}
              server={server}
              isActive={server.id === serverId}
              onDelete={() => setDeleteServer(server)}
            />
          ))
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteServer} onOpenChange={() => setDeleteServer(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove Server</DialogTitle>
            <DialogDescription>
              Are you sure you want to remove <strong>"{deleteServer?.name}"</strong>?
              This will only remove it from your list and won't affect the server itself.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteServer(null)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              <Trash2 className="h-4 w-4 mr-2" />
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
