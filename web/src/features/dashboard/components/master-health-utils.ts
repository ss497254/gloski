import type { SelfTestLine } from '@/shared/tower/types'
import type { StatusTone } from '@/shared/lib/status'

export function groupSections(lines: SelfTestLine[]): Array<{ name: string; lines: SelfTestLine[] }> {
  const order: string[] = []
  const grouped = new Map<string, SelfTestLine[]>()
  for (const line of lines) {
    if (!grouped.has(line.section)) {
      grouped.set(line.section, [])
      order.push(line.section)
    }
    grouped.get(line.section)!.push(line)
  }
  return order.map((name) => ({ name, lines: grouped.get(name)! }))
}

export function sectionSummary(lines: SelfTestLine[]): string {
  const failed = lines.filter((line) => line.status === 'critical').length
  const warnings = lines.filter((line) => line.status === 'warning').length
  if (failed) return `${failed} failed`
  if (warnings) return `${warnings} to check`
  if (lines.every((line) => line.status === 'skipped')) return 'not run yet'
  return 'all clear'
}

export function sectionTone(lines: SelfTestLine[]): StatusTone {
  if (lines.some((line) => line.status === 'critical')) return 'destructive'
  if (lines.some((line) => line.status === 'warning')) return 'warning'
  if (lines.every((line) => line.status === 'skipped')) return 'muted'
  return 'success'
}
