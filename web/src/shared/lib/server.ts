import { GloskiClient } from '@gloski/sdk'

export type ServerStatus = 'online' | 'offline' | 'connecting' | 'unauthorized'

// Plain data shape (what gets persisted)
export interface ServerData {
  id: string
  url: string
  name: string
  apiKey: string | null
  token: string | null
  status: ServerStatus
}

// Single instance per server — created once, mutated in place via valtio proxy
export class Server {
  id: string
  url: string
  name: string
  private apiKey: string | null
  private token: string | null
  private status: ServerStatus

  private client: GloskiClient | null = null
  private fingerprint: string | null = null

  constructor(data: ServerData) {
    this.id = data.id
    this.url = data.url
    this.name = data.name
    this.apiKey = data.apiKey
    this.token = data.token
    this.status = data.status
  }

  getClient(): GloskiClient {
    const fp = `${this.url}|${this.apiKey || ''}|${this.token || ''}`

    if (this.fingerprint === fp && this.client) {
      return this.client
    }

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

    this.fingerprint = fp

    return this.client
  }

  isAuthenticated(): boolean {
    return !!(this.apiKey || this.token)
  }

  getStatus(): ServerStatus {
    return this.status
  }
}
