import type { SystemStats } from '@/shared/api'

export interface ServerStats {
  status: ServerStatus
  stats?: SystemStats | null
  statsLoading?: boolean
  statsError?: string | null
}
export type ServerStatus = 'online' | 'offline' | 'connecting' | 'unauthorized'

// Plain data shape (what gets persisted)
export interface ServerData {
  id: string
  url: string
  name: string
  apiKey: string | null
  token: string | null
  status: ServerStatus
  /** API path prefix, e.g. /api/devices/<id>/api when proxied by the tower */
  apiPrefix?: string
  /** Devices discovered through the control tower are rebuilt on each load */
  managedBy?: 'tower'
}

export type ServerWithStats = ServerData & ServerStats
