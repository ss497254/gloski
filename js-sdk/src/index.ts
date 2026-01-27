// Main client
export { GloskiClient } from './client'

// Error classes
export { GloskiError } from './errors'

// Event emitter (for extending)
export { EventEmitter } from './events'

// Resources (for type inference)
export {
  AuthResource,
  SystemResource,
  FilesResource,
  PinnedSubResource,
  JobsResource,
  SearchResource,
  TerminalResource,
  TerminalConnection,
  PackagesResource,
  CronResource,
  DownloadsResource,
  type ProgressCallback,
} from './resources'

// All types
export type {
  // Client config
  GloskiClientConfig,

  // API response types
  APIResponse,
  HealthResponse,

  // Auth types
  LoginResponse,
  AuthStatus,

  // System types
  SystemStats,
  CPUStats,
  MemoryStats,
  SwapStats,
  DiskStats,
  NetworkStats,
  NetworkInterface,
  LoadAvg,
  ProcessInfo,
  StatsSample,
  StatsHistoryResponse,
  SystemInfo,
  ServerStatus,
  GoRuntimeInfo,
  HealthCheck,

  // File types
  FileEntry,
  ListResponse,
  ReadResponse,
  UploadResponse,
  SearchResult,
  SearchOptions,
  SearchResponse,

  // Job types
  JobStatus,
  Job,
  JobsResponse,
  JobLogsResponse,

  // Package types
  PackageManagerInfo,
  Package,
  PackageDetails,

  // Cron types
  CronJob,
  CronJobInput,

  // Terminal types
  TerminalOptions,
  TerminalState,
  TerminalEvents,

  // Download types
  DownloadStatus,
  ShareLink,
  Download,
  DownloadsResponse,
  AddDownloadRequest,
  CreateShareOptions,

  // Pinned folder types
  PinnedFolder,
  PinnedFoldersResponse,
} from './types'
