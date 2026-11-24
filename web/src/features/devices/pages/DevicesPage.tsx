import { devicePath } from '@/app/navigation'
import { PageLayout } from '@/layouts'
import { cn, formatBytes, formatRelativeTime, formatUptime } from '@/shared/lib/utils'
import { useTower } from '@/shared/tower'
import type { FleetDevice } from '@/shared/tower/types'
import { Alert, AlertDescription } from '@/ui/alert'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Card } from '@/ui/card'
import { Skeleton } from '@/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/ui/table'
import { DeployDeviceDialog, EmptyState, StatusBadge, StatusDot, TableControls, UsageBar } from '@/shared/components'
import { Activity, ArrowUpRight, CircleAlert, Play, Plus, Radar, RefreshCw, RotateCw, Square, Trash2, Wifi } from 'lucide-react'
import { useMemo, useState } from 'react'
import { paginate, sortDevices, type DeviceSort } from '../lib/table'
import { useNavigate } from 'react-router-dom'
import { DiagnoseButton } from '../components/DiagnoseDialog'
import { useDeviceActions } from '../hooks/useDeviceActions'

function DeviceRow({ device, hidden }: { device: FleetDevice; hidden: string[] }) {
  const navigate = useNavigate()
  const show = (field: string) => !hidden.includes(field)
  const { run, busy } = useDeviceActions(device)
  const online = device.status === 'online'
  const stats = device.stats
  // Why the master could not get this device's agent running, if it tried.
  const problem = !online && device.deployment?.status === 'failed' ? device.deployment.error : undefined
  const hostKeyChanged = problem?.includes('host key changed') ?? false
  const stopped = device.deployment?.status === 'stopped'
  const status = online ? 'online' : stopped ? 'stopped' : 'offline'

  return (
    <TableRow className="cursor-pointer" onClick={() => navigate(devicePath(device.id))}>
      <TableCell className="px-4 py-3">
        <div className="flex items-center gap-3">
          <StatusDot status={status} />
          <div className="min-w-0">
            <p className="flex items-center gap-2 truncate text-sm font-medium text-foreground">
              {device.name}
              {device.master && (
                <Badge
                  variant="outline"
                  className="border-primary/30 bg-primary/10 px-1.5 py-0 text-[10px] uppercase tracking-wider text-primary"
                >
                  Master
                </Badge>
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
            {!online && !problem && (
              <p className="mt-1 max-w-md whitespace-normal text-xs text-muted-foreground">
                No live agent session
                {device.last_seen ? ` · last seen ${formatRelativeTime(device.last_seen)}` : ''}. Diagnose to find
                where the link breaks.
              </p>
            )}
          </div>
        </div>
      </TableCell>
      {show('status') && (
        <TableCell className="px-4 py-3">
          <StatusBadge status={status} label={online ? 'Online' : stopped ? 'Stopped' : 'Offline'} />
        </TableCell>
      )}
      {show('usage') && (
        <TableCell className="px-4 py-3">
          <div className="space-y-1.5">
            <UsageBar label="cpu" value={stats?.cpu.usage_percent ?? 0} />
            <UsageBar label="mem" value={stats?.memory.used_percent ?? 0} />
          </div>
        </TableCell>
      )}
      {show('network') && (
        <TableCell className="px-4 py-3 font-mono text-xs text-muted-foreground">
          <div className="text-success">↓ {formatBytes(stats?.network.rx_rate ?? 0)}/s</div>
          <div className="text-info">↑ {formatBytes(stats?.network.tx_rate ?? 0)}/s</div>
        </TableCell>
      )}
      {show('uptime') && (
        <TableCell className="px-4 py-3 font-mono text-xs text-muted-foreground">
          {stats ? formatUptime(stats.uptime) : '—'}
        </TableCell>
      )}
      {show('seen') && (
        <TableCell className="px-4 py-3 text-xs text-muted-foreground">{formatRelativeTime(device.last_seen)}</TableCell>
      )}
      <TableCell className="px-4 py-3">
        <div className="flex items-center justify-end gap-1" onClick={(event) => event.stopPropagation()}>
          <DiagnoseButton deviceId={device.id} deviceName={device.name} variant="ghost" size="sm" />
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
          <Button variant="ghost" size="icon-sm" title="Remove device" disabled={busy} onClick={() => void run('remove')}>
            <Trash2 className="h-3.5 w-3.5 text-destructive" />
          </Button>
          <Button variant="ghost" size="icon-sm" title="Open device" onClick={() => navigate(devicePath(device.id))}>
            <ArrowUpRight className="h-4 w-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  )
}

export function DevicesPage() {
  const { mode, info, devices, connected, error, refresh } = useTower()
  const [deployOpen, setDeployOpen] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [sort, setSort] = useState('name')
  const [hidden, setHidden] = useState<string[]>([])
  const [pageSize, setPageSize] = useState(25)
  const [page, setPage] = useState(0)

  const sorted = useMemo(() => sortDevices(devices, sort as DeviceSort), [devices, sort])
  const { visible, pageCount, currentPage, first, last } = useMemo(
    () => paginate(sorted, page, pageSize),
    [sorted, page, pageSize],
  )

  const toggleField = (field: string) =>
    setHidden((current) => (current.includes(field) ? current.filter((item) => item !== field) : [...current, field]))

  if (mode === 'loading') {
    return (
      <PageLayout title="Devices" description="Connecting to the control tower…">
        <Card className="gap-0 py-0 p-4">
          {[0, 1, 2].map((index) => (
            <Skeleton key={index} className="mb-2 h-12 w-full last:mb-0" />
          ))}
        </Card>
      </PageLayout>
    )
  }

  if (mode === 'standalone') {
    return (
      <PageLayout title="Devices" description="Waiting for the control tower">
        <Card className="gap-0 py-0">
          <EmptyState
            icon={Radar}
            title="No control tower detected"
            description="Open the dashboard served by the Gloski master to manage this tailnet's devices."
          />
        </Card>
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
          <TableControls
            sort={sort}
            sortOptions={[
              { value: 'name', label: 'Name' },
              { value: 'status', label: 'Status' },
              { value: 'uptime', label: 'Uptime' },
              { value: 'seen', label: 'Last seen' },
            ]}
            onSortChange={setSort}
            visibleFields={['status', 'usage', 'network', 'uptime', 'seen'].filter((field) => !hidden.includes(field))}
            fieldOptions={[
              { value: 'status', label: 'Status' },
              { value: 'usage', label: 'Usage' },
              { value: 'network', label: 'Network' },
              { value: 'uptime', label: 'Uptime' },
              { value: 'seen', label: 'Seen' },
            ]}
            onToggleField={toggleField}
            pageSize={pageSize}
            onPageSizeChange={(size) => {
              setPageSize(size)
              setPage(0)
            }}
          />
          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <StatusDot status={connected ? 'online' : 'offline'} />
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
          <Button size="sm" onClick={() => setDeployOpen(true)} disabled={deployDisabled} title={deployDisabled ? "The master has no agent builds; run 'make build'" : undefined}>
            <Plus className="mr-2 h-4 w-4" />
            Deploy agent
          </Button>
        </div>
      }
    >
      {error && (
        <Alert variant="destructive" className="mb-4">
          <CircleAlert />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {devices.length === 0 ? (
        <Card className="gap-0 py-0">
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
        </Card>
      ) : (
        <Card className="gap-0 overflow-hidden py-0">
          <div className="max-h-[70vh] overflow-auto">
            <Table className="min-w-[900px]">
              <TableHeader className="sticky top-0 z-10 bg-muted/50 hover:bg-muted/50">
                <TableRow>
                  <TableHead className="px-4 py-2.5 text-[11px] uppercase tracking-wider text-muted-foreground">
                    Device
                  </TableHead>
                  {!hidden.includes('status') && (
                    <TableHead className="px-4 py-2.5 text-[11px] uppercase tracking-wider text-muted-foreground">
                      Status
                    </TableHead>
                  )}
                  {!hidden.includes('usage') && (
                    <TableHead className="px-4 py-2.5 text-[11px] uppercase tracking-wider text-muted-foreground">
                      Usage
                    </TableHead>
                  )}
                  {!hidden.includes('network') && (
                    <TableHead className="px-4 py-2.5 text-[11px] uppercase tracking-wider text-muted-foreground">
                      Network
                    </TableHead>
                  )}
                  {!hidden.includes('uptime') && (
                    <TableHead className="px-4 py-2.5 text-[11px] uppercase tracking-wider text-muted-foreground">
                      Uptime
                    </TableHead>
                  )}
                  {!hidden.includes('seen') && (
                    <TableHead className="px-4 py-2.5 text-[11px] uppercase tracking-wider text-muted-foreground">
                      Seen
                    </TableHead>
                  )}
                  <TableHead className="px-4 py-2.5" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {visible.map((device) => (
                  <DeviceRow key={device.id} device={device} hidden={hidden} />
                ))}
              </TableBody>
            </Table>
          </div>
          <div className="flex items-center justify-between border-t border-border px-4 py-2 text-xs text-muted-foreground">
            <span>
              {first}–{last} of {sorted.length}
            </span>
            <div className="flex gap-2">
              <Button
                variant="ghost"
                size="sm"
                disabled={currentPage === 0}
                onClick={() => setPage(Math.max(0, currentPage - 1))}
              >
                Previous
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={currentPage >= pageCount - 1}
                onClick={() => setPage(Math.min(pageCount - 1, currentPage + 1))}
              >
                Next
              </Button>
            </div>
          </div>
        </Card>
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
