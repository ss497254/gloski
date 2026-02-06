import type { HttpClient, RequestOptions } from '../http'
import type {
  ProcessInfo,
  ServerStatus,
  StatsConnectionOptions,
  StatsHistoryResponse,
  SystemInfo,
  SystemStats,
} from '../types'
import { createStatsConnection, type StatsConnection } from './stats-ws'

/**
 * System information and monitoring resource
 */
export class SystemResource {
  private http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  /**
   * Get server status (health checks, version, uptime, runtime info)
   */
  async getStatus(options?: RequestOptions): Promise<ServerStatus> {
    return this.http.request<ServerStatus>('/system/status', options)
  }

  /**
   * Get current system stats
   */
  async getStats(options?: RequestOptions): Promise<SystemStats> {
    return this.http.request<SystemStats>('/system/stats', options)
  }

  /**
   * Get system stats history
   * @param duration - Duration string (e.g., "5m", "1h")
   */
  async getHistory(duration?: string, options?: RequestOptions): Promise<StatsHistoryResponse> {
    const params = duration ? `?duration=${encodeURIComponent(duration)}` : ''
    return this.http.request<StatsHistoryResponse>(`/system/stats/history${params}`, options)
  }

  /**
   * Get system information
   */
  async getInfo(options?: RequestOptions): Promise<SystemInfo> {
    return this.http.request<SystemInfo>('/system/info', options)
  }

  /**
   * Get list of running processes
   * @param limit - Maximum number of processes to return (default: 100)
   */
  async getProcesses(limit = 100): Promise<ProcessInfo[]> {
    const response = await this.http.request<{ processes: ProcessInfo[] }>(`/system/processes?limit=${limit}`)
    return response.processes
  }

  /**
   * Create a WebSocket connection for real-time stats streaming
   *
   * @example
   * ```typescript
   * const statsConn = client.system.connectStats()
   *
   * statsConn.on('open', () => {
   *   console.log('Connected to stats stream')
   * })
   *
   * statsConn.on('stats', (stats) => {
   *   console.log('CPU:', stats.cpu.usage_percent)
   *   console.log('Memory:', stats.memory.used_percent)
   * })
   *
   * statsConn.on('error', (error) => {
   *   console.error('Stats connection error:', error)
   * })
   *
   * // Later: close the connection
   * statsConn.close()
   * ```
   *
   * @param options - Connection options (auto-reconnect, retry settings)
   * @returns StatsConnection instance with event emitter
   */
  connectStats(options?: StatsConnectionOptions): StatsConnection {
    return createStatsConnection(this.http, options)
  }
}
