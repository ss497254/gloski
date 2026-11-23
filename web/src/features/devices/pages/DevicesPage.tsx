import { PageLayout } from '@/layouts'
import { cn, formatBytes, formatRelativeTime, formatUptime } from '@/shared/lib/utils'
import { useTower } from '@/shared/tower'
import type { FleetDevice } from '@/shared/tower/types'
import { Button } from '@/ui/button'
import { DeployDeviceDialog, EmptyState, UsageBar } from '@/shared/components'
import { Skeleton } from '@/ui/skeleton'
import {
  Activity,
  ArrowUpRight,
  Circle,
  Play,
  Plus,
  Radar,
  RefreshCw,
  RotateCw,
  Square,
  Trash2,
  Wifi,
} from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { devicePath } from '@/app/navigation'
import { DiagnoseButton } from '../components/DiagnoseDialog'
import { useDeviceActions } from '../hooks/useDeviceActions'

function DeviceRow({ device }: { device: FleetDevice }) {
  const navigate = useNavigate()
  const { run, busy } = useDeviceActions(device)
  const online = device.status === 'online'
  const stats = device.stats
  // Why the master could not get this device's agent running, if it tried.
  const problem = !online && device.deployment?.status === 'failed' ? device.deployment.error : undefined
  const hostKeyChanged = problem?.includes('host key changed') ?? false
  const stopped = device.deployment?.status === 'stopped'

  return (
    <tr
      className="cursor-pointer border-b border-border/70 transition-colors last:border-0 hover:bg-muted/40"
      onClick={() => navigate(devicePath(device.id))}
    >
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          <span className={cn('h-2 w-2 shrink-0 rounded-full', online ? 'bg-success' : 'bg-muted-foreground/40')} />
          <div className="min-w-0">
            <p className="flex items-center gap-2 truncate text-sm font-medium text-foreground">
              {device.name}
              {device.master && (
                <span className="rounded border border-primary/30 bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wider text-foreground">
                  Master
                </span>
              )}
            </p>
            <p className="truncate text-xs text-muted-foreground">{device.hostname || device.url}</p>
            {problem && (
              <p
                className="mt-1 max-w-md whitespace-normal text-xs text-destructive"
                onClick={(event) => event.stopPropagation()}
              >
                {problem}
                {hostKeyChanged && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="ml-2 h-6 px-2 text-xs"
                    disabled={busy}
                    onClick={() => void run('trust')}
                  >
                    Trust new key
                  </Button>
                )}
              </p>
            )}
          </div>
        </div>
      </td>
      <td className="px-4 py-3">
        <span
          className={cn(
            'inline-flex items-center gap-1.5 text-xs font-medium',
            online ? 'text-success' : 'text-muted-foreground'
          )}
        >
          <Circle className="h-2 w-2 fill-current" />
          {online ? 'Online' : stopped ? 'Stopped' : 'Offline'}
        </span>
      </td>
      <td className="px-4 py-3">
        <div className="space-y-1.5">
          <UsageBar label="cpu" value={stats?.cpu.usage_percent ?? 0} />
          <UsageBar label="mem" value={stats?.memory.used_percent ?? 0} />
        </div>
      </td>
      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
        <div className="text-success">↓ {formatBytes(stats?.network.rx_rate ?? 0)}/s</div>
        <div className="text-info">↑ {formatBytes(stats?.network.tx_rate ?? 0)}/s</div>
      </td>
      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{stats ? formatUptime(stats.uptime) : '—'}</td>
      <td className="px-4 py-3 text-xs text-muted-foreground">{formatRelativeTime(device.last_seen)}</td>
      <td className="px-4 py-3">
        <div className="flex items-center justify-end gap-1" onClick={(event) => event.stopPropagation()}>
          <DiagnoseButton deviceId={device.id} deviceName={device.name} iconOnly variant="ghost" size="icon-sm" />
          {device.managed && online && (
            <Button
              variant="ghost"
              size="icon-sm"
              title="Restart agent"
              disabled={busy}
              onClick={() => void run('restart')}
            >
              <RotateCw className="h-3.5 w-3.5" />
            </Button>
          )}
          {(device.managed || (online && !device.master)) && (
            <Button
              variant="ghost"
              size="icon-sm"
              title={online ? 'Stop agent' : 'Install and start agent'}
              disabled={busy}
              onClick={() => void run(online ? 'stop' : 'start')}
            >
              {busy ? (
                <RefreshCw className="h-4 w-4 animate-spin" />
              ) : online ? (
                <Square className="h-3.5 w-3.5" />
              ) : (
                <Play className="h-3.5 w-3.5" />
              )}
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon-sm"
            title="Remove device"
            disabled={busy}
            onClick={() => void run('remove')}
          >
            <Trash2 className="h-3.5 w-3.5 text-destructive" />
          </Button>
          <Button variant="ghost" size="icon-sm" title="Open device" onClick={() => navigate(devicePath(device.id))}>
            <ArrowUpRight className="h-4 w-4" />
          </Button>
        </div>
      </td>
    </tr>
  )
}

export function DevicesPage() {
  const { mode, info, devices, connected, error, refresh } = useTower()
  const [deployOpen, setDeployOpen] = useState(false)
  const [refreshing, setRefreshing] = useState(false)

  if (mode === 'loading') {
    return (
      <PageLayout title="Devices" description="Connecting to the control tower…">
        <div className="space-y-2 rounded-lg border border-border bg-card p-4">
          {[0, 1, 2].map((index) => (
            <Skeleton key={index} className="h-12 w-full" />
          ))}
        </div>
      </PageLayout>
    )
  }

  if (mode === 'standalone') {
    return (
      <PageLayout title="Devices" description="Waiting for the control tower">
        <EmptyState
          icon={Radar}
          title="No control tower detected"
          description="Open the dashboard served by the Gloski master to manage this tailnet's devices."
        />
      </PageLayout>
    )
  }

  const online = devices.filter((device) => device.status === 'online').length
  const deployDisabled = !info?.deploy_available

  return (
    <PageLayout
      title="Devices"
      description={`${online} of ${devices.length} devices online${info?.name ? ` · tower ${info.name}` : ''}`}
      actions={
        <div className="flex items-center gap-2">
          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className={cn('h-2 w-2 rounded-full', connected ? 'bg-success' : 'bg-destructive')} />
            {connected ? 'live' : 'reconnecting'}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={async () => {
              setRefreshing(true)
              try {
                await refresh()
              } finally {
                setRefreshing(false)
              }
            }}
          >
            <RefreshCw className={cn('mr-2 h-4 w-4', refreshing && 'animate-spin')} />
            Refresh
          </Button>
          <Button
            size="sm"
            onClick={() => setDeployOpen(true)}
            disabled={deployDisabled}
            title={deployDisabled ? "The master has no agent builds; run 'make build'" : undefined}
          >
            <Plus className="mr-2 h-4 w-4" />
            Deploy agent
          </Button>
        </div>
      }
    >
      {error && (
        <div className="mb-4 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      {devices.length === 0 ? (
        <div className="rounded-lg border border-border bg-card">
          <EmptyState
            icon={Wifi}
            title="No devices yet"
            description="Deploy gloski-agent to a machine on your tailnet. The master installs it over SSH and keeps it running from then on."
            action={
              <Button onClick={() => setDeployOpen(true)} disabled={deployDisabled}>
                <Plus className="mr-2 h-4 w-4" />
                Deploy your first agent
              </Button>
            }
          />
        </div>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-border bg-card">
          <table className="w-full min-w-[900px] text-left">
            <thead className="border-b border-border bg-muted/40 text-[10px] uppercase tracking-wider text-muted-foreground">
              <tr>
                <th className="px-4 py-2 font-medium">Device</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Usage</th>
                <th className="px-4 py-2 font-medium">Network</th>
                <th className="px-4 py-2 font-medium">Uptime</th>
                <th className="px-4 py-2 font-medium">Seen</th>
                <th className="px-4 py-2" />
              </tr>
            </thead>
            <tbody>
              {devices.map((device) => (
                <DeviceRow key={device.id} device={device} />
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-4 flex items-center gap-4 text-xs text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <Activity className="h-3.5 w-3.5" />
          Heartbeat every {info?.heartbeat_interval ?? 10}s
        </span>
        <span className="flex items-center gap-1.5">
          <Radar className="h-3.5 w-3.5" />
          {info?.tailscale_available ? 'Tailnet connected' : 'Tailscale API unavailable'}
        </span>
      </div>

      <DeployDeviceDialog open={deployOpen} onOpenChange={setDeployOpen} onDeployed={() => void refresh()} />
    </PageLayout>
  )
}
