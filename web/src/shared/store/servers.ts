import { Server, type ServerData, type ServerStatus } from '@/shared/lib'
import { proxy } from 'valtio'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface TowerDeviceInput {
  id: string
  name: string
  status: ServerStatus
}

interface ServersState {
  servers: Server[]
  addServer: (url: string, name: string, apiKey?: string) => string
  removeServer: (id: string) => void
  getServer: (id: string) => Server | undefined
  /** Reconcile the virtual server list with devices reported by the tower. */
  syncTowerDevices: (devices: TowerDeviceInput[]) => void
}

export const useServersStore = create<ServersState>()(
  persist(
    (set, get) => ({
      servers: [],

      addServer: (url, name, apiKey) => {
        const id = Math.random().toString(36).substring(2, 10)
        const server = proxy(
          new Server({
            id,
            url: url.replace(/\/$/, ''),
            name,
            apiKey: apiKey || null,
            token: null,
            status: 'offline',
          })
        )
        set((state) => ({ servers: [...state.servers, server] }))
        return id
      },

      removeServer: (id) => {
        set((state) => ({
          servers: state.servers.filter((s) => s.id !== id),
        }))
      },

      getServer: (id) => get().servers.find((s) => s.id === id),

      syncTowerDevices: (devices) => {
        set((state) => {
          const masterURL = typeof window !== 'undefined' ? window.location.origin : ''
          const existing = new Map(state.servers.map((server) => [server.id, server]))
          const towerServers: Server[] = []
          const seen = new Set<string>()

          for (const device of devices) {
            const id = `tower-${device.id}`
            seen.add(id)
            const current = existing.get(id)
            if (current) {
              current.name = device.name
              current.setStatus(device.status)
              towerServers.push(current)
            } else {
              towerServers.push(
                proxy(
                  new Server({
                    id,
                    name: device.name,
                    url: masterURL,
                    apiKey: null,
                    token: null,
                    status: device.status,
                    apiPrefix: `/api/devices/${device.id}/api`,
                    managedBy: 'tower',
                  })
                )
              )
            }
          }

          const manual = state.servers.filter((server) => server.managedBy !== 'tower' && !seen.has(server.id))
          return { servers: [...towerServers, ...manual] }
        })
      },
    }),
    {
      name: 'gloski-servers',
      // Don't persist tokens for security, only persist server configs
      partialize: (state) => ({
        servers: state.servers
          .filter((s) => s.managedBy !== 'tower')
          .map((s) => ({
            id: s.id,
            url: s.url,
            name: s.name,
            apiKey: s['apiKey'],
            token: null,
            status: 'offline' as ServerStatus,
          })),
      }),
      onRehydrateStorage: () => (state) => {
        if (state) {
          state.servers = state.servers.map((s) => proxy(new Server(s as unknown as ServerData)))
        }
      },
    }
  )
)

// Helper to get sorted servers (online first)
export function getSortedServers(servers: Server[]): Server[] {
  const order: Record<ServerStatus, number> = {
    online: 0,
    connecting: 1,
    unauthorized: 2,
    offline: 3,
  }
  return [...servers].sort((a, b) => order[a['status']] - order[b['status']])
}

export { Server }
