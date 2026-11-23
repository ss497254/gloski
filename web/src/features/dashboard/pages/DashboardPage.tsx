import { PageLayout } from '@/layouts'
import { devicePath } from '@/app/navigation'
import { cn, formatBytes, formatRelativeTime, formatUptime } from '@/shared/lib/utils'
import { getDashboard } from '@/shared/tower/client'
import type { DashboardSummary, FleetDevice } from '@/shared/tower/types'
import { useTower } from '@/shared/tower'
import { Button } from '@/ui/button'
import { DeployDeviceDialog, EmptyState, UsageBar } from '@/shared/components'
import { Skeleton } from '@/ui/skeleton'
import {
  Activity,
  AlertTriangle,
  ArrowUpRight,
  Circle,
  Copy,
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

function DeviceCard({ device }: { device: FleetDevice }) {
  const online = device.status === 'online'
  const stats = device.stats
  return (
    <Link
      to={devicePath(device.id)}
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
          ↓ {formatBytes(stats?.network.rx_rate ?? 0)}/s · ↑ {formatBytes(stats?.network.tx_rate ?? 0)}/s
        </span>
        <span>{stats ? formatUptime(stats.uptime) : formatRelativeTime(device.last_seen)}</span>
      </div>
    </Link>
  )
}

export function DashboardPage() {
  const { mode, devices, info } = useTower()
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [deployOpen, setDeployOpen] = useState(false)
  const [showAllIssues, setShowAllIssues] = useState(false)

  const load = useCallback(async () => {
    try {
      setSummary(await getDashboard())
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load dashboard')
    } finally {
      setLoading(false)
    }
  }, [])

  const copyReport = useCallback(async () => {
    if (!summary) return
    const lines = [
      `Gloski report${info?.name ? ` — ${info.name}` : ''}`,
      `Generated: ${summary.generated_at}`,
      `Devices: ${summary.online} online, ${summary.offline} offline`,
      `Issues: ${summary.issues.length}`,
      '',
      ...summary.issues.flatMap((issue) => [
        `- [${issue.severity}] ${issue.title}: ${issue.detail}`,
        ...(issue.advice ? [`  Advice: ${issue.advice}`] : []),
      ]),
    ]
    try {
      await navigator.clipboard.writeText(lines.join('\n'))
      toast.success('Report copied')
    } catch {
      toast.error('Could not copy the report')
    }
  }, [summary, info])

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
  const visibleIssues = showAllIssues ? issues : issues.slice(0, 8)
  const online = summary?.online ?? devices.filter((device) => device.status === 'online').length
  const offline = summary?.offline ?? devices.length - online

  return (
    <PageLayout
      title="Dashboard"
      description={info?.name ? `${info.name} · overview of your tailnet` : 'Overview of your tailnet'}
      actions={
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void copyReport()} disabled={!summary}>
            <Copy className="mr-2 h-4 w-4" />
            Copy report
          </Button>
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
            {visibleIssues.map((issue) => (
              <Link
                key={issue.id}
                to={issue.action ?? (issue.device_id ? devicePath(issue.device_id) : '/devices')}
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
                  {issue.advice && <p className="mt-1 text-xs text-foreground/80">→ {issue.advice}</p>}
                </div>
                <ArrowUpRight className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              </Link>
            ))}
          </div>
          {issues.length > 8 && (
            <button
              type="button"
              className="w-full border-t border-border px-4 py-2 text-left text-xs text-muted-foreground transition-colors hover:bg-muted/40 hover:text-foreground"
              onClick={() => setShowAllIssues((value) => !value)}
            >
              {showAllIssues ? 'Show fewer' : `+${issues.length - 8} more`}
            </button>
          )}
        </section>
      )}

      <section className="mt-6">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-foreground">Devices</h2>
          <Link to="/devices" className="text-xs text-muted-foreground hover:text-foreground hover:underline">
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
