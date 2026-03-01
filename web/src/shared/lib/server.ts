import type { ServerData, ServerStats, ServerStatus } from '@/shared/lib/types'
import { GloskiClient } from '@gloski/sdk'
import type { RequestOptions } from '../../../../js-sdk/src/http'

// Single instance per server — created once, mutated in place via valtio proxy
export class Server {
  id: string
  url: string
  name: string
  private apiKey: string | null
  private token: string | null
  private status: ServerStatus
  private client: GloskiClient

  constructor(data: ServerData) {
    this.id = data.id
    this.url = data.url
    this.name = data.name
    this.apiKey = data.apiKey
    this.token = data.token
    this.status = data.status

    this.client = new GloskiClient({
      url: this.url,
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
    return !!(this.apiKey || this.token)
  }

  getStatus(): ServerStatus {
    return this.status
  }

  async getStats(requestOptions?: RequestOptions): Promise<ServerStats> {
    // If not authenticated, return early with error (no need to attempt API call)
    if (!this.isAuthenticated()) {
      return { status: this.getStatus(), stats: null, statsError: 'No authentication configured' }
    }

    try {
      const stats = await this.client.system.getStats(requestOptions)
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
