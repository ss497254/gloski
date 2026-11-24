import { devicePath } from '@/app/navigation'
import { cn, formatRelativeTime } from '@/shared/lib/utils'
import type { DashboardSummary } from '@/shared/tower/types'
import { StatusDot } from '@/shared/components'
import { toneSurface } from '@/shared/lib/status'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Card } from '@/ui/card'
import { ArrowUpRight, AlertTriangle, CheckCircle2 } from 'lucide-react'
import { Link } from 'react-router-dom'

export function verdictCopy(counts: DashboardSummary['verdict'] | undefined) {
  if (!counts) return { title: 'Checking…', hint: 'Reading the fleet…' }
  if (counts.critical > 0)
    return {
      title: `${counts.critical} ${counts.critical === 1 ? 'thing' : 'things'} to fix`,
      hint: 'Start with the first item below — it is the most important.',
    }
  if (counts.warning > 0)
    return {
      title: `Mostly healthy · ${counts.warning} to check`,
      hint: 'Nothing is broken; a few things are worth a look when you have time.',
    }
  if (counts.unknown > 0)
    return {
      title: `Nothing failed · ${counts.unknown} not checked`,
      hint: 'Grey means skipped, stale or never run — not a failure.',
    }
  return { title: 'Healthy · nothing to fix', hint: 'Everything checked looks good.' }
}

/** The at-a-glance health screen: a verdict, counts and the next action. */
export function FleetHealth({ summary, expanded, onToggleExpanded }: FleetHealthProps) {
  const issues = summary.issues
  const visible = expanded ? issues : issues.slice(0, 8)
  const copy = verdictCopy(summary.verdict)

  return (
    <>
      <Card className="mt-5 gap-0 py-0">
        <div className="flex flex-wrap items-start gap-x-6 gap-y-3 px-4 py-4">
          <div className="min-w-52">
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground">Fleet health</p>
            <p className="mt-1 text-lg font-semibold text-foreground">{copy.title}</p>
            <p className="mt-0.5 text-xs text-muted-foreground">{copy.hint}</p>
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-muted-foreground">
            {([
              ['ok', 'OK', 'success'],
              ['warning', 'to check', 'warning'],
              ['critical', 'critical', 'destructive'],
              ['unknown', 'unknown', 'muted'],
            ] as const).map(([key, label, tone]) => (
              <span key={key} className="flex items-center gap-1.5">
                <span className={cn('size-2 rounded-full', toneSurface[tone])} />
                <span className="tabular-nums text-foreground/80">{summary.verdict?.[key] ?? 0}</span>
                {label}
              </span>
            ))}
          </div>
          <p className="ml-auto text-[11px] text-muted-foreground">Updated {formatRelativeTime(summary.generated_at)}</p>
        </div>
      </Card>

      <Card className="mt-5 gap-0 overflow-hidden py-0">
        <header className="border-b border-border px-4 py-3">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <AlertTriangle className="size-4 text-warning" />
            Needs attention
            <Badge variant="secondary" className="ml-1 px-1.5 py-0 text-[10px] tabular-nums">
              {issues.length}
            </Badge>
          </h2>
        </header>
        <div className="divide-y divide-border/70">
          {issues.length === 0 ? (
            <div className="flex items-center gap-3 px-4 py-4 text-sm text-muted-foreground">
              <CheckCircle2 className="size-4 text-success" />
              {summary.verdict?.unknown
                ? `Nothing failed. ${summary.verdict.unknown} result${summary.verdict.unknown === 1 ? '' : 's'} not checked yet.`
                : 'Nothing needs your attention. Every device answered its last check.'}
            </div>
          ) : (
            visible.map((issue) => (
              <Link
                key={issue.id}
                to={issue.action ?? (issue.device_id ? devicePath(issue.device_id) : '/devices')}
                className="group flex items-start gap-3 px-4 py-3 text-sm transition-colors hover:bg-muted/50"
              >
                <StatusDot status={issue.severity === 'error' ? 'destructive' : 'warning'} className="mt-1.5" />
                <div className="min-w-0">
                  <p className="font-medium text-foreground">{issue.title}</p>
                  <p className="truncate text-xs text-muted-foreground">{issue.detail}</p>
                  {issue.advice && <p className="mt-1 text-xs text-foreground/80">→ {issue.advice}</p>}
                </div>
                <span className="ml-auto mt-0.5 flex shrink-0 items-center gap-1 text-xs text-muted-foreground transition-colors group-hover:text-primary">
                  {issue.action?.includes('/jobs') ? 'View task' : 'Open'}
                  <ArrowUpRight className="size-3.5" />
                </span>
              </Link>
            ))
          )}
        </div>
        {issues.length > 8 && (
          <Button
            variant="ghost"
            className="w-full justify-start rounded-none border-t border-border text-xs text-muted-foreground hover:text-foreground"
            onClick={onToggleExpanded}
          >
            {expanded ? 'Show fewer' : `+${issues.length - 8} more`}
          </Button>
        )}
      </Card>
    </>
  )
}

interface FleetHealthProps {
  summary: DashboardSummary
  expanded: boolean
  onToggleExpanded: () => void
}
