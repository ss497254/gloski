import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type ServerStatus = 'online' | 'offline' | 'connecting' | 'unauthorized'

export interface Server {
  id: string
  url: string
  name: string
  apiKey: string | null
  token: string | null
  status: ServerStatus
}

interface ServersState {
  servers: Server[]
  addServer: (url: string, name: string, apiKey?: string) => string
  removeServer: (id: string) => void
  updateServer: (id: string, data: Partial<Omit<Server, 'id'>>) => void
  getServer: (id: string) => Server | undefined
}

function generateId(): string {
  return Math.random().toString(36).substring(2, 10)
}

export const useServersStore = create<ServersState>()(
  persist(
    (set, get) => ({
      servers: [],

      addServer: (url, name, apiKey) => {
        const id = generateId()
        const server: Server = {
          id,
          url: url.replace(/\/$/, ''), // Remove trailing slash
          name,
          apiKey: apiKey || null,
          token: null,
          status: 'offline',
        }
        set((state) => ({ servers: [...state.servers, server] }))
        return id
      },

      removeServer: (id) => {
        set((state) => ({
          servers: state.servers.filter((s) => s.id !== id),
        }))
      },

      updateServer: (id, data) => {
        set((state) => ({
          servers: state.servers.map((s) =>
            s.id === id ? { ...s, ...data } : s
          ),
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
          ...s,
          token: null,
          status: 'offline' as ServerStatus,
        })),
      }),
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
