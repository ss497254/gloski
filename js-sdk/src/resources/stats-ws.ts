import type { HttpClient } from '../http'
import type { StatsConnectionEvents, StatsConnectionOptions, SystemStats } from '../types'
import { BaseWebSocketConnection } from './base-ws'

/**
 * Stats WebSocket connection with auto-reconnect
 * Provides real-time system stats updates
 */
export class StatsConnection extends BaseWebSocketConnection<StatsConnectionEvents> {
  constructor(url: string, options: StatsConnectionOptions = {}) {
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
export function createStatsConnection(http: HttpClient, options?: StatsConnectionOptions): StatsConnection {
  const url = http.buildWebSocketUrl('/system/stats/ws')
  return new StatsConnection(url, options)
}
