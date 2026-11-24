/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useMemo, type ReactNode } from 'react'
import { useParams } from 'react-router-dom'
import { useTower } from '@/shared/tower'
import { DeviceClient } from '@/shared/lib/device'

interface DeviceContextValue {
  device: DeviceClient
  deviceId: string
}

const DeviceContext = createContext<DeviceContextValue | null>(null)

/** Resolves the device in the current route for device-scoped feature pages. */
export function DeviceProvider({ children }: { children: ReactNode }) {
  const { deviceId } = useParams<{ deviceId: string }>()
  const { mode, devices } = useTower()
  const fleetDevice = devices.find((item) => item.id === deviceId)
  const hasFleetDevice = fleetDevice !== undefined
  const client = useMemo(
    () =>
      deviceId
        ? new DeviceClient({
            id: deviceId,
            name: deviceId,
            url: window.location.origin,
            status: 'offline',
            apiPrefix: `/api/devices/${encodeURIComponent(deviceId)}/api`,
          })
        : null,
    [deviceId]
  )

  const value = useMemo<DeviceContextValue | null>(
    () => (hasFleetDevice && deviceId && client ? { device: client, deviceId } : null),
    [client, deviceId, hasFleetDevice]
  )

  useEffect(() => () => client?.getClient().dispose(), [client])

  if (!value) {
    return (
      <div className="p-6 text-sm text-muted-foreground">
        {mode === 'loading' || mode === 'tower' ? 'Connecting to device…' : 'Device not found.'}
      </div>
    )
  }
  return <DeviceContext.Provider value={value}>{children}</DeviceContext.Provider>
}

export function useDevice(): DeviceContextValue {
  const context = useContext(DeviceContext)
  if (!context) throw new Error('useDevice must be used within a DeviceProvider')
  return context
}
