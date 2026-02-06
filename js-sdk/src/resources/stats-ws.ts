import { EventEmitter } from '../events'
import type { HttpClient } from '../http'
import type { StatsConnectionEvents, StatsConnectionOptions, StatsConnectionState, SystemStats } from '../types'

const DEFAULT_MAX_RECONNECT_ATTEMPTS = 10
const DEFAULT_RECONNECT_DELAY = 1000
const DEFAULT_MAX_RECONNECT_DELAY = 30000

/**
 * Stats WebSocket connection with auto-reconnect
 * Provides real-time system stats updates
 */
export class StatsConnection extends EventEmitter<StatsConnectionEvents> {
  private ws: WebSocket | null = null
  private _state: StatsConnectionState = 'connecting'
  private reconnectAttempts = 0
  private reconnectTimer?: ReturnType<typeof setTimeout>
  private manualClose = false
  private wasReconnect = false
  private readonly url: string

  private readonly options: {
    autoReconnect: boolean
    maxReconnectAttempts: number
    reconnectDelay: number
    maxReconnectDelay: number
  }

  constructor(url: string, options: StatsConnectionOptions = {}) {
    super()
    this.url = url

    this.options = {
      autoReconnect: options.autoReconnect ?? true,
      maxReconnectAttempts: options.maxReconnectAttempts ?? DEFAULT_MAX_RECONNECT_ATTEMPTS,
      reconnectDelay: options.reconnectDelay ?? DEFAULT_RECONNECT_DELAY,
      maxReconnectDelay: options.maxReconnectDelay ?? DEFAULT_MAX_RECONNECT_DELAY,
    }

    this.setupWebSocket()
  }

  /**
   * Current connection state
   */
  get state(): StatsConnectionState {
    return this._state
  }

  /**
   * Close the connection (disables auto-reconnect)
   */
  close(): void {
    this.manualClose = true
    this.clearReconnectTimer()

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }

    this._state = 'closed'
  }

  /**
   * Manually trigger a reconnect
   */
  reconnect(): void {
    if (this._state === 'closed' || this._state === 'closing') {
      this.manualClose = false
      this.reconnectAttempts = 0
      this.wasReconnect = true
      this.setupWebSocket()
    }
  }

  /**
   * Check if the connection is open
   */
  get isOpen(): boolean {
    return this._state === 'open'
  }

  /**
   * Check if the connection is connecting or reconnecting
   */
  get isConnecting(): boolean {
    return this._state === 'connecting' || this._state === 'reconnecting'
  }

  private setupWebSocket(): void {
    this._state = this.wasReconnect ? 'reconnecting' : 'connecting'

    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        this._state = 'open'
        this.reconnectAttempts = 0

        if (this.wasReconnect) {
          this.emit('reconnected')
          this.wasReconnect = false
        } else {
          this.emit('open')
        }
      }

      this.ws.onclose = (event) => {
        this.emit('close', event)

        // Auto-reconnect if enabled and not manually closed
        if (this.options.autoReconnect && !this.manualClose) {
          this.scheduleReconnect()
        } else {
          this._state = 'closed'
        }
      }

      this.ws.onerror = (error) => {
        this.emit('error', error)
      }

      this.ws.onmessage = (event) => {
        try {
          const stats = JSON.parse(event.data) as SystemStats
          this.emit('stats', stats)
        } catch (error) {
          this.emit('error', error as Event)
        }
      }
    } catch (error) {
      this._state = 'closed'
      this.emit('error', error as Event)
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      this._state = 'closed'
      return
    }

    this._state = 'reconnecting'
    this.reconnectAttempts++

    // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s... (capped at maxReconnectDelay)
    const delay = Math.min(
      this.options.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
      this.options.maxReconnectDelay
    )

    this.emit('reconnecting', this.reconnectAttempts)

    this.reconnectTimer = setTimeout(() => {
      this.wasReconnect = true
      this.setupWebSocket()
    }, delay)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = undefined
    }
  }
}

/**
 * Factory function to create stats WebSocket connections
 * @internal - Used by SystemResource
 */
export function createStatsConnection(http: HttpClient, options?: StatsConnectionOptions): StatsConnection {
  const url = http.buildWebSocketUrl('/system/stats/ws')
  return new StatsConnection(url, options)
}
