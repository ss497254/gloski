import { GloskiClient, unwrap, type RequestOptions, type SystemStats } from '@/shared/api'

export type DeviceStatus = 'online' | 'offline' | 'connecting' | 'unauthorized'

export interface DeviceStats {
  status: DeviceStatus
  stats?: SystemStats | null
  statsLoading?: boolean
  statsError?: string | null
}

export interface DeviceData {
  id: string
  name: string
  url: string
  apiPrefix?: string
  status: DeviceStatus
}

/** One device in the fleet. The master proxies its API, so no browser-side credentials are needed. */
export class DeviceClient {
  id: string
  name: string
  url: string
  apiPrefix?: string
  private status: DeviceStatus
  private client: GloskiClient

  constructor(data: DeviceData) {
    this.id = data.id
    this.name = data.name
    this.url = data.url
    this.apiPrefix = data.apiPrefix
    this.status = data.status

    this.client = new GloskiClient({
      url: this.url,
      apiPrefix: this.apiPrefix,
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

  getStatus(): DeviceStatus {
    return this.status
  }

  setStatus(status: DeviceStatus): void {
    this.status = status
  }

  async getStats(requestOptions?: RequestOptions): Promise<DeviceStats> {
    try {
      const stats = unwrap(await this.client.system.getStats(requestOptions))
      this.status = 'online'
      return { stats, status: 'online', statsError: null }
    } catch (err) {
      if (!(err instanceof Error)) {
        return { status: this.status, stats: null, statsError: 'Unknown error' }
      }
      if (err.name === 'AbortError') {
        return { status: this.status, stats: null, statsError: 'Request cancelled' }
      }
      if (err.message.includes('401')) {
        this.status = 'unauthorized'
        return { status: 'unauthorized', stats: null, statsError: 'Unauthorized' }
      }
      this.status = 'offline'
      return { status: 'offline', stats: null, statsError: err.message }
    }
  }
}
