// Main client
export { GloskiClient } from './client'

// Error classes, utilities, and type guards
export { GloskiError, ErrorCode, safe, isGloskiError, isNetworkError } from './errors'
export type { ErrorCode as ErrorCodeType } from './errors'

// HTTP types (for request options)
export type { RequestOptions } from './http'

// Middleware and utilities
export { withRetry, paginate } from './middleware'
export type { RetryOptions, PaginateOptions } from './middleware'

// Event emitter (for extending)
export { EventEmitter } from './events'

// Base WebSocket class (for extending)
export { BaseWebSocketConnection, type BaseWebSocketEvents } from './resources'

// Resources (for type inference)
export {
  AuthResource,
  CronResource,
  DownloadsResource,
  DownloadWatcher,
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

// Branded ID helpers
export { jobId, downloadId, pinnedFolderId, uploadId } from './types'

// All types
export type {
  AddDownloadRequest,
  // API response types
  APIResponse,
  // Base WebSocket types
  BaseWebSocketOptions,
  BaseWebSocketState,
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
  // Branded ID types
  Brand,
  DownloadId,
  // Download types
  DownloadStatus,
  // Download watcher types
  DownloadWatcherEvents,
  DownloadWatcherOptions,
  // File types
  FileEntry,
  // Client config
  GloskiClientConfig,
  // Middleware types
  Middleware,
  MiddlewareNext,
  MiddlewareRequest,
  GoRuntimeInfo,
  HealthCheck,
  HealthResponse,
  // Package types
  InstalledPackagesResponse,
  Job,
  JobLogsResponse,
  JobsResponse,
  JobId,
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
  PinnedFolderId,
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
  UploadId,
  UploadResponse,
} from './types'
