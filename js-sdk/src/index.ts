// Main client
export { GloskiClient } from './client'

// Error classes and utilities
export { GloskiError, safe } from './errors'

// Event emitter (for extending)
export { EventEmitter } from './events'

// Resources (for type inference)
export {
  AuthResource,
  CronResource,
  DownloadsResource,
  FilesResource,
  JobsResource,
  PackagesResource,
  PinnedSubResource,
  SearchResource,
  StatsConnection,
  SystemResource,
  TerminalConnection,
  TerminalResource,
  type ProgressCallback,
} from './resources'

// All types
export type {
  AddDownloadRequest,
  // API response types
  APIResponse,
  // Result type
  Result,
  AuthStatus,
  // Chunked upload types
  ChunkedUploadChunkResponse,
  ChunkedUploadCompleteRequest,
  ChunkedUploadCompleteResponse,
  ChunkedUploadInfo,
  ChunkedUploadInit,
  CPUStats,
  CreateShareOptions,
  // Cron types
  CronJob,
  CronJobInput,
  CronJobsResponse,
  CronScope,
  DiskStats,
  Download,
  DownloadsResponse,
  // Download types
  DownloadStatus,
  // File types
  FileEntry,
  // Client config
  GloskiClientConfig,
  GoRuntimeInfo,
  HealthCheck,
  HealthResponse,
  // Package types
  InstalledPackagesResponse,
  Job,
  JobLogsResponse,
  JobsResponse,
  // Job types
  JobStatus,
  ListResponse,
  LoadAvg,
  MemoryStats,
  NetworkInterface,
  NetworkStats,
  Package,
  PackageManagerInfo,
  PackageSearchResponse,
  // Pinned folder types
  PinnedFolder,
  PinnedFoldersResponse,
  ProcessInfo,
  ReadResponse,
  SearchOptions,
  SearchResponse,
  SearchResult,
  ServerHealthReport,
  ShareLink,
  StatsConnectionEvents,
  // Stats WebSocket types
  StatsConnectionOptions,
  StatsConnectionState,
  StatsHistoryResponse,
  StatsSample,
  SwapStats,
  SystemInfo,
  // System types
  SystemStats,
  TerminalEvents,
  // Terminal types
  TerminalOptions,
  TerminalState,
  UpgradeInfo,
  UploadResponse,
} from './types'
