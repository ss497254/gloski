import { describe, expect, test } from 'bun:test'
import type { FleetDevice } from '@/shared/tower'
import { paginate, sortDevices } from '../table'

const device = (name: string, status: 'online' | 'offline', uptime: number, seen: string): FleetDevice => ({
  id: name,
  name,
  url: '',
  hostname: name,
  platform: '',
  version: '',
  status,
  stats: { uptime } as FleetDevice['stats'],
  created_at: seen,
  last_seen: seen,
  managed: false,
  master: false,
})

const devices = [
  device('b', 'offline', 100, '2026-11-23T10:00:00Z'),
  device('a', 'online', 50, '2026-11-23T11:00:00Z'),
  device('c', 'online', 200, '2026-11-23T09:00:00Z'),
]

describe('sortDevices', () => {
  test('sorts by name by default without mutating the input', () => {
    expect(sortDevices(devices, 'name').map((item) => item.name)).toEqual(['a', 'b', 'c'])
    expect(devices[0].name).toBe('b')
  })

  test('sorts by status then name', () => {
    expect(sortDevices(devices, 'status').map((item) => item.name)).toEqual(['b', 'a', 'c'])
  })

  test('sorts by uptime and last seen, descending', () => {
    expect(sortDevices(devices, 'uptime').map((item) => item.name)).toEqual(['c', 'b', 'a'])
    expect(sortDevices(devices, 'seen').map((item) => item.name)).toEqual(['a', 'b', 'c'])
  })
})

describe('paginate', () => {
  test('slices a page and reports its bounds', () => {
    const page = paginate(devices, 1, 2)
    expect(page.visible.map((item) => item.name)).toEqual(['c'])
    expect(page).toMatchObject({ pageCount: 2, currentPage: 1, first: 3, last: 3 })
  })

  test('clamps an out-of-range page when the list shrinks', () => {
    const page = paginate(devices.slice(0, 1), 9, 2)
    expect(page).toMatchObject({ pageCount: 1, currentPage: 0, first: 1, last: 1 })
  })

  test('handles an empty list', () => {
    expect(paginate([], 0, 10)).toMatchObject({ pageCount: 1, currentPage: 0, first: 0, last: 0 })
  })
})
