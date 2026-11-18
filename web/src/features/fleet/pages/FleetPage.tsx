import { PageLayout } from '@/layouts'
import { useTower } from '@/shared/tower'
import { Button } from '@/ui/button'
import { Card, CardContent } from '@/ui/card'
import { Skeleton } from '@/ui/skeleton'
import { Activity, Layers, RefreshCw, Server, Terminal, Wifi } from 'lucide-react'
import { Link } from 'react-router-dom'
import { MiniBtop } from '../components/MiniBtop'

export function FleetPage() {
  const { mode, info, devices, connected, error, refresh } = useTower()

  if (mode === 'loading') {
    return (
      <PageLayout title="Fleet" description="Connecting to the control tower…">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {[0, 1, 2].map((index) => (
            <Skeleton key={index} className="h-64 w-full" />
          ))}
        </div>
      </PageLayout>
    )
  }

  if (mode === 'standalone') {
    return (
      <PageLayout title="Fleet" description="This Gloski instance is not running as a control tower">
        <Card>
          <CardContent className="flex flex-col items-start gap-4 p-6">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                <Layers className="h-5 w-5" />
              </div>
              <div>
                <h2 className="font-medium">Standalone mode</h2>
                <p className="text-sm text-muted-foreground">
                  Run the master binary with <code className="rounded bg-muted px-1">--mode master</code> to manage a
                  fleet. The existing server list remains available.
                </p>
              </div>
            </div>
            <Button asChild variant="outline">
              <Link to="/servers">Open servers</Link>
            </Button>
          </CardContent>
        </Card>
      </PageLayout>
    )
  }

  const online = devices.filter((device) => device.status === 'online').length

  return (
    <PageLayout
      title={info?.name ? `${info.name} fleet` : 'Fleet'}
      description="One screen for every device on the tailnet"
      actions={
        <div className="flex items-center gap-2">
          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className={connected ? 'h-2 w-2 rounded-full bg-emerald-500' : 'h-2 w-2 rounded-full bg-red-500'} />
            {connected ? 'live' : 'reconnecting'}
          </span>
          <Button variant="outline" size="sm" onClick={() => refresh()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>
      }
    >
      <div className="mb-5 grid grid-cols-2 gap-3 md:grid-cols-4">
        {[
          { label: 'Devices', value: devices.length, icon: Server, tone: 'text-foreground' },
          { label: 'Online', value: online, icon: Wifi, tone: 'text-emerald-500' },
          { label: 'Offline', value: devices.length - online, icon: Activity, tone: 'text-red-500' },
          {
            label: 'Heartbeat',
            value: info ? `${info.heartbeat_interval}s` : '—',
            icon: Terminal,
            tone: 'text-muted-foreground',
          },
        ].map((stat) => (
          <Card key={stat.label}>
            <CardContent className="flex items-center gap-3 p-4">
              <stat.icon className={`h-4 w-4 ${stat.tone}`} />
              <div>
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground">{stat.label}</p>
                <p className="text-lg font-semibold leading-none">{stat.value}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {error && (
        <div className="mb-4 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      {devices.length === 0 ? (
        <Card>
          <CardContent className="p-6">
            <div className="flex flex-col items-center py-10 text-center">
              <Layers className="mb-4 h-10 w-10 text-muted-foreground" />
              <h2 className="mb-1 text-lg font-semibold">No devices enrolled</h2>
              <p className="mb-6 max-w-md text-sm text-muted-foreground">
                Install <code className="rounded bg-muted px-1">gloski-agent</code> on a device on your tailnet and
                point it at this tower. The agent appears here as soon as it registers.
              </p>
              <pre className="max-w-full overflow-x-auto rounded-md border bg-muted/40 p-3 text-left text-xs">
                {`GLOSKI_MODE=agent \\
GLOSKI_AGENT_MASTER_URL=${window.location.origin} \\
GLOSKI_AGENT_JOIN_TOKEN=$(cat ~/.gloski/data/master-join-token) \\
GLOSKI_AGENT_NAME=my-device \\
GLOSKI_HOST=0.0.0.0 \\
gloski-agent`}
              </pre>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {devices.map((device) => (
            <MiniBtop key={device.id} device={device} />
          ))}
        </div>
      )}
    </PageLayout>
  )
}
