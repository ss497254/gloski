import { useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '@/ui/button'
import { Card, CardContent } from '@/ui/card'
import { PageLayout } from '@/layouts'
import { useServersStore, type Server } from '../stores/servers'
import { checkServerHealth } from '@/shared/services/api'
import { cn } from '@/shared/lib/utils'
import { Server as ServerIcon, Plus, Trash2, ExternalLink, Circle } from 'lucide-react'

const statusColors: Record<string, string> = {
  online: 'bg-green-500',
  offline: 'bg-gray-400',
  connecting: 'bg-yellow-500',
  unauthorized: 'bg-red-500',
}

const statusLabels: Record<string, string> = {
  online: 'Online',
  offline: 'Offline',
  connecting: 'Connecting',
  unauthorized: 'Unauthorized',
}

function ServerCard({ server }: { server: Server }) {
  const { removeServer } = useServersStore()
  const navigate = useNavigate()

  return (
    <Card
      className="cursor-pointer hover:border-primary/50 transition-colors"
      onClick={() => navigate(`/servers/${server.id}`)}
    >
      <CardContent className="p-4">
        <div className="flex items-start gap-4">
          <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center shrink-0">
            <ServerIcon className="h-5 w-5 text-muted-foreground" />
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="font-medium truncate">{server.name}</h3>
              <div className="flex items-center gap-1.5">
                <Circle className={cn('h-2 w-2 fill-current', statusColors[server.status])} />
                <span className="text-xs text-muted-foreground">{statusLabels[server.status]}</span>
              </div>
            </div>
            <p className="text-sm text-muted-foreground truncate">{server.url}</p>
          </div>

          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              onClick={(e) => {
                e.stopPropagation()
                window.open(server.url, '_blank')
              }}
            >
              <ExternalLink className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={(e) => {
                e.stopPropagation()
                if (confirm(`Remove server "${server.name}"?`)) {
                  removeServer(server.id)
                }
              }}
            >
              <Trash2 className="h-4 w-4 text-destructive" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export function ServersPage() {
  const servers = useServersStore((s) => s.servers)
  const updateServer = useServersStore((s) => s.updateServer)

  // Check server health on mount
  useEffect(() => {
    servers.forEach(async (server) => {
      updateServer(server.id, { status: 'connecting' })
      try {
        await checkServerHealth(server.url)
        updateServer(server.id, { status: 'online' })
      } catch {
        updateServer(server.id, { status: 'offline' })
      }
    })
  }, []) // Only run on mount

  return (
    <PageLayout
      title="Servers"
      description="Manage your connected servers"
      actions={
        <Button asChild>
          <Link to="/servers/add">
            <Plus className="h-4 w-4 mr-2" />
            Add Server
          </Link>
        </Button>
      }
    >
      {servers.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-64 text-center">
          <ServerIcon className="h-12 w-12 text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium mb-2">No servers configured</h3>
          <p className="text-muted-foreground mb-4">Add a server to start managing it from here</p>
          <Button asChild>
            <Link to="/servers/add">
              <Plus className="h-4 w-4 mr-2" />
              Add your first server
            </Link>
          </Button>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {servers.map((server) => (
            <ServerCard key={server.id} server={server} />
          ))}
        </div>
      )}
    </PageLayout>
  )
}
