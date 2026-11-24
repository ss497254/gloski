import { describe, expect, test } from 'bun:test'
import { masterHealthBadge } from '../sidebar-utils'
import type { SelfTestLine, SelfTestResult } from '@/shared/tower/types'

function resultWith(statuses: SelfTestLine['status'][]): SelfTestResult {
  return {
    status: 'ok',
    generated_at: '2026-11-24T10:00:00Z',
    lines: statuses.map((status, index) => ({
      key: `line-${index}`,
      section: 'Other',
      title: `Line ${index}`,
      status,
      summary: 'summary',
    })),
  }
}

describe('sidebar master health badge', () => {
  test('no badge while loading or all clear', () => {
    expect(masterHealthBadge(null)).toBeNull()
    expect(masterHealthBadge(resultWith(['ok', 'skipped']))).toBeNull()
  })

  test('skipped lines never raise a badge', () => {
    expect(masterHealthBadge(resultWith(['skipped', 'skipped']))).toBeNull()
  })

  test('warnings count, a failure turns the badge red', () => {
    expect(masterHealthBadge(resultWith(['warning', 'warning', 'ok']))).toEqual({ count: 2, tone: 'warning' })
    expect(masterHealthBadge(resultWith(['warning', 'critical']))).toEqual({ count: 2, tone: 'destructive' })
  })
})
