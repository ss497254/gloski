import { describe, expect, test } from 'bun:test'
import type { MetricSample } from '@/shared/tower'
import { chartCeiling, fillGaps, pointsForMetric } from '../history'

const sample = (metric: string, value: number, ts: string): MetricSample => ({
  device_id: 'dev',
  metric,
  value,
  ts,
})

describe('pointsForMetric', () => {
  const samples = [
    sample('cpu', 10, '2026-11-23T10:00:00Z'),
    sample('memory', 60, '2026-11-23T10:00:00Z'),
    sample('cpu', 20, '2026-11-23T10:05:00Z'),
  ]

  test('keeps only the requested series in order', () => {
    expect(pointsForMetric(samples, 'cpu')).toEqual([
      { ts: '2026-11-23T10:00:00Z', value: 10 },
      { ts: '2026-11-23T10:05:00Z', value: 20 },
    ])
  })

  test('returns an empty series for an unknown metric', () => {
    expect(pointsForMetric(samples, 'swap')).toEqual([])
  })
})

describe('fillGaps', () => {
  const fiveMinutes = 5 * 60 * 1000

  test('keeps a dense series unchanged', () => {
    const points = [
      { ts: '2026-11-23T10:00:00Z', value: 1 },
      { ts: '2026-11-23T10:05:00Z', value: 2 },
    ]
    expect(fillGaps(points, fiveMinutes, 100)).toEqual(points)
  })

  test('marks a long gap with a null point and caps the result', () => {
    const points = [
      { ts: '2026-11-23T10:00:00Z', value: 1 },
      { ts: '2026-11-23T10:30:00Z', value: 2 },
    ]
    const filled = fillGaps(points, fiveMinutes, 2)
    expect(filled).toHaveLength(2)
    expect(filled[0].value).toBeNull()
    expect(filled[1].value).toBe(2)
  })
})

describe('chartCeiling', () => {
  test('never drops below the floor', () => {
    expect(chartCeiling([], 100)).toBe(100)
  })

  test('rounds the peak up to a multiple of the floor', () => {
    expect(chartCeiling([{ ts: 'now', value: 250 }], 100)).toBe(300)
  })
})
