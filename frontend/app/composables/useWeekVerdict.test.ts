import { describe, expect, it } from 'vitest'
import type { ActiveEnergyEntry } from '~/stores/activeEnergy'
import type { WeightEntry } from '~/stores/weights'

function weightEntry(recordedAt: string, weightKg: number): WeightEntry {
  return {
    id: recordedAt,
    weightKg,
    recordedAt,
    source: 'manual',
    createdAt: recordedAt,
    updatedAt: recordedAt
  }
}

function energyEntry(day: string, activeEnergyKcal: number): ActiveEnergyEntry {
  return { day, activeEnergyKcal }
}

// 2024-01-01 is a Monday; each following Monday starts the next week.
const week1 = ['2024-01-01T08:00:00Z', '2024-01-02T08:00:00Z']
const week2 = ['2024-01-08T08:00:00Z', '2024-01-09T08:00:00Z']
const week3 = ['2024-01-15T08:00:00Z']
const week4 = ['2024-01-22T08:00:00Z']

describe('useWeekVerdict', () => {
  it('computeWeightTrend returns null with fewer than two weeks of data', () => {
    const { computeWeightTrend } = useWeekVerdict()
    expect(computeWeightTrend([weightEntry(week1[0]!, 80)])).toBeNull()
  })

  it('computeWeightTrend detects a meaningful drop', () => {
    const { computeWeightTrend } = useWeekVerdict()
    const entries = [weightEntry(week1[0]!, 82), weightEntry(week2[0]!, 80)]
    expect(computeWeightTrend(entries)).toBe('down')
  })

  it('computeWeightTrend detects a meaningful rise', () => {
    const { computeWeightTrend } = useWeekVerdict()
    const entries = [weightEntry(week1[0]!, 80), weightEntry(week2[0]!, 82)]
    expect(computeWeightTrend(entries)).toBe('up')
  })

  it('computeWeightTrend treats small changes as steady', () => {
    const { computeWeightTrend } = useWeekVerdict()
    const entries = [weightEntry(week1[0]!, 80), weightEntry(week2[0]!, 80.1)]
    expect(computeWeightTrend(entries)).toBe('steady')
  })

  it('computeWeightTrend reads a slow, consistent loss across weeks as down', () => {
    const { computeWeightTrend } = useWeekVerdict()
    // 0.2 kg/week - too small for any single week-over-week comparison to
    // call, but four weeks of it is real loss.
    const entries = [
      weightEntry(week1[0]!, 80),
      weightEntry(week2[0]!, 79.8),
      weightEntry(week3[0]!, 79.6),
      weightEntry(week4[0]!, 79.4)
    ]
    expect(computeWeightTrend(entries)).toBe('down')
  })

  it('computeWeightTrend holds its read when one week bounces against a falling trend', () => {
    const { computeWeightTrend } = useWeekVerdict()
    // The last two weeks alone would say "up"; the four-week slope doesn't.
    const entries = [
      weightEntry(week1[0]!, 82),
      weightEntry(week2[0]!, 81),
      weightEntry(week3[0]!, 80),
      weightEntry(week4[0]!, 80.4)
    ]
    expect(computeWeightTrend(entries)).toBe('down')
  })

  it('computeWeightTrend spaces weeks by date, so a skipped week is a real gap', () => {
    const { computeWeightTrend } = useWeekVerdict()
    // 0.5 kg over the two weeks between them is 0.25 kg/week, not 0.5.
    const entries = [weightEntry(week1[0]!, 80), weightEntry(week3[0]!, 79.5)]
    expect(computeWeightTrend(entries)).toBe('steady')
  })

  it('computeWeightTrend averages within a week rather than comparing readings', () => {
    const { computeWeightTrend } = useWeekVerdict()
    // A heavy last weigh-in doesn't outvote the week it sits in.
    const entries = [
      weightEntry(week1[0]!, 82),
      weightEntry(week1[1]!, 82),
      weightEntry(week2[0]!, 80),
      weightEntry(week2[1]!, 81.4)
    ]
    expect(computeWeightTrend(entries)).toBe('down')
  })

  it('computeEnergyTrend detects a meaningful increase', () => {
    const { computeEnergyTrend } = useWeekVerdict()
    const entries = [energyEntry(week1[0]!, 200), energyEntry(week2[0]!, 400)]
    expect(computeEnergyTrend(entries)).toBe('up')
  })

  it('computeEnergyTrend treats a small change as steady', () => {
    const { computeEnergyTrend } = useWeekVerdict()
    const entries = [energyEntry(week1[0]!, 200), energyEntry(week2[0]!, 205)]
    expect(computeEnergyTrend(entries)).toBe('steady')
  })

  it('computeEnergyTrend is not dragged down by a partial current week', () => {
    const { computeEnergyTrend } = useWeekVerdict()
    // Two days into a week burning the same per day as the full week before it.
    const entries = [
      energyEntry('2024-01-01', 300),
      energyEntry('2024-01-02', 300),
      energyEntry('2024-01-03', 300),
      energyEntry('2024-01-04', 300),
      energyEntry('2024-01-05', 300),
      energyEntry('2024-01-06', 300),
      energyEntry('2024-01-07', 300),
      energyEntry('2024-01-08', 300),
      energyEntry('2024-01-09', 300)
    ]
    expect(computeEnergyTrend(entries)).toBe('steady')
  })

  it('computeVerdict returns null when both trends are unknown', () => {
    const { computeVerdict } = useWeekVerdict()
    expect(computeVerdict(null, null)).toBeNull()
  })

  it('computeVerdict is "better" when weight is down, even if energy is down too', () => {
    const { computeVerdict } = useWeekVerdict()
    expect(computeVerdict('down', 'down')).toBe('better')
  })

  it('computeVerdict is "better" when energy is up, even if weight is also up', () => {
    const { computeVerdict } = useWeekVerdict()
    expect(computeVerdict('up', 'up')).toBe('better')
  })

  it('computeVerdict is "worse" only when weight is up and energy did not pick up the slack', () => {
    const { computeVerdict } = useWeekVerdict()
    expect(computeVerdict('up', 'down')).toBe('worse')
    expect(computeVerdict('up', 'steady')).toBe('worse')
    expect(computeVerdict('up', null)).toBe('worse')
  })

  it('computeVerdict is "steady" when neither signal moved', () => {
    const { computeVerdict } = useWeekVerdict()
    expect(computeVerdict('steady', 'steady')).toBe('steady')
    expect(computeVerdict('steady', null)).toBe('steady')
    expect(computeVerdict(null, 'steady')).toBe('steady')
  })

  it('verdictLabel maps each verdict to its display label', () => {
    const { verdictLabel } = useWeekVerdict()
    expect(verdictLabel('better')).toBe('Better this week')
    expect(verdictLabel('worse')).toBe('Worse this week')
    expect(verdictLabel('steady')).toBe('Steady this week')
    expect(verdictLabel(null)).toBe('Not enough data yet')
  })
})
