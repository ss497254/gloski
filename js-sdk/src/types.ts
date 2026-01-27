// =============================================================================
// Client Configuration
// =============================================================================

export interface GloskiClientConfig {
  /** Server URL (e.g., "https://server.example.com") */
  url: string
  /** API key for authentication */
  apiKey?: string
  /** JWT token for authentication */
  token?: string
  /** Request timeout in milliseconds (default: 30000) */
  timeout?: number
  /** API path prefix (default: "/api") */
  apiPrefix?: string
  /** Called when server returns 401 Unauthorized */
  onUnauthorized?: () => void
  /** Called when server is unreachable (network error) */
  onOffline?: () => void
  /** Called when server comes back online */
  onOnline?: () => void
}

// =============================================================================
// API Response Types
// =============================================================================

export interface APIResponse<T = unknown> {
  success: boolean
  data?: T
  error?: string
}

export interface HealthResponse {
  status: string
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

export interface StatsSample {
  stats: SystemStats
  timestamp: string
}

export interface StatsHistoryResponse {
  samples: StatsSample[]
  count: number
  duration: string
}

export interface SystemInfo {
  hostname: string
  os: string
  os_version: string
  kernel: string
  architecture: string
  cpu_model: string
  cpu_cores: number
  total_memory: number
  total_swap: number
  timezone: string
  uptime: number
  boot_time: number
  features: Record<string, boolean>
}

export interface ServerStatus {
  status: 'healthy' | 'degraded'
  version: string
  uptime: number
  timestamp: number
  go: GoRuntimeInfo
  features?: Record<string, boolean>
  checks: Record<string, HealthCheck>
}

export interface GoRuntimeInfo {
  version: string
  num_cpu: number
  goroutines: number
  memory_mb: number
}

export interface HealthCheck {
  status: 'healthy' | 'unhealthy' | 'warning' | 'unavailable'
  message?: string
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

export interface UploadResponse {
  filename: string
}

export interface SearchResult {
  path: string
  name: string
  type: 'file' | 'directory'
  size: number
  match?: string
  line_num?: number
}

export interface SearchOptions {
  /** Starting path for search */
  path: string
  /** Search query */
  query: string
  /** Search within file contents */
  content?: boolean
  /** Maximum results to return */
  limit?: number
}

export interface SearchResponse {
  results: SearchResult[]
  count: number
}

// =============================================================================
// Pinned Folder Types
// =============================================================================

export interface PinnedFolder {
  id: string
  path: string
  name: string
  created_at: string
}

export interface PinnedFoldersResponse {
  folders: PinnedFolder[]
  home_dir: string
}

// =============================================================================
// Job Types
// =============================================================================

export type JobStatus = 'pending' | 'running' | 'stopped' | 'failed' | 'finished'

export interface Job {
  id: string
  command: string
  cwd: string
  status: JobStatus
  pid?: number
  exit_code?: number
  log_file?: string
  created_at: string
  started_at?: string
  finished_at?: string
}

export interface JobsResponse {
  jobs: Job[]
}

export interface JobLogsResponse {
  logs: string[]
}

// =============================================================================
// Package Types
// =============================================================================

export interface PackageManagerInfo {
  manager: string
  available: boolean
}

export interface Package {
  name: string
  version: string
  description?: string
  size?: number
  installed?: boolean
}

export interface PackageDetails extends Package {
  dependencies?: string[]
  homepage?: string
  maintainer?: string
}

// =============================================================================
// Cron Types
// =============================================================================

export interface CronJob {
  id: string
  schedule: string
  command: string
  user?: string
}

export interface CronJobInput {
  schedule: string
  command: string
}

// =============================================================================
// Terminal Types
// =============================================================================

export interface TerminalOptions {
  /** Initial working directory */
  cwd?: string
  /** Auto-reconnect on disconnect (default: true) */
  autoReconnect?: boolean
  /** Max reconnect attempts (default: 5) */
  maxReconnectAttempts?: number
  /** Reconnect delay in ms (default: 1000, with exponential backoff) */
  reconnectDelay?: number
}

export type TerminalState = 'connecting' | 'open' | 'closing' | 'closed' | 'reconnecting'

export interface TerminalEvents {
  open: []
  data: [data: string]
  close: [event: CloseEvent]
  error: [error: Event]
  reconnecting: [attempt: number]
  reconnected: []
  [key: string]: unknown[]
}

// =============================================================================
// Download Types
// =============================================================================

export type DownloadStatus =
  | 'pending'
  | 'downloading'
  | 'paused'
  | 'completed'
  | 'failed'
  | 'cancelled'

export interface ShareLink {
  token: string
  url: string
  expires_at?: string
  created_at: string
}

export interface Download {
  id: string
  url: string
  destination: string
  filename: string
  file_path: string
  status: DownloadStatus
  progress: number
  total: number
  speed: number
  error?: string
  created_at: string
  started_at?: string
  completed_at?: string
  retries: number
  max_retries: number
  share_links?: ShareLink[]
}

export interface DownloadsResponse {
  downloads: Download[]
}

export interface AddDownloadRequest {
  url: string
  destination: string
  filename?: string
}

export interface CreateShareOptions {
  /** Expiry time in seconds (null = never expires) */
  expires_in?: number
}
