import { DeviceProvider } from '@/shared/context'
import { Outlet } from 'react-router-dom'

/** Provides the current device to all routes under /devices/:deviceId. */
export function DeviceLayout() {
  return (
    <DeviceProvider>
      <Outlet />
    </DeviceProvider>
  )
}
