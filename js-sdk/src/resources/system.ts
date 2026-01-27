import type { HttpClient } from '../http'
import type {
  SystemStats,
  StatsHistoryResponse,
  SystemInfo,
  ProcessInfo,
  ServerStatus,
} from '../types'

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
  async getStatus(): Promise<ServerStatus> {
    return this.http.request<ServerStatus>('/system/status')
  }

  /**
   * Get current system stats
   */
  async getStats(): Promise<SystemStats> {
    return this.http.request<SystemStats>('/system/stats')
  }

  /**
   * Get system stats history
   * @param duration - Duration string (e.g., "5m", "1h")
   */
  async getHistory(duration?: string): Promise<StatsHistoryResponse> {
    const params = duration ? `?duration=${encodeURIComponent(duration)}` : ''
    return this.http.request<StatsHistoryResponse>(`/system/stats/history${params}`)
  }

  /**
   * Get system information
   */
  async getInfo(): Promise<SystemInfo> {
    return this.http.request<SystemInfo>('/system/info')
  }

  /**
   * Get list of running processes
   * @param limit - Maximum number of processes to return (default: 100)
   */
  async getProcesses(limit = 100): Promise<ProcessInfo[]> {
    const response = await this.http.request<{ processes: ProcessInfo[] }>(
      `/system/processes?limit=${limit}`
    )
    return response.processes
  }
}
