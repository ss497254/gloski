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
  agent_builds: string[]
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

export type DiagnosisStatus = 'ok' | 'warning' | 'critical' | 'skipped'

export interface DiagnosisStep {
  key: string
  title: string
  status: DiagnosisStatus
  summary: string
  detail?: string
  advice?: string
}

export interface Diagnosis {
  device_id: string
  device: string
  status: DiagnosisStatus
  headline: string
  advice?: string
  steps: DiagnosisStep[]
  generated_at: string
}

export interface DeviceAction {
  id: string
  device_id: string
  kind: 'agent_start' | 'agent_stop' | 'agent_restart' | 'trust_host' | 'undo'
  target: string
  from?: string
  to?: string
  result: 'ok' | 'failed'
  error?: string
  undoable: boolean
  undone_at?: string
  created_at: string
}

export interface SelfTestLine {
  key: string
  title: string
  status: DiagnosisStatus
  summary: string
  detail?: string
}

export interface SelfTestResult {
  status: DiagnosisStatus
  lines: SelfTestLine[]
  generated_at: string
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

/** One metric value recorded by the master from a device heartbeat. */
export interface MetricSample {
  device_id: string
  metric: string
  ts: string
  value: number
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
  agent_port?: number
}

export interface DashboardIssue {
  id: string
  severity: 'error' | 'warning' | 'info'
  title: string
  detail: string
  advice?: string
  action?: string
  device_id?: string
}

export type CheckStatus = 'ok' | 'warning' | 'critical' | 'skipped' | 'unknown'

export interface CheckBlock {
  check_id: string
  status: CheckStatus
  summary: string
  stale: boolean
  ran_at: string
  details?: Array<{ key: string; value: string }>
}

export interface CheckGroupCell {
  group: string
  checks: CheckBlock[]
}

export interface CheckMatrixRow {
  device_id: string
  device_name: string
  device_status: string
  cells: CheckGroupCell[]
}

export interface CheckMatrix {
  groups: string[]
  rows: CheckMatrixRow[]
}

export interface DashboardSummary {
  devices: number
  online: number
  offline: number
  verdict: { ok: number; warning: number; critical: number; unknown: number }
  issues: DashboardIssue[]
  check_matrix: CheckMatrix
  recent_jobs: Array<{
    id: string
    device_id: string
    command: string
    status: string
    created_at: string
  }>
  generated_at: string
}

export interface FleetEvent {
  type: 'snapshot' | 'devices' | 'device.updated' | 'device.removed'
  device?: FleetDevice
  device_id?: string
  devices?: FleetDevice[]
}
