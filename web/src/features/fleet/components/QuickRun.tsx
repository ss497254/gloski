import { deviceApiPrefix } from '@/shared/tower'
import { Button } from '@/ui/button'
import { Card, CardContent } from '@/ui/card'
import { Input } from '@/ui/input'
import { ExternalLink, Loader2, Play } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'

const PRESETS = ['uptime', 'df -h', 'free -h', 'ps aux --sort=-%cpu | head']

interface JobPayload {
  id?: string
  command?: string
  success?: boolean
  data?: { id?: string; command?: string }
  error?: string
}

interface QuickRunProps {
  deviceId: string
}

/** Start an ad-hoc job on a device with one click, or type a custom command. */
export function QuickRun({ deviceId }: QuickRunProps) {
  const [command, setCommand] = useState('')
  const [cwd, setCwd] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [started, setStarted] = useState<{ id: string; command: string } | null>(null)

  const run = async (requested: string) => {
    const trimmed = requested.trim()
    if (!trimmed || busy) return

    setBusy(true)
    setError(null)
    try {
      const response = await fetch(`${deviceApiPrefix(deviceId)}/jobs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ command: trimmed, cwd: cwd.trim() || undefined }),
      })
      const payload = (await response.json().catch(() => null)) as JobPayload | null
      const job = payload?.data ?? payload
      if (!response.ok || !job?.id) {
        throw new Error(payload?.error || `Device returned ${response.status}`)
      }
      setStarted({ id: job.id, command: job.command ?? trimmed })
      setCommand('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start the job')
    } finally {
      setBusy(false)
    }
  }

  const submit = (event: FormEvent) => {
    event.preventDefault()
    void run(command)
  }

  return (
    <Card className="mb-4 border-white/10 bg-[#0b0f14] text-[#c9d1d9]">
      <CardContent className="p-4">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <span className="mr-1 font-mono text-[10px] uppercase tracking-wider text-white/35">quick run</span>
          {PRESETS.map((preset) => (
            <Button
              key={preset}
              type="button"
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() => void run(preset)}
              className="h-7 border-white/10 bg-white/[0.03] px-2 font-mono text-[11px] text-white/70 hover:bg-white/[0.07] hover:text-white"
            >
              <Play className="mr-1.5 h-3 w-3" />
              {preset}
            </Button>
          ))}
        </div>

        <form onSubmit={submit} className="flex flex-col gap-2 sm:flex-row">
          <Input
            value={command}
            onChange={(event) => setCommand(event.target.value)}
            placeholder="command to run on this device…"
            className="h-8 flex-1 border-white/10 bg-black/30 font-mono text-xs text-white placeholder:text-white/25"
            disabled={busy}
          />
          <Input
            value={cwd}
            onChange={(event) => setCwd(event.target.value)}
            placeholder="working directory (optional)"
            className="h-8 border-white/10 bg-black/30 font-mono text-xs text-white placeholder:text-white/25 sm:w-64"
            disabled={busy}
          />
          <Button type="submit" size="sm" disabled={busy || !command.trim()} className="h-8">
            {busy ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <Play className="mr-1.5 h-3.5 w-3.5" />}
            Run
          </Button>
        </form>

        {error && <p className="mt-2 font-mono text-xs text-red-400">{error}</p>}
        {started && (
          <p className="mt-2 flex flex-wrap items-center gap-1.5 font-mono text-xs text-white/40">
            started <span className="text-white/70">{started.command}</span>
            <span className="text-white/25">({started.id.slice(0, 8)})</span>
            <Link
              to={`/servers/tower-${deviceId}/jobs`}
              className="ml-1 inline-flex items-center gap-1 text-emerald-400 hover:text-emerald-300"
            >
              open jobs <ExternalLink className="h-3 w-3" />
            </Link>
          </p>
        )}
      </CardContent>
    </Card>
  )
}
