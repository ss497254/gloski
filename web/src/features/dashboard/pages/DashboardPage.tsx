import { devicePath } from '@/app/navigation'
import { PageLayout } from '@/layouts'
import { cn, formatBytes, formatRelativeTime, formatUptime } from '@/shared/lib/utils'
import { useTower } from '@/shared/tower'
import { getDashboard, getSelfTest } from '@/shared/tower/client'
import type { DashboardSummary, FleetDevice, SelfTestResult } from '@/shared/tower/types'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Card } from '@/ui/card'
import { Skeleton } from '@/ui/skeleton'
import { DeployDeviceDialog, EmptyState, StatusBadge, StatusDot, StatusMark, UsageBar, type StatusTone, toneSurface } from '@/shared/components'
import { FleetCheckMatrix, FleetHealth } from '../components'
import {
  Activity,
  AlertTriangle,
  ArrowUpRight,
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

function Metric({
  icon: Icon,
  label,
  value,
  tone,
}: {
  icon: typeof Cpu
  label: string
  value: string
  tone?: StatusTone
}) {
  return (
    <Card className="group gap-0 p-4">
      <div className="flex items-center gap-3">
        <div className={cn('flex size-11 items-center justify-center rounded-xl', tone ? toneSurface[tone] : 'bg-primary/10 text-primary')}>
          <Icon className="size-5" />
        </div>
        <div className="min-w-0">
          <p className="truncate text-xs font-medium text-muted-foreground">{label}</p>
          <p className="text-2xl font-semibold tabular-nums text-foreground">{value}</p>
        </div>
      </div>
    </Card>
  )
}

function DeviceCard({ device }: { device: FleetDevice }) {
  const online = device.status === 'online'
  const stats = device.stats
  return (
    <Link to={devicePath(device.id)} className="group block">
      <Card className="h-full gap-0 p-4 transition-colors hover:border-primary/50">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <StatusDot status={device.status} />
              <h3 className="truncate font-medium text-foreground">{device.name}</h3>
              {device.master && (
                <Badge
                  variant="outline"
                  className="border-primary/30 bg-primary/10 px-1.5 py-0 text-[10px] uppercase tracking-wider text-primary"
                >
                  Master
                </Badge>
              )}
            </div>
            <p className="mt-1 truncate text-xs text-muted-foreground">
              {device.hostname || 'device'} · {online ? 'online' : 'offline'}
            </p>
          </div>
          <ArrowUpRight className="size-4 shrink-0 text-muted-foreground transition-colors group-hover:text-foreground" />
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
      </Card>
    </Link>
  )
}

export function DashboardPage() {
  const { mode, devices, info } = useTower()
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [health, setHealth] = useState<SelfTestResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [deployOpen, setDeployOpen] = useState(false)
  const [showAllIssues, setShowAllIssues] = useState(false)

  const load = useCallback(async () => {
    // The health card is optional; a slow quick_check must not delay the
    // dashboard itself.
    void getSelfTest().then(setHealth).catch(() => undefined)
    try {
      setSummary(await getDashboard())
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load dashboard')
    }
    setLoading(false)
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
        <Card className="gap-0 py-0">
          <EmptyState
            icon={Server}
            title="Master not available"
            description="Open the dashboard served by the Gloski master to manage devices on this tailnet."
          />
        </Card>
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
        <Metric icon={Wifi} label="Online" value={String(online)} tone="success" />
        <Metric icon={Activity} label="Offline" value={String(offline)} tone={offline ? 'warning' : undefined} />
        <Metric icon={AlertTriangle} label="Issues" value={String(issues.length)} tone={issues.length ? 'warning' : undefined} />
      </div>

      {summary && (
        <FleetHealth
          summary={summary}
          expanded={showAllIssues}
          onToggleExpanded={() => setShowAllIssues((value) => !value)}
        />
      )}

      <Link to="/settings?tab=health" className="mt-5 block">
        <Card className="gap-0 p-4 transition-colors hover:border-primary/50">
          <div className="flex items-center gap-3">
            <StatusMark
              tone={
                !health
                  ? 'muted'
                  : health.status === 'critical'
                    ? 'destructive'
                    : health.status === 'warning'
                      ? 'warning'
                      : 'success'
              }
            />
            <div className="min-w-0 flex-1">
              <p className="text-xs font-medium text-muted-foreground">Master health</p>
              <p className="truncate text-sm text-foreground">
                {!health
                  ? 'Running the master self-test…'
                  : health.status === 'ok'
                    ? 'All clear. Every master check passed.'
                    : health.status === 'critical'
                      ? 'A master check failed and needs attention.'
                      : `${health.lines.filter((line) => line.status === 'warning').length} to check on the master.`}
              </p>
            </div>
            <span className="shrink-0 text-xs text-muted-foreground">
              {health ? `${health.lines.filter((line) => line.status === 'warning').length} warnings · ${health.lines.filter((line) => line.status === 'critical').length} failed` : 'checking'}
            </span>
            <ArrowUpRight className="size-4 shrink-0 text-muted-foreground" />
          </div>
        </Card>
      </Link>

      {summary?.check_matrix && <FleetCheckMatrix matrix={summary.check_matrix} />}

      <section className="mt-6">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-foreground">Devices</h2>
          <Link to="/devices" className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
            View all
            <ArrowUpRight className="size-3" />
          </Link>
        </div>
        {devices.length === 0 ? (
          <Card className="gap-0 py-0">
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
          </Card>
        ) : (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {devices.map((device) => (
              <DeviceCard key={device.id} device={device} />
            ))}
          </div>
        )}
      </section>

      {summary && summary.recent_jobs.length > 0 && (
        <Card className="mt-6 gap-0 overflow-hidden py-0">
          <header className="border-b border-border px-4 py-3">
            <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
              <ListTodo className="size-4 text-muted-foreground" />
              Recent tasks
            </h2>
          </header>
          <div className="divide-y divide-border/70">
            {summary.recent_jobs.slice(0, 6).map((job) => (
              <div key={job.id} className="flex items-center gap-3 px-4 py-2 text-xs">
                <StatusBadge status={job.status} className="min-w-[4.5rem] justify-center px-1.5 py-0 text-[10px] uppercase tracking-wider" />
                <code className="min-w-0 flex-1 truncate font-mono text-foreground/80">{job.command}</code>
                <span className="shrink-0 text-muted-foreground">{formatRelativeTime(job.created_at)}</span>
              </div>
            ))}
          </div>
        </Card>
      )}

      <DeployDeviceDialog open={deployOpen} onOpenChange={setDeployOpen} onDeployed={() => void load()} />
      {loading && <span className="sr-only">Loading dashboard…</span>}
    </PageLayout>
  )
}
