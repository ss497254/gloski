import { useTower } from '@/shared/tower'
import type { FleetDevice } from '@/shared/tower/types'
import { useState } from 'react'
import { toast } from 'sonner'

export type DeviceAction = 'start' | 'stop' | 'remove'

/** Shared lifecycle actions for a device, used by the list and detail views. */
export function useDeviceActions(device: FleetDevice | undefined, onRemoved?: () => void) {
  const { startDevice, stopDevice, removeDevice } = useTower()
  const [busy, setBusy] = useState(false)

  const run = async (action: DeviceAction) => {
    if (!device) return
    if (action === 'stop' && !window.confirm(`Stop the agent on "${device.name}"?`)) return
    if (action === 'remove' && !window.confirm(`Remove "${device.name}" from the fleet?`)) return

    setBusy(true)
    try {
      if (action === 'start') await startDevice(device.id)
      if (action === 'stop') await stopDevice(device.id)
      if (action === 'remove') {
        await removeDevice(device.id)
        onRemoved?.()
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

  return { run, busy }
}
