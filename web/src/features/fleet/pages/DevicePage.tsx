import { PageLayout } from '@/layouts'
import { useTower } from '@/shared/tower'
import { Button } from '@/ui/button'
import { ArrowLeft, FolderOpen, Settings2, Trash2 } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { BtopView } from '../components/BtopView'
import { QuickRun } from '../components/QuickRun'
import { useDeviceStats } from '../hooks/useDeviceStats'

export function DevicePage() {
  const { deviceId } = useParams<{ deviceId: string }>()
  const { mode, devices, removeDevice } = useTower()
  const navigate = useNavigate()
  const device = devices.find((item) => item.id === deviceId)
  const live = useDeviceStats(device?.id, mode === 'tower' && device?.status === 'online')

  if (mode === 'standalone') {
    return (
      <PageLayout title="Device" description="Not connected to a control tower">
        <Button asChild variant="outline">
          <Link to="/fleet">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to fleet
          </Link>
        </Button>
      </PageLayout>
    )
  }

  if (!device) {
    return (
      <PageLayout title="Device not found" description="It may have been removed from the fleet">
        <Button asChild variant="outline">
          <Link to="/fleet">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to fleet
          </Link>
        </Button>
      </PageLayout>
    )
  }

  const handleRemove = async () => {
    if (!window.confirm(`Remove "${device.name}" from the fleet? The agent keeps running until it is stopped.`)) {
      return
    }
    await removeDevice(device.id)
    navigate('/fleet')
  }

  return (
    <PageLayout
      title={device.name}
      description={`${device.hostname || device.name} · ${device.url || 'agent URL pending'}`}
      actions={
        <div className="flex items-center gap-2">
          <Button asChild variant="outline" size="sm">
            <Link to="/fleet">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Fleet
            </Link>
          </Button>
          <Button asChild variant="outline" size="sm">
            <Link to={`/servers/tower-${device.id}`}>
              <FolderOpen className="mr-2 h-4 w-4" />
              Files & tools
            </Link>
          </Button>
          <Button asChild variant="outline" size="sm">
            <Link to={`/servers/tower-${device.id}/jobs`}>
              <Settings2 className="mr-2 h-4 w-4" />
              Jobs
            </Link>
          </Button>
          <Button variant="ghost" size="icon" onClick={handleRemove} title="Remove device">
            <Trash2 className="h-4 w-4 text-destructive" />
          </Button>
        </div>
      }
    >
      <QuickRun deviceId={device.id} />
      <BtopView
        device={device}
        stats={live.stats ?? device.stats ?? null}
        processes={live.processes}
        connected={live.connected}
      />
    </PageLayout>
  )
}
