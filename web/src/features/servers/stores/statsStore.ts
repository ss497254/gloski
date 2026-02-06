import { createStore } from 'zustand/vanilla'
import type { StatsConnection } from '@gloski/sdk'
import type { SystemStats } from '@/shared/lib/types'
import type { Server } from './servers'

interface StatsState {
  stats: SystemStats | null
  isConnected: boolean
  error: string | null
  connection: StatsConnection | null
  disconnectTimer: number | null
}

interface StatsActions {
  connect: (server: Server, onOnline?: () => void) => void
  disconnect: () => void
}

export type StatsStore = StatsState & StatsActions

/**
 * Creates a Zustand vanilla store for managing stats WebSocket connection
 * This store instance is passed through context to avoid re-renders
 */
export function createStatsStore() {
  return createStore<StatsStore>((set, get) => ({
    // State
    stats: null,
    isConnected: false,
    error: null,
    connection: null,
    disconnectTimer: null,

    // Actions
    connect: (server: Server, onOnline?: () => void) => {
      const { connection: existingConnection, disconnectTimer } = get()

      // Cancel pending disconnect if reconnecting
      if (disconnectTimer) {
        clearTimeout(disconnectTimer)
        set({ disconnectTimer: null })
      }

      // Close existing connection if any
      if (existingConnection) {
        existingConnection.close()
      }

      // Create new connection
      const client = server.getClient()
      const connection = client.system.connectStats({
        autoReconnect: true,
        maxReconnectAttempts: 10,
        reconnectDelay: 1000,
        maxReconnectDelay: 30000,
      })

      // Setup event listeners
      connection.on('open', () => {
        set({ isConnected: true, error: null })
      })

      connection.on('stats', (stats: SystemStats) => {
        set({ stats, error: null })
        onOnline?.()
      })

      connection.on('error', () => {
        set({ error: 'WebSocket connection error', isConnected: false })
      })

      connection.on('close', () => {
        set({ isConnected: false })
      })

      connection.on('reconnecting', (attempt: number) => {
        set({ error: `Reconnecting... (attempt ${attempt})` })
      })

      connection.on('reconnected', () => {
        set({ error: null })
      })

      set({ connection })
    },

    disconnect: () => {
      const { disconnectTimer } = get()

      // Cancel any existing disconnect timer
      if (disconnectTimer) {
        clearTimeout(disconnectTimer)
      }

      // Delay disconnect for 3 seconds to handle remount cases
      const timer = setTimeout(() => {
        const { connection } = get()
        if (connection) {
          connection.close()
          set({ connection: null, isConnected: false, stats: null, disconnectTimer: null })
        }
      }, 3000)

      set({ disconnectTimer: timer })
    },
  }))
}

export type StatsStoreInstance = ReturnType<typeof createStatsStore>
