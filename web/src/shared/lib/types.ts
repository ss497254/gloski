// =============================================================================
// API Response Types
// =============================================================================

export interface APIResponse<T = unknown> {
  success: boolean
  data?: T
  error?: string
}

// =============================================================================
// Auth Types
// =============================================================================

export interface LoginResponse {
  token: string
  expires_in: number
}

export interface AuthStatus {
  authenticated: boolean
}

// =============================================================================
// System Types
// =============================================================================

export interface SystemStats {
  hostname: string
  platform: string
  uptime: number
  boot_time: number
  cpu: CPUStats
  memory: MemoryStats
  swap: SwapStats
  disks: DiskStats[]
  network: NetworkStats
  load_avg: LoadAvg
  processes: number
  goroutines: number
}

export interface CPUStats {
  cores: number
  model_name: string
  usage_percent: number
  user: number
  system: number
  idle: number
  iowait: number
}

export interface MemoryStats {
  total: number
  used: number
  free: number
  available: number
  cached: number
  buffers: number
  used_percent: number
}

export interface SwapStats {
  total: number
  used: number
  free: number
  used_percent: number
}

export interface DiskStats {
  device: string
  mount_point: string
  fs_type: string
  total: number
  used: number
  free: number
  used_percent: number
}

export interface NetworkStats {
  interfaces: NetworkInterface[]
  total_rx: number
  total_tx: number
}

export interface NetworkInterface {
  name: string
  rx_bytes: number
  tx_bytes: number
  rx_packets: number
  tx_packets: number
  up: boolean
}

export interface LoadAvg {
  load1: number
  load5: number
  load15: number
}

export interface ProcessInfo {
  pid: number
  name: string
  state: string
  user: string
  cpu: number
  memory: number
  vsz: number
  rss: number
  command: string
}

// =============================================================================
// File Types
// =============================================================================

export interface FileEntry {
  name: string
  path: string
  type: 'file' | 'directory'
  size: number
  modified: string
  permissions: string
}

export interface ListResponse {
  path: string
  entries: FileEntry[]
}

export interface ReadResponse {
  content: string
  path: string
}

export interface SearchResult {
  path: string
  name: string
  type: 'file' | 'directory'
  size: number
  match?: string
  line_num?: number
}

// =============================================================================
// Task Types
// =============================================================================

export interface Task {
  id: string
  command: string
  cwd: string
  status: 'running' | 'stopped' | 'completed' | 'failed'
  pid: number
  started_at: string
  ended_at?: string
  exit_code?: number
}

export interface TasksResponse {
  tasks: Task[]
}

export interface TaskLogsResponse {
  logs: string
}

// =============================================================================
// Systemd Types
// =============================================================================

export interface SystemdUnit {
  name: string
  description: string
  load_state: string
  active_state: string
  sub_state: string
  enabled: boolean
}

export interface SystemdResponse {
  units: SystemdUnit[]
}

export interface SystemdLogsResponse {
  logs: string
}
