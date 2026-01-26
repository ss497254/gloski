import type { GloskiClientConfig, HealthResponse } from './types'
import { HttpClient, checkHealth } from './http'
import {
  AuthResource,
  SystemResource,
  FilesResource,
  JobsResource,
  SystemdResource,
  SearchResource,
  TerminalResource,
  PackagesResource,
  CronResource,
  DownloadsResource,
} from './resources'

/**
 * Gloski SDK Client
 *
 * Main entry point for interacting with a Gloski server.
 *
 * @example
 * ```typescript
 * const client = new GloskiClient({
 *   url: 'https://server.example.com',
 *   apiKey: 'your-api-key',
 * })
 *
 * // Get system stats
 * const stats = await client.system.getStats()
 *
 * // List files
 * const files = await client.files.list('/home')
 *
 * // Connect to terminal
 * const term = client.terminal.connect()
 * term.on('data', (data) => console.log(data))
 * term.write('ls -la\n')
 * ```
 */
export class GloskiClient {
  private readonly http: HttpClient

  /** Authentication resource */
  readonly auth: AuthResource

  /** System information and monitoring */
  readonly system: SystemResource

  /** File operations */
  readonly files: FilesResource

  /** Job management (ad-hoc command execution) */
  readonly jobs: JobsResource

  /** Systemd service management */
  readonly systemd: SystemdResource

  /** File search */
  readonly search: SearchResource

  /** Terminal WebSocket connections */
  readonly terminal: TerminalResource

  /** Package management (optional, may not be available) */
  readonly packages: PackagesResource

  /** Cron job management (optional, may not be available) */
  readonly cron: CronResource

  /** Download manager */
  readonly downloads: DownloadsResource

  /**
   * Create a new Gloski client
   * @param config - Client configuration
   */
  constructor(config: GloskiClientConfig) {
    // Normalize URL (remove trailing slash)
    const normalizedConfig: GloskiClientConfig = {
      ...config,
      url: config.url.replace(/\/+$/, ''),
    }

    // Create HTTP client with callbacks
    this.http = new HttpClient(normalizedConfig, {
      onUnauthorized: config.onUnauthorized,
      onOffline: config.onOffline,
      onOnline: config.onOnline,
    })

    // Initialize resources
    this.auth = new AuthResource(this.http)
    this.system = new SystemResource(this.http)
    this.files = new FilesResource(this.http)
    this.jobs = new JobsResource(this.http)
    this.systemd = new SystemdResource(this.http)
    this.search = new SearchResource(this.http)
    this.terminal = new TerminalResource(this.http)
    this.packages = new PackagesResource(this.http)
    this.cron = new CronResource(this.http)
    this.downloads = new DownloadsResource(this.http)
  }

  /**
   * Check server health (static method, no authentication required)
   *
   * @example
   * ```typescript
   * const health = await GloskiClient.checkHealth('https://server.example.com')
   * console.log(health.status) // "ok"
   * ```
   *
   * @param url - Server URL
   * @param timeout - Request timeout in milliseconds (default: 5000)
   */
  static checkHealth(url: string, timeout?: number): Promise<HealthResponse> {
    return checkHealth(url.replace(/\/+$/, ''), timeout)
  }
}
