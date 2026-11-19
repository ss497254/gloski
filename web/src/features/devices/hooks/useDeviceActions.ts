import { useTower } from '@/shared/tower'
import { trustDeviceHost } from '@/shared/tower/client'
import type { FleetDevice } from '@/shared/tower/types'
import { useState } from 'react'
import { toast } from 'sonner'

export type DeviceAction = 'start' | 'restart' | 'stop' | 'remove' | 'trust'

const done: Record<DeviceAction, string> = {
  start: 'Agent installed and started',
  restart: 'Agent restarted',
  stop: 'Agent stopped',
  remove: 'Device removed',
  trust: 'New host key trusted and agent installed',
}

/** Shared lifecycle actions for a device, used by the list and detail views. */
export function useDeviceActions(device: FleetDevice | undefined, onRemoved?: () => void) {
  const { startDevice, stopDevice, removeDevice, refresh } = useTower()
  const [busy, setBusy] = useState(false)

  const run = async (action: DeviceAction) => {
    if (!device) return
    if (
      action === 'stop' &&
      !window.confirm(`Stop the agent on "${device.name}"? It stays stopped until you start it.`)
    )
      return
    if (action === 'restart' && !window.confirm(`Restart the agent on "${device.name}"? Running jobs will end.`)) return
    if (action === 'remove' && !window.confirm(`Remove "${device.name}" from the fleet?`)) return

    setBusy(true)
    try {
      // Start and restart are the same thing: the master runs the install
      // script, which replaces and restarts whatever is on the device.
      if (action === 'start' || action === 'restart') await startDevice(device.id)
      if (action === 'stop') await stopDevice(device.id)
      if (action === 'trust') {
        await trustDeviceHost(device.id)
        await refresh()
      }
      if (action === 'remove') {
        await removeDevice(device.id)
        onRemoved?.()
      }
      toast.success(done[action])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : `Failed to ${action} device`)
    } finally {
      setBusy(false)
    }
  }

  return { run, busy }
}
