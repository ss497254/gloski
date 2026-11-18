import type { ServerData, ServerStats, ServerStatus } from '@/shared/lib/types'
import { GloskiClient, unwrap } from '@/shared/api'
import type { RequestOptions } from '@/shared/api'

// Single instance per server — created once, mutated in place via valtio proxy
export class Server {
  id: string
  url: string
  name: string
  apiPrefix?: string
  managedBy?: 'tower'
  private apiKey: string | null
  private token: string | null
  private status: ServerStatus
  private client: GloskiClient

  constructor(data: ServerData) {
    this.id = data.id
    this.url = data.url
    this.name = data.name
    this.apiPrefix = data.apiPrefix
    this.managedBy = data.managedBy
    this.apiKey = data.apiKey
    this.token = data.token
    this.status = data.status

    this.client = new GloskiClient({
      url: this.url,
      apiPrefix: this.apiPrefix,
      apiKey: this.apiKey ?? undefined,
      token: this.token ?? undefined,
      onUnauthorized: () => {
        this.status = 'unauthorized'
      },
      onOffline: () => {
        this.status = 'offline'
      },
      onOnline: () => {
        this.status = 'online'
      },
    })
  }

  getClient(): GloskiClient {
    return this.client
  }

  isAuthenticated(): boolean {
    // Tower-managed devices are authenticated by the master proxy.
    return this.managedBy === 'tower' || !!(this.apiKey || this.token)
  }

  getStatus(): ServerStatus {
    return this.status
  }

  setStatus(status: ServerStatus): void {
    this.status = status
  }

  async getStats(requestOptions?: RequestOptions): Promise<ServerStats> {
    // If not authenticated, return early with error (no need to attempt API call)
    if (!this.isAuthenticated()) {
      return { status: this.getStatus(), stats: null, statsError: 'No authentication configured' }
    }

    try {
      const stats = unwrap(await this.client.system.getStats(requestOptions))
      // If the call succeeded but status is not online, update it to online
      if (this.getStatus() !== 'online') this.status = 'online'

      return { stats, status: 'online', statsError: null }
    } catch (err) {
      if (!(err instanceof Error)) {
        return {
          status: this.getStatus(),
          stats: null,
          statsError: 'Unknown error',
        }
      } else if (err.name === 'AbortError') {
        return {
          status: this.getStatus(),
          stats: null,
          statsError: 'Request cancelled',
        }
      } else if (err.message.includes('401')) {
        this.status = 'unauthorized'
        return {
          status: 'unauthorized',
          stats: null,
          statsError: 'Unauthorized - invalid API key or token',
        }
      } else {
        this.status = 'offline'
        return {
          status: 'offline',
          stats: null,
          statsError: err.message,
        }
      }
    }
  }
}
