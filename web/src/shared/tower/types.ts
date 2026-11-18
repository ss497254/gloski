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
  port: number
  deploy_available: boolean
  agent_binary: string
  tailscale_available: boolean
  tailscale_socket: string
}

export interface Deployment {
  id: string
  name: string
  host: string
  user: string
  ssh_port: number
  agent_port: number
  master_url: string
  status: 'pending' | 'uploading' | 'starting' | 'running' | 'stopped' | 'failed'
  remote_pid?: string
  device_id?: string
  error?: string
  created_at: string
  updated_at: string
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
  deploy_id?: string
  managed: boolean
  master: boolean
  deployment?: Deployment | null
}

export interface TailnetNode {
  hostname: string
  dns_name: string
  os: string
  ips: string[]
  online: boolean
  self: boolean
}

export interface TailnetResponse {
  available: boolean
  backend_state?: string
  error?: string
  self?: TailnetNode
  peers: TailnetNode[]
}

export interface DeployRequest {
  name: string
  host: string
  user: string
  ssh_port: number
  agent_port: number
  master_url: string
}

export interface FleetEvent {
  type: 'snapshot' | 'devices' | 'device.updated' | 'device.removed'
  device?: FleetDevice
  device_id?: string
  devices?: FleetDevice[]
}
