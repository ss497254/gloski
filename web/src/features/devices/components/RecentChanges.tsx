import { formatRelativeTime } from '@/shared/lib/utils'
import { useTower } from '@/shared/tower'
import { listDeviceActions, undoDeviceAction } from '@/shared/tower/client'
import type { DeviceAction } from '@/shared/tower/types'
import { Button } from '@/ui/button'
import { History, Loader2, Undo2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'

const labels: Record<DeviceAction['kind'], string> = {
  agent_start: 'Agent started',
  agent_stop: 'Agent stopped',
  agent_restart: 'Agent restarted',
  trust_host: 'SSH host key trusted',
  undo: 'Change undone',
}

function describe(action: DeviceAction): string {
  if (action.from && action.to) return `${action.from} → ${action.to}`
  if (action.to) return `→ ${action.to}`
  return ''
}

export function RecentChanges({ deviceId }: { deviceId: string }) {
  const { refresh } = useTower()
  const [actions, setActions] = useState<DeviceAction[]>([])
  const [loading, setLoading] = useState(true)
  const [historyError, setHistoryError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  const load = useCallback(async () => {
    setHistoryError(null)
    try {
      setActions(await listDeviceActions(deviceId))
    } catch (err) {
      setHistoryError(err instanceof Error ? err.message : 'Could not load recent changes')
    } finally {
      setLoading(false)
    }
  }, [deviceId])

  useEffect(() => {
    void load()
  }, [load])

  const undo = async (action: DeviceAction) => {
    setBusy(action.id)
    try {
      await undoDeviceAction(deviceId, action.id)
      toast.success('Change undone')
      await load()
      await refresh()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not undo the change')
    } finally {
      setBusy(null)
    }
  }

  return (
    <section className="rounded-lg border border-border bg-card">
      <header className="flex items-center gap-2 border-b border-border px-4 py-3">
        <History className="h-4 w-4 text-muted-foreground" />
        <h2 className="text-sm font-semibold text-foreground">Recent changes</h2>
      </header>
      {loading ? (
        <div className="flex items-center gap-2 px-4 py-6 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading changes…
        </div>
      ) : historyError ? (
        <div className="flex items-center justify-between gap-3 px-4 py-4 text-sm text-destructive">
          <span>{historyError}</span>
          <Button variant="outline" size="sm" onClick={() => void load()}>
            Retry
          </Button>
        </div>
      ) : actions.length === 0 ? (
        <p className="px-4 py-6 text-sm text-muted-foreground">No changes recorded yet.</p>
      ) : (
        <ul className="divide-y divide-border/70">
          {actions.map((action) => {
            const canUndo = action.undoable && !action.undone_at && action.result === 'ok'
            return (
              <li key={action.id} className="flex items-center gap-3 px-4 py-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline gap-x-2">
                    <span className="text-sm font-medium text-foreground">{labels[action.kind] ?? action.kind}</span>
                    {describe(action) && (
                      <span className="font-mono text-xs text-muted-foreground">{describe(action)}</span>
                    )}
                  </div>
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {formatRelativeTime(action.created_at)}
                    {action.result === 'failed' && (
                      <span className="ml-2 text-destructive">failed{action.error ? `: ${action.error}` : ''}</span>
                    )}
                    {action.undone_at && <span className="ml-2">· undone</span>}
                  </p>
                </div>
                {canUndo && (
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={busy !== null}
                    onClick={() => void undo(action)}
                  >
                    {busy === action.id ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Undo2 className="mr-2 h-4 w-4" />
                    )}
                    Undo
                  </Button>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}
