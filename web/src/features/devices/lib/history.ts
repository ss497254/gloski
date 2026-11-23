import type { MetricSample } from '@/shared/tower'

export interface HistoryPoint {
  // MetricsChart takes records with a string index signature.
  [key: string]: string | number
  ts: string
  value: number
}

/** Extract one series from a history response, preserving chart order. */
export function pointsForMetric(samples: MetricSample[], metric: string): HistoryPoint[] {
  return samples
    .filter((sample) => sample.metric === metric)
    .map((sample) => ({ ts: sample.ts, value: sample.value }))
}

/**
 * A finite upper bound for a chart whose values are not percentages. Without
 * one, a single spike would rescale the whole axis on every refresh.
 */
export function chartCeiling(points: HistoryPoint[], floor: number): number {
  const peak = points.reduce((highest, point) => Math.max(highest, point.value), 0)
  return Math.max(floor, Math.ceil((peak * 1.2) / floor) * floor)
}
