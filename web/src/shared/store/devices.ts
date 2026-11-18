import { DeviceClient, type DeviceStatus } from '@/shared/lib/device'
import { proxy } from 'valtio'
import { create } from 'zustand'

interface DeviceInput {
  id: string
  name: string
  status: DeviceStatus
}

interface DevicesState {
  devices: DeviceClient[]
  /** Reconcile the client list with devices reported by the master. */
  syncDevices: (devices: DeviceInput[]) => void
}

export const useDevicesStore = create<DevicesState>()((set) => ({
  devices: [],

  syncDevices: (incoming) => {
    set((state) => {
      const existing = new Map(state.devices.map((device) => [device.id, device]))
      const seen = new Set<string>()

      const devices = incoming.map((input) => {
        seen.add(input.id)
        const current = existing.get(input.id)
        if (current) {
          current.name = input.name
          current.setStatus(input.status)
          return current
        }
        return proxy(
          new DeviceClient({
            id: input.id,
            name: input.name,
            url: window.location.origin,
            status: input.status,
            apiPrefix: `/api/devices/${input.id}/api`,
          })
        )
      })

      return { devices }
    })
  },
}))
