import { StatusMark } from '@/shared/components'
import type { StatusTone } from '@/shared/lib/status'
import { cn, formatRelativeTime } from '@/shared/lib/utils'
import { getSelfTest } from '@/shared/tower/client'
import type { DiagnosisStatus, SelfTestLine, SelfTestResult } from '@/shared/tower/types'
import { Alert, AlertDescription } from '@/ui/alert'
import { Button } from '@/ui/button'
import { Card } from '@/ui/card'
import { Spinner } from '@/ui/spinner'
import { AlertTriangle, CheckCircle2, CircleDashed, Copy, RefreshCw, XCircle } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'

const toneClasses: Record<DiagnosisStatus, string> = {
  ok: 'border-success/30 bg-success/5',
  warning: 'border-warning/30 bg-warning/5',
  critical: 'border-destructive/30 bg-destructive/5',
  skipped: 'border-border bg-muted/30',
}

function StatusIcon({ status }: { status: DiagnosisStatus }) {
  switch (status) {
    case 'ok':
      return <CheckCircle2 className="h-4 w-4 shrink-0 text-success" />
    case 'warning':
      return <AlertTriangle className="h-4 w-4 shrink-0 text-warning" />
    case 'critical':
      return <XCircle className="h-4 w-4 shrink-0 text-destructive" />
    default:
      return <CircleDashed className="h-4 w-4 shrink-0 text-muted-foreground/60" />
  }
}

function Line({ line }: { line: SelfTestLine }) {
  return (
    <li className={cn('flex gap-3 px-4 py-3', line.status === 'skipped' && 'opacity-60')}>
      <StatusIcon status={line.status} />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-baseline gap-x-2">
          <span className="text-sm font-medium text-foreground">{line.title}</span>
          <span className="text-[10px] uppercase tracking-wider text-muted-foreground">{line.status}</span>
        </div>
        <p className="text-sm text-muted-foreground">{line.summary}</p>
        {line.detail && (
          <pre className="mt-1 max-h-32 overflow-auto whitespace-pre-wrap break-words rounded bg-muted/50 px-2 py-1 font-mono text-[11px] text-muted-foreground">
            {line.detail}
          </pre>
        )}
      </div>
    </li>
  )
}

/** Keep the backend's order, but read it as a few named sections. */
export function groupSections(lines: SelfTestLine[]): Array<{ name: string; lines: SelfTestLine[] }> {
  const order: string[] = []
  const grouped = new Map<string, SelfTestLine[]>()
  for (const line of lines) {
    if (!grouped.has(line.section)) {
      grouped.set(line.section, [])
      order.push(line.section)
    }
    grouped.get(line.section)!.push(line)
  }
  return order.map((name) => ({ name, lines: grouped.get(name)! }))
}

export function sectionSummary(lines: SelfTestLine[]): string {
  const failed = lines.filter((line) => line.status === 'critical').length
  const warnings = lines.filter((line) => line.status === 'warning').length
  if (failed) return `${failed} failed`
  if (warnings) return `${warnings} to check`
  if (lines.every((line) => line.status === 'skipped')) return 'not run yet'
  return 'all clear'
}

function sectionTone(lines: SelfTestLine[]): StatusTone {
  if (lines.some((line) => line.status === 'critical')) return 'destructive'
  if (lines.some((line) => line.status === 'warning')) return 'warning'
  if (lines.every((line) => line.status === 'skipped')) return 'muted'
  return 'success'
}

function verdict(result: SelfTestResult): string {
  switch (result.status) {
    case 'critical':
      return 'The master has a problem that needs attention.'
    case 'warning':
      return 'The master has a warning worth looking at.'
    default:
      return 'The master passed its self-test.'
  }
}

export function MasterHealthSettings() {
  const [result, setResult] = useState<SelfTestResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setResult(await getSelfTest())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Self-test failed')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void run()
  }, [run])

  const copyReport = async () => {
    if (!result) return
    const lines = [
      `Gloski master self-test — ${result.generated_at}`,
      verdict(result),
      '',
      ...groupSections(result.lines).flatMap((section) => [
        `${section.name} — ${sectionSummary(section.lines)}`,
        ...section.lines.map(
          (line) => `  [${line.status}] ${line.title}: ${line.summary}${line.detail ? `\n      ${line.detail}` : ''}`,
        ),
        '',
      ]),
    ]
    try {
      await navigator.clipboard.writeText(lines.join('\n'))
      toast.success('Report copied')
    } catch {
      toast.error('Could not copy the report')
    }
  }

  return (
    <section className="space-y-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-base font-semibold text-foreground">Master health</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Read-only checks of the master itself. Nothing is changed here.
          </p>
        </div>
        <div className="flex shrink-0 gap-2">
          <Button variant="outline" size="sm" onClick={() => void copyReport()} disabled={!result}>
            <Copy className="mr-2 h-4 w-4" />
            Copy report
          </Button>
          <Button size="sm" onClick={() => void run()} disabled={loading}>
            {loading ? <Spinner /> : <RefreshCw className="mr-2 h-4 w-4" />}
            Run again
          </Button>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {result && (
        <Alert
          role={result.status === 'warning' || result.status === 'critical' ? 'alert' : 'status'}
          variant={result.status === 'critical' ? 'destructive' : result.status === 'warning' ? 'warning' : 'default'}
          className={cn('items-start', toneClasses[result.status])}
        >
          <StatusIcon status={result.status} />
          <div className="min-w-0">
            <p className="text-sm font-medium text-foreground">{verdict(result)}</p>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {result.lines.filter((line) => line.status === 'warning').length} warnings ·{' '}
              {result.lines.filter((line) => line.status === 'critical').length} failed · updated{' '}
              {formatRelativeTime(result.generated_at)}
            </p>
          </div>
        </Alert>
      )}

      {result && (
        <div className="space-y-3">
          {groupSections(result.lines).map((section) => (
            <Card key={section.name} className="gap-0 overflow-hidden py-0">
              <header className="flex items-center gap-2 border-b border-border px-4 py-2.5">
                <StatusMark tone={sectionTone(section.lines)} />
                <h3 className="text-sm font-medium text-foreground">{section.name}</h3>
                <span className="ml-auto text-[11px] text-muted-foreground">{sectionSummary(section.lines)}</span>
              </header>
              <ul className="divide-y divide-border/70">
                {section.lines.map((line) => (
                  <Line key={line.key} line={line} />
                ))}
              </ul>
            </Card>
          ))}
        </div>
      )}
    </section>
  )
}
