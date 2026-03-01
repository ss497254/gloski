import type { HttpClient } from '../http'
import type { StatsConnectionEvents, SystemStats, WebSocketConnectionOptions } from '../types'
import { WebSocketConnection } from './base-ws'

/**
 * Stats WebSocket connection with auto-reconnect
 * Provides real-time system stats updates
 */
export class StatsConnection extends WebSocketConnection<StatsConnectionEvents> {
  constructor(url: string, options: WebSocketConnectionOptions = {}) {
    super(url, {
      autoReconnect: options.autoReconnect,
      maxReconnectAttempts: options.maxReconnectAttempts ?? 10,
      reconnectDelay: options.reconnectDelay,
      maxReconnectDelay: options.maxReconnectDelay,
    })
  }

  protected handleMessage(event: MessageEvent): void {
    try {
      const stats = JSON.parse(event.data) as SystemStats
      this.emit('stats', stats)
    } catch (error) {
      this.emit('error', error as Event)
    }
  }
}

/**
 * Factory function to create stats WebSocket connections
 * @internal - Used by SystemResource
 */
export function createStatsConnection(http: HttpClient, options?: WebSocketConnectionOptions): StatsConnection {
  const url = http.buildWebSocketUrl('/system/stats/ws')
  return new StatsConnection(url, options)
}
