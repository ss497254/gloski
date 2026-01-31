import { Button } from '@/ui/button'
import { Card, CardContent } from '@/ui/card'
import { Activity, Plus, Server as ServerIcon, Wifi } from 'lucide-react'
import { Link } from 'react-router-dom'
import { ServerCard } from '@/features/servers'
import { DashboardProvider, useDashboard } from '../context'
import { PageLayout } from '@/layouts'
import { DashboardSkeleton } from '@/shared/components'

function DashboardContent() {
  const { servers, displayServers, onlineCount, offlineCount, isLoading } = useDashboard()

  return (
    <PageLayout
      title='Server Management'
      description="Monitor and manage your Linux servers from a unified dashboard"
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
      {isLoading ? (
        <DashboardSkeleton />
      ) : servers.length === 0 ? (
        <div className="p-6">
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="h-20 w-20 rounded-2xl bg-primary/10 flex items-center justify-center mb-6">
              <ServerIcon className="h-10 w-10 text-primary" />
            </div>
            <h2 className="text-xl font-semibold mb-2">No servers configured</h2>
            <p className="text-muted-foreground mb-6 max-w-sm">
              Add your first server to start monitoring system resources, managing files, and running commands remotely.
            </p>
            <Button asChild size="lg">
              <Link to="/servers/add">
                <Plus className="h-4 w-4 mr-2" />
                Add Your First Server
              </Link>
            </Button>
          </div>
        </div>
      ) : (
        <div className="p-6 space-y-6">
          {/* Stats Overview */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Card>
                <CardContent className="p-6">
                  <div className="flex items-center gap-4">
                    <div className="h-12 w-12 rounded-lg bg-primary/10 flex items-center justify-center">
                      <ServerIcon className="h-6 w-6 text-primary" />
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Total Servers</p>
                      <p className="text-2xl font-semibold">{servers.length}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardContent className="p-6">
                  <div className="flex items-center gap-4">
                    <div className="h-12 w-12 rounded-lg bg-emerald-500/10 flex items-center justify-center">
                      <Wifi className="h-6 w-6 text-emerald-600" />
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Online</p>
                      <p className="text-2xl font-semibold text-emerald-600">{onlineCount}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardContent className="p-6">
                  <div className="flex items-center gap-4">
                    <div className="h-12 w-12 rounded-lg bg-red-500/10 flex items-center justify-center">
                      <Activity className="h-6 w-6 text-red-600" />
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Offline</p>
                      <p className="text-2xl font-semibold text-red-600">{offlineCount}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Servers Grid */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold">Your Servers</h3>
                <Button asChild variant="outline" size="sm">
                  <Link to="/servers">View All</Link>
                </Button>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {displayServers.map((server) => (
                  <ServerCard key={server.id} {...server} />
                ))}
              </div>
            </div>
        </div>
      )}
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
