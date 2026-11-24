import type { DashboardSummary } from '@/shared/tower/types'

export function verdictCopy(counts: DashboardSummary['verdict'] | undefined) {
  if (!counts) return { title: 'Checking…', hint: 'Reading the fleet…' }
  if (counts.critical > 0)
    return {
      title: `${counts.critical} ${counts.critical === 1 ? 'thing' : 'things'} to fix`,
      hint: 'Start with the first item below — it is the most important.',
    }
  if (counts.warning > 0)
    return {
      title: `Mostly healthy · ${counts.warning} to check`,
      hint: 'Nothing is broken; a few things are worth a look when you have time.',
    }
  if (counts.unknown > 0)
    return {
      title: `Nothing failed · ${counts.unknown} not checked`,
      hint: 'Grey means skipped, stale or never run — not a failure.',
    }
  return { title: 'Healthy · nothing to fix', hint: 'Everything checked looks good.' }
}
