// =============================================================================
// Branded ID Types
// =============================================================================

declare const __brand: unique symbol
export type Brand<T, B extends string> = T & { readonly [__brand]: B }

export type JobId = Brand<string, 'JobId'>
export type DownloadId = Brand<string, 'DownloadId'>
export type PinnedFolderId = Brand<string, 'PinnedFolderId'>
export type UploadId = Brand<string, 'UploadId'>

/** Cast a string to a JobId */
export const jobId = (id: string) => id as JobId
/** Cast a string to a DownloadId */
export const downloadId = (id: string) => id as DownloadId
/** Cast a string to a PinnedFolderId */
export const pinnedFolderId = (id: string) => id as PinnedFolderId
/** Cast a string to an UploadId */
export const uploadId = (id: string) => id as UploadId

// =============================================================================
// Result Type
// =============================================================================

export type Result<T> = { data: T; error: null } | { data: null; error: import('./errors').GloskiError }

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
  /** Custom fetch implementation (default: globalThis.fetch) */
  fetch?: typeof globalThis.fetch
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
// Middleware Types
// =============================================================================

export interface MiddlewareRequest {
  url: string
  method: string
  headers: Record<string, string>
  body?: string | FormData
  signal?: AbortSignal
}

export type MiddlewareNext = (request: MiddlewareRequest) => Promise<Response>
export type Middleware = (request: MiddlewareRequest, next: MiddlewareNext) => Promise<Response>

// =============================================================================
// Auth Types
// =============================================================================

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
  environment?: Record<string, string>
}

export interface ServerHealthReport {
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

export interface DirectoryListing {
  path: string
  entries: FileEntry[]
}

export interface FileContent {
  content: string
  path: string
}

export interface FileUploadResult {
  filename: string
}

export interface FileSearchMatch {
  path: string
  name: string
  type: 'file' | 'directory'
  size: number
  match?: string
  line_num?: number
}

export interface FileSearchOptions {
  /** Starting path for search */
  path: string
  /** Search query */
  query: string
  /** Search within file contents */
  content?: boolean
  /** Maximum results to return */
  limit?: number
}

export interface FileSearchResponse {
  results: FileSearchMatch[]
  count: number
}

// =============================================================================
// Chunked Upload Types
// =============================================================================

export interface ChunkedUploadInit {
  filename: string
  destination: string
  total_size: number
  chunk_size: number
}

export interface ChunkedUploadInfo {
  upload_id: UploadId
  filename: string
  destination: string
  total_size: number
  chunk_size: number
  total_chunks: number
}

export interface ChunkedUploadChunkResponse {
  chunk_index: number
  status: string
}

export interface ChunkedUploadCompleteRequest {
  upload_id: UploadId
  destination: string
  filename: string
  total_chunks: number
}

export interface ChunkedUploadCompleteResponse {
  status: string
  filename: string
}

// =============================================================================
// Pinned Folder Types
// =============================================================================

export interface PinnedFolder {
  id: PinnedFolderId
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
  id: JobId
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
}

export interface SystemPackage {
  name: string
  version: string
  architecture?: string
  description?: string
  status?: string
}

export interface InstalledPackagesResponse {
  manager: string
  packages: SystemPackage[]
  count: number
}

export interface UpgradeInfo {
  upgradable_count: number
  security_count: number
  packages: SystemPackage[]
}

export interface PackageSearchResponse {
  packages: SystemPackage[]
  count: number
}

// =============================================================================
// Cron Types
// =============================================================================

export type CronScope = 'user' | 'system' | 'all'

export interface CronJob {
  schedule: string
  command: string
  user?: string
  description?: string
  enabled: boolean
  source: string
}

export interface CronJobsResponse {
  jobs: CronJob[]
  count: number
}

export interface NewCronJob {
  schedule: string
  command: string
}

// =============================================================================
// WebSocket Types
// =============================================================================

export interface WebSocketConnectionOptions {
  /** Auto-reconnect on disconnect (default: true) */
  autoReconnect?: boolean
  /** Max reconnect attempts (default: 5) */
  maxReconnectAttempts?: number
  /** Initial reconnect delay in ms (default: 1000, with exponential backoff) */
  reconnectDelay?: number
  /** Max reconnect delay in ms (default: 30000) */
  maxReconnectDelay?: number
}

export type WebSocketConnectionState = 'connecting' | 'open' | 'closing' | 'closed' | 'reconnecting'

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

export interface TerminalEvents {
  open: []
  data: [data: string]
  close: [event: CloseEvent]
  error: [error: Event | Error]
  reconnecting: [attempt: number]
  reconnected: []
}

// =============================================================================
// Stats WebSocket Types
// =============================================================================

export interface StatsConnectionEvents {
  open: []
  stats: [stats: SystemStats]
  close: [event: CloseEvent]
  error: [error: Event | Error]
  reconnecting: [attempt: number]
  reconnected: []
}

// =============================================================================
// Download Types
// =============================================================================

export type DownloadStatus = 'pending' | 'downloading' | 'paused' | 'completed' | 'failed' | 'cancelled'

export interface ShareLink {
  token: string
  url: string
  expires_at?: string
  created_at: string
}

export interface Download {
  id: DownloadId
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

export interface NewDownloadRequest {
  url: string
  destination: string
  filename?: string
}

export interface CreateShareOptions {
  /** Expiry time in seconds (null = never expires) */
  expires_in?: number
}

// =============================================================================
// Download Watcher Types
// =============================================================================

export interface DownloadWatcherOptions {
  /** Polling interval in milliseconds (default: 2000) */
  interval?: number
}

export interface DownloadWatcherEvents {
  /** Emitted when a download's progress changes */
  progress: [download: Download]
  /** Emitted when a download completes */
  completed: [download: Download]
  /** Emitted when a download fails */
  failed: [download: Download]
  /** Emitted on every poll with the full list */
  update: [downloads: Download[]]
  /** Emitted when polling encounters an error */
  error: [error: Error]
}
