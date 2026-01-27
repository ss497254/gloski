import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { GloskiClient } from '@gloski/sdk'

export type ServerStatus = 'online' | 'offline' | 'connecting' | 'unauthorized'

// Base server data (what gets stored/persisted)
interface ServerData {
  id: string
  url: string
  name: string
  apiKey: string | null
  token: string | null
  status: ServerStatus
}

// Server with client accessor method
export interface Server extends ServerData {
  getClient: () => GloskiClient
}

// Client instances cache (not persisted, not part of state)
const clientInstances = new Map<string, GloskiClient>()

// Track auth fingerprint to detect when we need to recreate client
const clientFingerprints = new Map<string, string>()

function getAuthFingerprint(server: ServerData): string {
  return `${server.url}|${server.apiKey || ''}|${server.token || ''}`
}

interface ServersState {
  servers: Server[]
  addServer: (url: string, name: string, apiKey?: string) => string
  removeServer: (id: string) => void
  updateServer: (id: string, data: Partial<Omit<ServerData, 'id'>>) => void
  getServer: (id: string) => Server | undefined
}

function generateId(): string {
  return Math.random().toString(36).substring(2, 10)
}

function createClientForServer(
  server: ServerData,
  updateServer: (id: string, data: Partial<Omit<ServerData, 'id'>>) => void
): GloskiClient {
  return new GloskiClient({
    url: server.url,
    apiKey: server.apiKey ?? undefined,
    token: server.token ?? undefined,
    onUnauthorized: () => {
      updateServer(server.id, { status: 'unauthorized' })
    },
    onOffline: () => {
      updateServer(server.id, { status: 'offline' })
    },
    onOnline: () => {
      updateServer(server.id, { status: 'online' })
    },
  })
}

// Creates a Server object with getClient method from plain data
function createServer(
  data: ServerData,
  updateServer: (id: string, data: Partial<Omit<ServerData, 'id'>>) => void
): Server {
  return {
    ...data,
    getClient() {
      const fingerprint = getAuthFingerprint(data)
      const existingFingerprint = clientFingerprints.get(data.id)

      // Return cached client if fingerprint matches
      if (existingFingerprint === fingerprint && clientInstances.has(data.id)) {
        return clientInstances.get(data.id)!
      }

      // Create new client and cache it
      const client = createClientForServer(data, updateServer)
      clientInstances.set(data.id, client)
      clientFingerprints.set(data.id, fingerprint)

      return client
    },
  }
}

export const useServersStore = create<ServersState>()(
  persist(
    (set, get) => ({
      servers: [],

      addServer: (url, name, apiKey) => {
        const id = generateId()
        const data: ServerData = {
          id,
          url: url.replace(/\/$/, ''), // Remove trailing slash
          name,
          apiKey: apiKey || null,
          token: null,
          status: 'offline',
        }
        const server = createServer(data, get().updateServer)
        set((state) => ({ servers: [...state.servers, server] }))
        return id
      },

      removeServer: (id) => {
        // Clean up client instance
        clientInstances.delete(id)
        clientFingerprints.delete(id)

        set((state) => ({
          servers: state.servers.filter((s) => s.id !== id),
        }))
      },

      updateServer: (id, data) => {
        // If auth-related properties changed, invalidate the cached client
        if ('url' in data || 'apiKey' in data || 'token' in data) {
          clientInstances.delete(id)
          clientFingerprints.delete(id)
        }

        set((state) => ({
          servers: state.servers.map((s) => (s.id === id ? createServer({ ...s, ...data }, get().updateServer) : s)),
        }))
      },

      getServer: (id) => {
        return get().servers.find((s) => s.id === id)
      },
    }),
    {
      name: 'gloski-servers',
      // Don't persist tokens for security, only persist server configs
      partialize: (state) => ({
        servers: state.servers.map((s) => ({
          id: s.id,
          url: s.url,
          name: s.name,
          apiKey: s.apiKey,
          token: null,
          status: 'offline' as ServerStatus,
        })),
      }),
      // Rehydrate servers with getClient method
      onRehydrateStorage: () => (state) => {
        if (state) {
          state.servers = state.servers.map((s) => createServer(s, state.updateServer))
        }
      },
    }
  )
)

// Helper to get sorted servers (online first)
export function getSortedServers(servers: Server[]): Server[] {
  return [...servers].sort((a, b) => {
    const order: Record<ServerStatus, number> = {
      online: 0,
      connecting: 1,
      unauthorized: 2,
      offline: 3,
    }
    return order[a.status] - order[b.status]
  })
}
