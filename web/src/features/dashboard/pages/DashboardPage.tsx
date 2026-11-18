import { PageLayout } from '@/layouts'
import { cn, formatRelativeTime, formatUptime } from '@/shared/lib/utils'
import { getDashboard } from '@/shared/tower/client'
import type { DashboardSummary, FleetDevice } from '@/shared/tower/types'
import { useTower } from '@/shared/tower'
import { Button } from '@/ui/button'
import { EmptyState } from '@/shared/components'
import { Skeleton } from '@/ui/skeleton'
import {
  Activity,
  AlertTriangle,
  ArrowUpRight,
  Circle,
  Cpu,
  ListTodo,
  Plus,
  RefreshCw,
  Server,
  Wifi,
} from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { DeployDeviceDialog } from '@/features/fleet/components/DeployDeviceDialog'

function Metric({ icon: Icon, label, value, tone }: { icon: typeof Cpu; label: string; value: string; tone?: string }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Icon className={cn('h-4 w-4', tone)} />
        {label}
      </div>
      <p className="mt-2 text-2xl font-semibold tabular-nums text-foreground">{value}</p>
    </div>
  )
}

function UsageBar({ label, value }: { label: string; value: number }) {
  const tone = value > 85 ? 'bg-destructive' : value > 65 ? 'bg-warning' : 'bg-success'
  return (
    <div className="flex items-center gap-2">
      <span className="w-8 text-[10px] uppercase tracking-wide text-muted-foreground">{label}</span>
      <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
        <div className={cn('h-full rounded-full', tone)} style={{ width: `${Math.min(100, Math.max(0, value))}%` }} />
      </div>
      <span className="w-9 text-right font-mono text-xs tabular-nums text-foreground">{value.toFixed(0)}%</span>
    </div>
  )
}

function DeviceCard({ device }: { device: FleetDevice }) {
  const online = device.status === 'online'
  const stats = device.stats
  return (
    <Link
      to={`/fleet/${device.id}`}
      className="group rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary/50"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn('h-2 w-2 shrink-0 rounded-full', online ? 'bg-success' : 'bg-muted-foreground/40')} />
            <h3 className="truncate font-medium text-foreground">{device.name}</h3>
            {device.master && (
              <span className="rounded border border-primary/30 bg-primary/10 px-1.5 py-0.5 text-[10px] uppercase tracking-wider">
                Master
              </span>
            )}
          </div>
          <p className="mt-1 truncate text-xs text-muted-foreground">
            {device.hostname || 'device'} · {online ? 'online' : 'offline'}
          </p>
        </div>
        <ArrowUpRight className="h-4 w-4 shrink-0 text-muted-foreground transition-colors group-hover:text-foreground" />
      </div>

      <div className="mt-4 space-y-2">
        <UsageBar label="cpu" value={stats?.cpu.usage_percent ?? 0} />
        <UsageBar label="mem" value={stats?.memory.used_percent ?? 0} />
      </div>

      <div className="mt-4 flex items-center justify-between border-t border-border pt-3 text-[11px] text-muted-foreground">
        <span className="font-mono">
          ↓ {formatBytesShort(stats?.network.rx_rate ?? 0)}/s · ↑ {formatBytesShort(stats?.network.tx_rate ?? 0)}/s
        </span>
        <span>{stats ? formatUptime(stats.uptime) : formatRelativeTime(device.last_seen)}</span>
      </div>
    </Link>
  )
}

function formatBytesShort(bytes: number): string {
  if (bytes < 1024) return `${Math.round(bytes)} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

export function DashboardPage() {
  const { mode, devices, info } = useTower()
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [deployOpen, setDeployOpen] = useState(false)

  const load = useCallback(async () => {
    try {
      setSummary(await getDashboard())
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load dashboard')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (mode !== 'tower') return
    void load()
    const timer = setInterval(() => void load(), 10000)
    return () => clearInterval(timer)
  }, [mode, load])

  if (mode === 'loading') {
    return (
      <PageLayout title="Dashboard" description="Connecting to the control tower…">
        <div className="grid gap-4 md:grid-cols-4">
          {[0, 1, 2, 3].map((index) => (
            <Skeleton key={index} className="h-24 w-full" />
          ))}
        </div>
      </PageLayout>
    )
  }

  if (mode === 'standalone') {
    return (
      <PageLayout title="Dashboard" description="No control tower detected">
        <EmptyState
          icon={Server}
          title="Master not available"
          description="Open the dashboard served by the Gloski master to manage devices on this tailnet."
        />
      </PageLayout>
    )
  }

  const issues = summary?.issues ?? []
  const online = summary?.online ?? devices.filter((device) => device.status === 'online').length
  const offline = summary?.offline ?? devices.length - online

  return (
    <PageLayout
      title="Dashboard"
      description={info?.name ? `${info.name} · overview of your tailnet` : 'Overview of your tailnet'}
      actions={
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void load()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setDeployOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Deploy agent
          </Button>
        </div>
      }
    >
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Metric icon={Server} label="Devices" value={String(devices.length)} />
        <Metric icon={Wifi} label="Online" value={String(online)} tone="text-success" />
        <Metric icon={Activity} label="Offline" value={String(offline)} tone={offline ? 'text-warning' : undefined} />
        <Metric
          icon={AlertTriangle}
          label="Issues"
          value={String(issues.length)}
          tone={issues.length ? 'text-warning' : undefined}
        />
      </div>

      {issues.length > 0 && (
        <section className="mt-5 rounded-lg border border-border bg-card">
          <header className="border-b border-border px-4 py-3">
            <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
              <AlertTriangle className="h-4 w-4 text-warning" />
              Needs attention
            </h2>
          </header>
          <div className="divide-y divide-border/70">
            {issues.slice(0, 8).map((issue) => (
              <Link
                key={issue.id}
                to={issue.device_id ? `/fleet/${issue.device_id}` : '/fleet'}
                className="flex items-start gap-3 px-4 py-3 text-sm transition-colors hover:bg-muted/40"
              >
                <Circle
                  className={cn(
                    'mt-1.5 h-2 w-2 shrink-0 fill-current',
                    issue.severity === 'error' ? 'text-destructive' : 'text-warning'
                  )}
                />
                <div className="min-w-0">
                  <p className="font-medium text-foreground">{issue.title}</p>
                  <p className="truncate text-xs text-muted-foreground">{issue.detail}</p>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}

      <section className="mt-6">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-foreground">Devices</h2>
          <Link to="/fleet" className="text-xs text-muted-foreground hover:text-foreground hover:underline">
            View all
          </Link>
        </div>
        {devices.length === 0 ? (
          <div className="rounded-lg border border-border bg-card">
            <EmptyState
              icon={Server}
              title="No devices yet"
              description="Deploy an agent to a machine on your tailnet, or wait for the master device to appear."
              action={
                <Button onClick={() => setDeployOpen(true)}>
                  <Plus className="mr-2 h-4 w-4" />
                  Deploy agent
                </Button>
              }
            />
          </div>
        ) : (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {devices.map((device) => (
              <DeviceCard key={device.id} device={device} />
            ))}
          </div>
        )}
      </section>

      {summary && summary.recent_jobs.length > 0 && (
        <section className="mt-6 rounded-lg border border-border bg-card">
          <header className="border-b border-border px-4 py-3">
            <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
              <ListTodo className="h-4 w-4 text-muted-foreground" />
              Recent tasks
            </h2>
          </header>
          <div className="divide-y divide-border/70">
            {summary.recent_jobs.slice(0, 6).map((job) => (
              <div key={job.id} className="flex items-center gap-3 px-4 py-2 text-xs">
                <span
                  className={cn(
                    'rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wider',
                    job.status === 'finished'
                      ? 'bg-success/10 text-success'
                      : job.status === 'failed'
                        ? 'bg-destructive/10 text-destructive'
                        : 'bg-muted text-muted-foreground'
                  )}
                >
                  {job.status}
                </span>
                <code className="min-w-0 flex-1 truncate font-mono text-foreground/80">{job.command}</code>
                <span className="shrink-0 text-muted-foreground">{formatRelativeTime(job.created_at)}</span>
              </div>
            ))}
          </div>
        </section>
      )}

      <DeployDeviceDialog open={deployOpen} onOpenChange={setDeployOpen} onDeployed={() => void load()} />
      {loading && <span className="sr-only">Loading dashboard…</span>}
    </PageLayout>
  )
}
