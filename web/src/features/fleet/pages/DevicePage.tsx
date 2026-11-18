import { PageLayout } from '@/layouts'
import { cn } from '@/shared/lib/utils'
import { useTower } from '@/shared/tower'
import type { FleetDevice } from '@/shared/tower/types'
import { Button } from '@/ui/button'
import { ArrowLeft, FolderOpen, ListTodo, Loader2, Play, Square, Terminal, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { BtopView } from '../components/BtopView'
import { QuickRun } from '../components/QuickRun'
import { useDeviceStats } from '../hooks/useDeviceStats'

function ToolLink({ to, icon: Icon, label }: { to: string; icon: typeof FolderOpen; label: string }) {
  return (
    <Button asChild variant="outline" size="sm">
      <Link to={to}>
        <Icon className="mr-2 h-3.5 w-3.5" />
        {label}
      </Link>
    </Button>
  )
}

export function DevicePage() {
  const { deviceId } = useParams<{ deviceId: string }>()
  const { mode, devices, startDevice, stopDevice, removeDevice } = useTower()
  const navigate = useNavigate()
  const device = devices.find((item) => item.id === deviceId)
  const live = useDeviceStats(device?.id, mode === 'tower' && device?.status === 'online')
  const [busy, setBusy] = useState(false)

  if (mode === 'standalone' || !device) {
    return (
      <PageLayout
        title="Device"
        description={mode === 'standalone' ? 'Not connected to a control tower' : 'Device not found'}
      >
        <Button asChild variant="outline">
          <Link to="/fleet">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to fleet
          </Link>
        </Button>
      </PageLayout>
    )
  }

  const managedDevice: FleetDevice = device
  const online = managedDevice.status === 'online'

  const run = async (action: 'start' | 'stop' | 'remove') => {
    if (action === 'stop' && !window.confirm(`Stop the agent on "${managedDevice.name}"?`)) return
    if (action === 'remove' && !window.confirm(`Remove "${managedDevice.name}" from the fleet?`)) return
    setBusy(true)
    try {
      if (action === 'start') await startDevice(managedDevice.id)
      if (action === 'stop') await stopDevice(managedDevice.id)
      if (action === 'remove') {
        await removeDevice(managedDevice.id)
        navigate('/fleet')
      }
      toast.success(
        action === 'start' ? 'Agent start requested' : action === 'stop' ? 'Agent stopped' : 'Device removed'
      )
    } catch (err) {
      toast.error(err instanceof Error ? err.message : `Failed to ${action} device`)
    } finally {
      setBusy(false)
    }
  }

  const toolBase = `/servers/tower-${managedDevice.id}`

  return (
    <PageLayout
      title={managedDevice.name}
      description={`${managedDevice.hostname || 'device'} · ${managedDevice.url || 'agent URL pending'}`}
      actions={
        <div className="flex items-center gap-2">
          <Button asChild variant="outline" size="sm">
            <Link to="/fleet">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Fleet
            </Link>
          </Button>
          {managedDevice.managed && (
            <Button variant="outline" size="sm" disabled={busy} onClick={() => void run(online ? 'stop' : 'start')}>
              {busy ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : online ? (
                <Square className="mr-2 h-3.5 w-3.5" />
              ) : (
                <Play className="mr-2 h-3.5 w-3.5" />
              )}
              {online ? 'Stop agent' : 'Start agent'}
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={busy}
            title="Remove device"
            onClick={() => void run('remove')}
          >
            <Trash2 className="h-4 w-4 text-destructive" />
          </Button>
        </div>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <span
          className={cn(
            'inline-flex items-center gap-1.5 rounded-full border border-border px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider',
            online ? 'text-success' : 'text-muted-foreground'
          )}
        >
          <span className={cn('h-1.5 w-1.5 rounded-full', online ? 'bg-success' : 'bg-muted-foreground/50')} />
          {managedDevice.status}
        </span>
        {managedDevice.master && (
          <span className="rounded border border-primary/30 bg-primary/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider">
            Master device
          </span>
        )}
        <ToolLink to={`${toolBase}/files`} icon={FolderOpen} label="Files" />
        <ToolLink to={`${toolBase}/terminal`} icon={Terminal} label="Terminal" />
        <ToolLink to={`${toolBase}/jobs`} icon={ListTodo} label="Jobs" />
      </div>

      <QuickRun deviceId={managedDevice.id} />
      <BtopView
        device={managedDevice}
        stats={live.stats ?? managedDevice.stats ?? null}
        processes={live.processes}
        connected={live.connected}
      />
    </PageLayout>
  )
}
