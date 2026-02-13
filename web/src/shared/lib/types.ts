// Re-export all types from SDK
export type {
  AddDownloadRequest,
  // API response types
  APIResponse,
  AuthStatus,
  CPUStats,
  CreateShareOptions,
  // Cron types
  CronJob,
  CronJobInput,
  DiskStats,
  Download,
  DownloadsResponse,
  // Download types
  DownloadStatus,
  // File types
  FileEntry,
  GoRuntimeInfo,
  HealthCheck,
  HealthResponse,
  Job,
  JobLogsResponse,
  JobsResponse,
  // Job types
  JobStatus,
  ListResponse,
  LoadAvg,
  // Auth types
  LoginResponse,
  MemoryStats,
  NetworkInterface,
  NetworkStats,
  Package,
  PackageDetails,
  // Package types
  PackageManagerInfo,
  ProcessInfo,
  ReadResponse,
  SearchOptions,
  SearchResponse,
  SearchResult,
  ServerStatus,
  ShareLink,
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
  UploadResponse,
} from '@gloski/sdk'

// App-level types used across features
import type { Server } from '@/features/servers/stores/servers'
import type { SystemStats } from '@gloski/sdk'

export interface ServerWithStats extends Server {
  stats?: SystemStats | null
  statsLoading?: boolean
  statsError?: string | null
}
