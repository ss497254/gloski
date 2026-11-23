import { diagnoseDevice } from '@/shared/tower/client'
import { cn } from '@/shared/lib/utils'
import type { Diagnosis, DiagnosisStatus, DiagnosisStep } from '@/shared/tower/types'
import { Button } from '@/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/ui/dialog'
import { AlertTriangle, CheckCircle2, CircleDashed, Copy, Loader2, RefreshCw, Stethoscope, XCircle } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'

interface DiagnoseButtonProps {
  deviceId: string
  deviceName: string
  iconOnly?: boolean
  variant?: 'ghost' | 'outline'
  size?: 'sm' | 'icon-sm'
}

function StatusIcon({ status, className }: { status: DiagnosisStatus; className?: string }) {
  const classes = cn('h-4 w-4 shrink-0', className)
  switch (status) {
    case 'ok':
      return <CheckCircle2 className={cn(classes, 'text-success')} />
    case 'warning':
      return <AlertTriangle className={cn(classes, 'text-warning')} />
    case 'critical':
      return <XCircle className={cn(classes, 'text-destructive')} />
    default:
      return <CircleDashed className={cn(classes, 'text-muted-foreground/60')} />
  }
}

const verdictClasses: Record<DiagnosisStatus, string> = {
  ok: 'border-success/30 bg-success/5',
  warning: 'border-warning/30 bg-warning/5',
  critical: 'border-destructive/30 bg-destructive/5',
  skipped: 'border-border bg-muted/30',
}

function StepRow({ step, showAdvice }: { step: DiagnosisStep; showAdvice: boolean }) {
  return (
    <li className={cn('flex gap-3 py-2.5', step.status === 'skipped' && 'opacity-60')}>
      <StatusIcon status={step.status} className="mt-0.5" />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-baseline gap-x-2">
          <span className="text-sm font-medium text-foreground">{step.title}</span>
          <span className="text-[10px] uppercase tracking-wider text-muted-foreground">{step.status}</span>
        </div>
        <p className="text-sm text-muted-foreground">{step.summary}</p>
        {step.detail && (
          <pre className="mt-1 max-h-28 overflow-auto whitespace-pre-wrap break-words rounded bg-muted/50 px-2 py-1 font-mono text-[11px] text-muted-foreground">
            {step.detail}
          </pre>
        )}
        {showAdvice && step.advice && <p className="mt-1 text-xs text-foreground/80">{step.advice}</p>}
      </div>
    </li>
  )
}

export function DiagnoseButton({
  deviceId,
  deviceName,
  iconOnly = false,
  variant = 'outline',
  size = 'sm',
}: DiagnoseButtonProps) {
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<Diagnosis | null>(null)
  const [error, setError] = useState<string | null>(null)

  const run = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setResult(await diagnoseDevice(deviceId))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Diagnosis failed')
    } finally {
      setLoading(false)
    }
  }, [deviceId])

  useEffect(() => {
    if (!open) return
    setResult(null)
    void run()
  }, [open, run])

  const copyReport = async () => {
    if (!result) return
    const lines = [
      `Diagnosis for ${result.device}`,
      `${result.status}: ${result.headline}`,
      ...(result.advice ? [result.advice] : []),
      `Generated: ${result.generated_at}`,
      '',
      'Steps:',
      ...result.steps.map(
        (step) => `[${step.status}] ${step.title}: ${step.summary}${step.detail ? `\n    ${step.detail}` : ''}`
      ),
    ]
    try {
      await navigator.clipboard.writeText(lines.join('\n'))
      toast.success('Report copied')
    } catch {
      toast.error('Could not copy the report')
    }
  }

  return (
    <>
      <Button
        variant={variant}
        size={size}
        title={`Diagnose ${deviceName}`}
        onClick={() => setOpen(true)}
        className={iconOnly ? undefined : 'gap-2'}
      >
        <Stethoscope className={iconOnly ? 'h-3.5 w-3.5' : 'h-4 w-4'} />
        {!iconOnly && 'Diagnose'}
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Diagnose {deviceName}</DialogTitle>
            <DialogDescription>
              Walks the chain from the master to the agent and stops at the first broken link.
            </DialogDescription>
          </DialogHeader>

          {loading && (
            <div className="flex items-center gap-2 rounded-md border border-border bg-muted/30 px-3 py-6 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Checking Tailscale, SSH and the agent…
            </div>
          )}

          {error && !loading && (
            <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}

          {result && !loading && (
            <div className="space-y-4">
              <div className={cn('flex gap-3 rounded-md border px-3 py-3', verdictClasses[result.status])}>
                <StatusIcon status={result.status} className="mt-0.5 h-5 w-5" />
                <div className="min-w-0">
                  <p className="text-sm font-medium text-foreground">{result.headline}</p>
                  {result.advice && <p className="mt-1 text-sm text-muted-foreground">{result.advice}</p>}
                </div>
              </div>

              <ul className="divide-y divide-border/70 rounded-md border border-border">
                {result.steps.map((step) => (
                  <StepRow key={step.key} step={step} showAdvice={step.summary !== result.headline} />
                ))}
              </ul>
            </div>
          )}

          <DialogFooter>
            <Button variant="ghost" onClick={() => void copyReport()} disabled={!result}>
              <Copy className="mr-2 h-4 w-4" />
              Copy report
            </Button>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Close
            </Button>
            <Button onClick={() => void run()} disabled={loading}>
              <RefreshCw className={cn('mr-2 h-4 w-4', loading && 'animate-spin')} />
              Run again
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
