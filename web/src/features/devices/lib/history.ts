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

/** One chart point; a null value marks a gap where no sample was recorded. */
export interface GapPoint {
  [key: string]: string | number | null
  ts: string
  value: number | null
}

/**
 * Insert a null marker when the gap between samples is much larger than the
 * sampling interval, so a chart does not draw a continuous line across time
 * where no data was recorded. The result is capped to the newest `cap` points.
 */
export function fillGaps(points: HistoryPoint[], intervalMs: number, cap: number): GapPoint[] {
  const result: GapPoint[] = []
  for (let i = 0; i < points.length; i++) {
    if (i > 0) {
      const previous = Date.parse(points[i - 1].ts)
      const current = Date.parse(points[i].ts)
      if (Number.isFinite(previous) && Number.isFinite(current) && current - previous > intervalMs * 1.5) {
        result.push({ ts: points[i - 1].ts, value: null })
      }
    }
    result.push({ ts: points[i].ts, value: points[i].value })
  }
  return result.slice(-cap)
}

/** A finite axis bound so one spike cannot rescale a non-percentage chart. */
export function chartCeiling(points: Array<{ value: number | null }>, floor: number): number {
  const peak = points.reduce((highest, point) => Math.max(highest, point.value ?? 0), 0)
  return Math.max(floor, Math.ceil((peak * 1.2) / floor) * floor)
}
