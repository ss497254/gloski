import { describe, expect, test } from 'bun:test'
import { groupSections, sectionSummary } from '../master-health-utils'
import type { SelfTestLine } from '@/shared/tower/types'

function line(key: string, section: string, status: SelfTestLine['status']): SelfTestLine {
  return { key, section, status, title: key, summary: key }
}

describe('self-test report sections', () => {
  test('keeps first-seen section order and groups lines', () => {
    const sections = groupSections([
      line('database', 'Data & storage', 'ok'),
      line('local-agent', 'Agents & builds', 'ok'),
      line('size', 'Data & storage', 'ok'),
    ])
    expect(sections.map((section) => section.name)).toEqual(['Data & storage', 'Agents & builds'])
    expect(sections[0].lines.map((item) => item.key)).toEqual(['database', 'size'])
  })

  test('summarises the worst state, not a count of every line', () => {
    const skipped = [line('backups', 'Other', 'skipped'), line('log', 'Other', 'skipped')]
    expect(sectionSummary(skipped)).toBe('not run yet')
    expect(sectionSummary([...skipped, line('runtime', 'Other', 'warning')])).toBe('1 to check')
    expect(sectionSummary([...skipped, line('runtime', 'Other', 'critical')])).toBe('1 failed')
    expect(sectionSummary([line('database', 'Data & storage', 'ok')])).toBe('all clear')
  })
})
