import { describe, expect, test } from 'bun:test'
import { verdictCopy } from '../FleetHealth'

describe('verdictCopy', () => {
  test('critical beats every other count', () => {
    const copy = verdictCopy({ ok: 8, warning: 2, critical: 1, unknown: 3 })
    expect(copy.title).toBe('1 thing to fix')
    expect(copy.hint).toContain('first item')
  })

  test('unknown is not reported as healthy', () => {
    const copy = verdictCopy({ ok: 2, warning: 0, critical: 0, unknown: 4 })
    expect(copy.title).toBe('Nothing failed · 4 not checked')
    expect(copy.hint).toContain('not a failure')
  })

  test('only a fully known, clean fleet is healthy', () => {
    expect(verdictCopy({ ok: 4, warning: 0, critical: 0, unknown: 0 }).title).toBe('Healthy · nothing to fix')
  })
})
