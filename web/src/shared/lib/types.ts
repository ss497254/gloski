import type { SystemStats } from '@gloski/sdk'

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
}

export type ServerWithStats = ServerData & ServerStats
