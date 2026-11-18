import { ServerProvider } from '@/shared/context'
import { Outlet, useParams } from 'react-router-dom'

/** Provides the virtual agent server for all routes under /devices/:deviceId. */
export function DeviceLayout() {
  const { deviceId } = useParams<{ deviceId: string }>()
  return (
    <ServerProvider serverId={deviceId ? `tower-${deviceId}` : undefined}>
      <Outlet />
    </ServerProvider>
  )
}
