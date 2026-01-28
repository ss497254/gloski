import { Button } from '@/ui/button'
import { Plus, Server as ServerIcon } from 'lucide-react'
import { Link } from 'react-router-dom'
import { ServerCard } from '@/features/servers'
import { DashboardProvider, useDashboard } from '../context'
import { PageLayout } from '@/layouts'

function DashboardContent() {
  const { servers, displayServers, onlineCount, offlineCount } = useDashboard()

  return (
    <PageLayout
      title='Dashboard'
      description={
        <p className="text-sm text-muted-foreground">
          {servers.length === 0 ? (
            'No servers configured'
          ) : (
            <span className="flex items-center gap-3">
              <span>
                {servers.length} server{servers.length !== 1 ? 's' : ''}
              </span>
              {onlineCount > 0 && (
                <span className="flex items-center gap-1 text-emerald-600">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                  {onlineCount} online
                </span>
              )}
              {offlineCount > 0 && (
                <span className="flex items-center gap-1 text-red-600">
                  <span className="h-1.5 w-1.5 rounded-full bg-red-500" />
                  {offlineCount} offline
                </span>
              )}
            </span>
          )}
        </p>
      }
      actions={
        <Button asChild>
          <Link to="/servers/add">
            <Plus className="h-4 w-4 mr-2" />
            Add Server
          </Link>
        </Button>
      }
    >

      {/* Content */}
      <div className="p-6">
        {servers.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="h-20 w-20 rounded-2xl bg-primary/10 flex items-center justify-center mb-6">
              <ServerIcon className="h-10 w-10 text-primary" />
            </div>
            <h2 className="text-xl font-semibold mb-2">No servers configured</h2>
            <p className="text-muted-foreground mb-6 max-w-sm">
              Add your first server to start monitoring system resources, managing files, and running commands.
            </p>
            <Button asChild size="lg">
              <Link to="/servers/add">
                <Plus className="h-4 w-4 mr-2" />
                Add Your First Server
              </Link>
            </Button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {displayServers.map((server) => (
              <ServerCard key={server.id} {...server} />
            ))}
          </div>
        )}
      </div>
    </PageLayout>
  )
}

export function DashboardPage() {
  return (
    <DashboardProvider>
      <DashboardContent />
    </DashboardProvider>
  )
}
