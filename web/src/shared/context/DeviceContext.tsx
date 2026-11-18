/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { useParams } from 'react-router-dom'
import { useDevicesStore } from '@/shared/store/devices'
import { useTower } from '@/shared/tower'
import type { DeviceClient } from '@/shared/lib/device'

interface DeviceContextValue {
  device: DeviceClient
  deviceId: string
}

const DeviceContext = createContext<DeviceContextValue | null>(null)

/** Resolves the device in the current route for device-scoped feature pages. */
export function DeviceProvider({ children }: { children: ReactNode }) {
  const { deviceId } = useParams<{ deviceId: string }>()
  const { mode } = useTower()
  const device = useDevicesStore((state) => (deviceId ? state.devices.find((item) => item.id === deviceId) : undefined))

  const value = useMemo<DeviceContextValue | null>(
    () => (device && deviceId ? { device, deviceId } : null),
    [device, deviceId]
  )

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
