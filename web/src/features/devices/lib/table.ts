import type { FleetDevice } from '@/shared/tower'

/** Sort key used by the devices table controls. */
export type DeviceSort = 'name' | 'status' | 'uptime' | 'seen'

export function sortDevices(devices: FleetDevice[], sort: DeviceSort): FleetDevice[] {
  const list = [...devices]
  list.sort((a, b) => {
    switch (sort) {
      case 'status':
        return a.status.localeCompare(b.status) || a.name.localeCompare(b.name)
      case 'seen':
        return b.last_seen.localeCompare(a.last_seen)
      case 'uptime':
        return (b.stats?.uptime ?? 0) - (a.stats?.uptime ?? 0)
      default:
        return a.name.localeCompare(b.name)
    }
  })
  return list
}

export interface Page<T> {
  visible: T[]
  pageCount: number
  currentPage: number
  first: number
  last: number
}

/** Clamp a requested page to the current data, so a shrinking fleet is safe. */
export function paginate<T>(items: T[], requestedPage: number, pageSize: number): Page<T> {
  const pageCount = Math.max(1, Math.ceil(items.length / pageSize))
  const currentPage = Math.min(Math.max(0, requestedPage), pageCount - 1)
  const from = currentPage * pageSize
  return {
    visible: items.slice(from, from + pageSize),
    pageCount,
    currentPage,
    first: items.length === 0 ? 0 : from + 1,
    last: Math.min(from + pageSize, items.length),
  }
}
