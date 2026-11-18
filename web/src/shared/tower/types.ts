import type { SystemStats } from '@/shared/api'

export type TowerMode = 'loading' | 'tower' | 'standalone'

export interface TowerInfo {
  mode: string
  name: string
  version: string
  started_at: string
  devices: number
  online: number
  heartbeat_interval: number
}

export interface FleetDevice {
  id: string
  name: string
  url: string
  hostname: string
  platform: string
  version: string
  status: 'online' | 'offline'
  stats?: SystemStats | null
  created_at: string
  last_seen: string
}

export interface FleetEvent {
  type: 'snapshot' | 'devices' | 'device.updated' | 'device.removed'
  device?: FleetDevice
  device_id?: string
  devices?: FleetDevice[]
}
