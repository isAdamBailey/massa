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

// 2024-01-01 is a Monday; entries on/after 2024-01-08 fall in the next week.
const week1 = ['2024-01-01T08:00:00Z', '2024-01-02T08:00:00Z']
const week2 = ['2024-01-08T08:00:00Z', '2024-01-09T08:00:00Z']

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
