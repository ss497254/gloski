/* eslint-disable react-refresh/only-export-components */
import { useDevicesStore } from '@/shared/store/devices'
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import {
  connectFleetSocket,
  deleteDevice,
  detectTower,
  listDevices,
  startDevice as startDeviceRequest,
  stopDevice as stopDeviceRequest,
  type FleetSocket,
} from './client'
import type { FleetDevice, TowerInfo, TowerMode } from './types'

interface TowerContextValue {
  mode: TowerMode
  info: TowerInfo | null
  devices: FleetDevice[]
  connected: boolean
  error: string | null
  refresh: () => Promise<void>
  removeDevice: (id: string) => Promise<void>
  startDevice: (id: string) => Promise<void>
  stopDevice: (id: string) => Promise<void>
}

const TowerContext = createContext<TowerContextValue | null>(null)

function sortDevices(devices: FleetDevice[]): FleetDevice[] {
  return [...devices].sort((a, b) => {
    if (a.status !== b.status) return a.status === 'online' ? -1 : 1
    return a.name.localeCompare(b.name)
  })
}

export function TowerProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<TowerMode>('loading')
  const [info, setInfo] = useState<TowerInfo | null>(null)
  const [devices, setDevices] = useState<FleetDevice[]>([])
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const socketRef = useRef<FleetSocket | null>(null)
  const syncDevices = useDevicesStore((s) => s.syncDevices)

  const refresh = useCallback(async () => {
    try {
      const list = await listDevices()
      setDevices(sortDevices(list))
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load devices')
    }
  }, [])

  useEffect(() => {
    let cancelled = false

    const boot = async () => {
      const detected = await detectTower()
      if (cancelled) return
      if (!detected) {
        setMode('standalone')
        return
      }

      setInfo(detected)
      setMode('tower')
      await refresh()
      if (cancelled) return

      socketRef.current = connectFleetSocket({
        onStatus: setConnected,
        onEvent: (event) => {
          setDevices((current) => {
            switch (event.type) {
              case 'snapshot':
              case 'devices':
                return sortDevices(event.devices ?? current)
              case 'device.updated': {
                if (!event.device) return current
                const next = current.filter((d) => d.id !== event.device!.id)
                next.push(event.device)
                return sortDevices(next)
              }
              case 'device.removed':
                return current.filter((d) => d.id !== event.device_id)
              default:
                return current
            }
          })
        },
      })
    }

    boot()
    return () => {
      cancelled = true
      socketRef.current?.close()
      socketRef.current = null
    }
  }, [refresh])

  // Keep tower devices available to the existing device-scoped feature pages.
  useEffect(() => {
    if (mode !== 'tower') return
    syncDevices(
      devices.map((device) => ({
        id: device.id,
        name: device.name,
        status: device.status === 'online' ? 'online' : 'offline',
      }))
    )
  }, [devices, mode, syncDevices])

  const removeDevice = useCallback(async (id: string) => {
    await deleteDevice(id)
    setDevices((current) => current.filter((device) => device.id !== id))
  }, [])

  const startDevice = useCallback(
    async (id: string) => {
      await startDeviceRequest(id)
      await refresh()
    },
    [refresh]
  )

  const stopDevice = useCallback(
    async (id: string) => {
      await stopDeviceRequest(id)
      await refresh()
    },
    [refresh]
  )

  const value = useMemo<TowerContextValue>(
    () => ({ mode, info, devices, connected, error, refresh, removeDevice, startDevice, stopDevice }),
    [mode, info, devices, connected, error, refresh, removeDevice, startDevice, stopDevice]
  )

  return <TowerContext.Provider value={value}>{children}</TowerContext.Provider>
}

export function useTower(): TowerContextValue {
  const context = useContext(TowerContext)
  if (!context) {
    throw new Error('useTower must be used within a TowerProvider')
  }
  return context
}

export function useTowerOptional(): TowerContextValue | null {
  return useContext(TowerContext)
}
