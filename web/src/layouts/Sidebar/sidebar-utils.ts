import type { SelfTestResult } from '@/shared/tower/types'

export function masterHealthBadge(
  result: SelfTestResult | null,
): { count: number; tone: 'warning' | 'destructive' } | null {
  if (!result) return null
  const failed = result.lines.filter((line) => line.status === 'critical').length
  const warnings = result.lines.filter((line) => line.status === 'warning').length
  if (!failed && !warnings) return null
  return { count: failed + warnings, tone: failed ? 'destructive' : 'warning' }
}
