// Main client
export { GloskiClient } from './client'

// Error classes, utilities, and type guards
export { ErrorCode, GloskiError, isGloskiError, isNetworkError, safe } from './errors'
export type { ErrorCode as ErrorCodeType } from './errors'

// HTTP types (for request options)
export type { RequestOptions } from './http'

// Middleware and utilities
export { paginate, withRetry } from './middleware'
export type { PaginateOptions, RetryOptions } from './middleware'

// Event emitter (for extending)
export { EventEmitter } from './events'

// WebSocket base class (for extending)
export { WebSocketConnection, type WebSocketConnectionEvents } from './resources'

// Resources (for type inference)
export {
  AuthResource,
  CronResource,
  DownloadsResource,
  DownloadWatcher,
  FilesResource,
  JobsResource,
  PackagesResource,
  PinnedFoldersResource,
  SearchResource,
  StatsConnection,
  SystemResource,
  TerminalConnection,
  TerminalResource,
  type TransferProgressCallback,
} from './resources'

// Branded ID helpers
export { downloadId, jobId, pinnedFolderId, uploadId } from './types'

// All types
export type {
  // API envelope
  APIResponse,
  AuthStatus,
  // Branded ID types
  Brand,
  // Chunked upload types
  ChunkedUploadChunkResponse,
  ChunkedUploadCompleteRequest,
  ChunkedUploadCompleteResponse,
  ChunkedUploadInfo,
  ChunkedUploadInit,
  // System types
  CPUStats,
  // Download types
  CreateShareOptions,
  // Cron types
  CronJob,
  CronJobsResponse,
  CronScope,
  // File types
  DirectoryListing,
  DiskStats,
  Download,
  DownloadId,
  DownloadsResponse,
  DownloadStatus,
  DownloadWatcherEvents,
  DownloadWatcherOptions,
  FileContent,
  FileEntry,
  FileSearchMatch,
  FileSearchOptions,
  FileSearchResponse,
  FileUploadResult,
  // Client config
  GloskiClientConfig,
  GoRuntimeInfo,
  HealthCheck,
  HealthResponse,
  // Package types
  InstalledPackagesResponse,
  // Job types
  Job,
  JobId,
  JobLogsResponse,
  JobsResponse,
  JobStatus,
  LoadAvg,
  MemoryStats,
  // Middleware types
  Middleware,
  MiddlewareNext,
  MiddlewareRequest,
  NetworkInterface,
  NetworkStats,
  NewCronJob,
  NewDownloadRequest,
  PackageManagerInfo,
  PackageSearchResponse,
  // Pinned folder types
  PinnedFolder,
  PinnedFolderId,
  PinnedFoldersResponse,
  ProcessInfo,
  // Result type
  Result,
  ServerHealthReport,
  ShareLink,
  StatsConnectionEvents,
  StatsHistoryResponse,
  StatsSample,
  SwapStats,
  SystemInfo,
  SystemPackage,
  SystemStats,
  // Terminal types
  TerminalEvents,
  TerminalOptions,
  UpgradeInfo,
  // Upload types
  UploadId,
  // WebSocket types
  WebSocketConnectionOptions,
  WebSocketConnectionState,
} from './types'
