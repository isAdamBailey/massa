import { describe, expect, it } from 'vitest'
import type { WeightEntry } from '~/stores/weights'

function entry(recordedAt: string, weightKg: number, bmi?: number): WeightEntry {
  return {
    id: recordedAt,
    weightKg,
    recordedAt,
    bmi,
    source: 'manual',
    createdAt: recordedAt,
    updatedAt: recordedAt
  }
}

describe('useWeeklyAverages', () => {
  it('computeWeeklyAverageBy groups entries into Monday-starting weeks and averages the chosen value', () => {
    const { computeWeeklyAverageBy } = useWeeklyAverages()

    // 2024-01-01 is a Monday; 2024-01-03 is in the same week; 2024-01-08 is
    // the following Monday.
    const entries = [
      entry('2024-01-01T08:00:00Z', 80, 25),
      entry('2024-01-03T08:00:00Z', 82, 27),
      entry('2024-01-08T08:00:00Z', 81, 26)
    ]

    const weeks = computeWeeklyAverageBy(entries, e => e.recordedAt, e => e.bmi)

    expect(weeks).toHaveLength(2)
    expect(weeks[0]!.average).toBeCloseTo(26, 5)
    expect(weeks[0]!.count).toBe(2)
    expect(weeks[1]!.average).toBeCloseTo(26, 5)
    expect(weeks[1]!.count).toBe(1)
  })

  it('computeWeeklyAverageBy skips entries where the value function returns null or undefined', () => {
    const { computeWeeklyAverageBy } = useWeeklyAverages()

    const entries = [
      entry('2024-01-01T08:00:00Z', 80, undefined),
      entry('2024-01-02T08:00:00Z', 82, 27)
    ]

    const weeks = computeWeeklyAverageBy(entries, e => e.recordedAt, e => e.bmi)

    expect(weeks).toHaveLength(1)
    expect(weeks[0]!.average).toBe(27)
    expect(weeks[0]!.count).toBe(1)
  })

  it('computeWeeklyAverages averages weightKg', () => {
    const { computeWeeklyAverages } = useWeeklyAverages()

    const entries = [
      entry('2024-01-01T08:00:00Z', 80),
      entry('2024-01-02T08:00:00Z', 84)
    ]

    const weeks = computeWeeklyAverages(entries)

    expect(weeks).toHaveLength(1)
    expect(weeks[0]!.average).toBe(82)
  })

  it('computeWeeklySumBy sums an arbitrary value grouped by a date extractor', () => {
    const { computeWeeklySumBy } = useWeeklyAverages()

    const items = [
      { day: '2024-01-01', kcal: 200 },
      { day: '2024-01-02', kcal: 150 },
      { day: '2024-01-08', kcal: 300 }
    ]

    const weeks = computeWeeklySumBy(items, i => i.day, i => i.kcal)

    expect(weeks).toHaveLength(2)
    expect(weeks[0]!.total).toBe(350)
    expect(weeks[1]!.total).toBe(300)
  })

  it('computeWeeklyAverageBy averages day-keyed items grouped by a date extractor', () => {
    const { computeWeeklyAverageBy } = useWeeklyAverages()

    // 2024-01-01 is a Monday; 2024-01-03 is in the same week; 2024-01-08 is
    // the following Monday.
    const items = [
      { day: '2024-01-01', level: 2 },
      { day: '2024-01-03', level: 4 },
      { day: '2024-01-08', level: 5 }
    ]

    const weeks = computeWeeklyAverageBy(items, i => i.day, i => i.level)

    expect(weeks).toHaveLength(2)
    expect(weeks[0]!.average).toBe(3)
    expect(weeks[0]!.count).toBe(2)
    expect(weeks[1]!.average).toBe(5)
    expect(weeks[1]!.count).toBe(1)
  })

  it('computeWeeklyAverageBy parses date-only strings as local midnight', () => {
    const { computeWeeklyAverageBy } = useWeeklyAverages()

    // 2024-01-07 is a Sunday, the last day of the week starting 2024-01-01.
    // Parsed as UTC in a negative-offset timezone it would fall back to
    // 2024-01-06 locally and stay in the same week either way, so instead we
    // assert it groups with the Monday-Sunday week rather than sliding into
    // the following Monday-starting week (2024-01-08).
    const items = [
      { day: '2024-01-01', level: 2 },
      { day: '2024-01-07', level: 4 },
      { day: '2024-01-08', level: 10 }
    ]

    const weeks = computeWeeklyAverageBy(items, i => i.day, i => i.level)

    expect(weeks).toHaveLength(2)
    expect(weeks[0]!.average).toBe(3)
    expect(weeks[0]!.count).toBe(2)
    expect(weeks[1]!.average).toBe(10)
    expect(weeks[1]!.count).toBe(1)
  })
})
