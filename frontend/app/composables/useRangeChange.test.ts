import { describe, expect, it } from 'vitest'
import type { ActiveEnergyEntry } from '~/stores/activeEnergy'
import type { OverwhelmEntry } from '~/stores/overwhelm'
import type { WeightEntry } from '~/stores/weights'
import { useRangeChange } from './useRangeChange'

const NOW = new Date(2024, 5, 30, 12, 0, 0) // Jun 30 2024, local noon

/** Local date `daysAgo` days before NOW, at the given hour. */
function daysAgo(days: number, hour = 8): Date {
  const date = new Date(NOW)
  date.setDate(date.getDate() - days)
  date.setHours(hour, 0, 0, 0)
  return date
}

function dayKey(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function weight(days: number, weightKg: number, bmi?: number): WeightEntry {
  const recordedAt = daysAgo(days).toISOString()
  return { id: recordedAt, weightKg, bmi, recordedAt, source: 'manual', createdAt: recordedAt, updatedAt: recordedAt }
}

function energy(days: number, activeEnergyKcal: number): ActiveEnergyEntry {
  return { day: dayKey(daysAgo(days)), activeEnergyKcal }
}

function overwhelm(days: number, overwhelmLevel: number): OverwhelmEntry {
  return { day: dayKey(daysAgo(days)), overwhelmLevel, tags: [] }
}

function sources(partial: { weights?: WeightEntry[], energy?: ActiveEnergyEntry[], overwhelm?: OverwhelmEntry[] }) {
  return { weights: partial.weights ?? [], energy: partial.energy ?? [], overwhelm: partial.overwhelm ?? [] }
}

/** Daily weights over the last 90 days: 90 kg at the start, sliding to 88 kg at the end. */
function ninetyDaySlide(): WeightEntry[] {
  return Array.from({ length: 90 }, (_, i) => {
    const days = 89 - i
    return weight(days, 90 - (2 * i) / 89, 25 - (0.5 * i) / 89)
  })
}

const NOT_ENOUGH = { headline: [{ text: 'Not enough readings in this span yet.' }], detail: null }

describe('useRangeChange', () => {
  describe('not enough data', () => {
    it('reports an empty span', () => {
      const { buildRangeChange } = useRangeChange()
      for (const metric of ['weight', 'bmi', 'energy', 'overwhelm'] as const) {
        expect(buildRangeChange(metric, sources({}), 90, 'metric', NOW)).toEqual(NOT_ENOUGH)
      }
    })

    it('reports a single reading', () => {
      const { buildRangeChange } = useRangeChange()
      expect(buildRangeChange('weight', sources({ weights: [weight(3, 80)] }), 90, 'metric', NOW)).toEqual(NOT_ENOUGH)
    })

    it('reports readings that all fall on one day', () => {
      const { buildRangeChange } = useRangeChange()
      const sameDay = [weight(3, 80), { ...weight(3, 79), recordedAt: daysAgo(3, 20).toISOString() }]
      expect(buildRangeChange('weight', sources({ weights: sameDay }), 90, 'metric', NOW)).toEqual(NOT_ENOUGH)
    })

    it('ignores readings outside the window', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(100, 85), weight(95, 84), weight(5, 80)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 30, 'metric', NOW)).toEqual(NOT_ENOUGH)
    })

    it('skips weight entries without a BMI for the BMI summary', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(10, 80, 24), weight(5, 79)]
      expect(buildRangeChange('bmi', sources({ weights: entries }), 30, 'metric', NOW)).toEqual(NOT_ENOUGH)
    })
  })

  describe('weight', () => {
    it('reports a drop with smoothed 7-day endpoints in metric', () => {
      const { buildRangeChange } = useRangeChange()
      expect(buildRangeChange('weight', sources({ weights: ninetyDaySlide() }), 90, 'metric', NOW)).toEqual({
        headline: [{ text: 'Down 1.8 kg', highlight: true }, { text: ' over the last 90 days' }],
        detail: '89.9 → 88.1 kg · 7-day averages'
      })
    })

    it('converts to pounds for imperial users', () => {
      const { buildRangeChange } = useRangeChange()
      expect(buildRangeChange('weight', sources({ weights: ninetyDaySlide() }), 90, 'imperial', NOW)).toEqual({
        headline: [{ text: 'Down 4.1 lb', highlight: true }, { text: ' over the last 90 days' }],
        detail: '198.3 → 194.2 lb · 7-day averages'
      })
    })

    it('reports a rise with neutral wording', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(6, 80), weight(4, 80.5), weight(1, 81)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 7, 'metric', NOW)).toEqual({
        headline: [{ text: 'Up 1.0 kg', highlight: true }, { text: ' over the last 7 days' }],
        detail: '80.0 → 81.0 kg · first and last readings'
      })
    })

    it('compares raw first and last readings on the 7-day span', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(6, 82), weight(5, 80), weight(2, 80), weight(1, 81)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 7, 'metric', NOW).detail)
        .toBe('82.0 → 81.0 kg · first and last readings')
    })

    it('smooths away a single outlier weigh-in at the end of a long span', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [
        ...Array.from({ length: 7 }, (_, i) => weight(29 - i, 80)),
        ...Array.from({ length: 6 }, (_, i) => weight(6 - i, 80)),
        weight(0, 81.5)
      ]
      const summary = buildRangeChange('weight', sources({ weights: entries }), 30, 'metric', NOW)
      // The 81.5 kg spike is one of seven readings in the closing week, so it moves the end by 0.2.
      expect(summary.headline[0]).toEqual({ text: 'Up 0.2 kg', highlight: true })
      expect(summary.detail).toBe('80.0 → 80.2 kg · 7-day averages')
    })

    it('falls back to first and last readings when data is too sparse for two 7-day groups', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(10, 80), weight(8, 79.5), weight(1, 79)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 90, 'metric', NOW).detail)
        .toBe('80.0 → 79.0 kg · first and last readings')
    })

    it('holds steady under the threshold but still shows the exact change', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(6, 80), weight(1, 79.9)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 7, 'metric', NOW)).toEqual({
        headline: [{ text: 'Holding steady', highlight: true }, { text: ' over the last 7 days' }],
        detail: '80.0 → 79.9 kg · down 0.1 · first and last readings'
      })
    })

    it('uses the pound threshold for imperial users', () => {
      const { buildRangeChange } = useRangeChange()
      // 0.2 kg ≈ 0.44 lb: steady in pounds, though it would clear the kg threshold.
      const entries = [weight(6, 80), weight(1, 80.2)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 7, 'imperial', NOW).headline[0])
        .toEqual({ text: 'Holding steady', highlight: true })
      expect(buildRangeChange('weight', sources({ weights: entries }), 7, 'metric', NOW).headline[0])
        .toEqual({ text: 'Up 0.2 kg', highlight: true })
    })

    it('says "no change" when the endpoints match', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(6, 80), weight(1, 80)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 7, 'metric', NOW).detail)
        .toBe('80.0 → 80.0 kg · no change · first and last readings')
    })

    it('says "since" when the first reading starts well after the window', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(20, 80), weight(1, 79)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 90, 'metric', NOW).headline)
        .toEqual([{ text: 'Down 1.0 kg', highlight: true }, { text: ' since Jun 10' }])
    })

    it('keeps "over the last" when the first reading is within a week of the window start', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(85, 80), weight(1, 79)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 90, 'metric', NOW).headline[1])
        .toEqual({ text: ' over the last 90 days' })
    })

    it('names the 6-month and 1-year spans', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(180, 80), weight(1, 79)]
      expect(buildRangeChange('weight', sources({ weights: entries }), 182, 'metric', NOW).headline[1])
        .toEqual({ text: ' over the last 6 months' })
      const yearEntries = [weight(360, 80), weight(1, 79)]
      expect(buildRangeChange('weight', sources({ weights: yearEntries }), 365, 'metric', NOW).headline[1])
        .toEqual({ text: ' over the last year' })
    })

    it('reads All as "since" the first reading, with the year when it is not this year', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(400, 85), weight(1, 80)]
      expect(buildRangeChange('weight', sources({ weights: entries }), null, 'metric', NOW).headline[1])
        .toEqual({ text: ` since ${daysAgo(400).toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}` })
    })
  })

  describe('bmi', () => {
    it('reports a labelled change without a unit', () => {
      const { buildRangeChange } = useRangeChange()
      expect(buildRangeChange('bmi', sources({ weights: ninetyDaySlide() }), 90, 'imperial', NOW)).toEqual({
        headline: [{ text: 'BMI down 0.5', highlight: true }, { text: ' over the last 90 days' }],
        detail: '25.0 → 24.5 · 7-day averages'
      })
    })

    it('holds steady under 0.1', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [weight(6, 80, 24.62), weight(1, 80, 24.6)]
      expect(buildRangeChange('bmi', sources({ weights: entries }), 7, 'metric', NOW)).toEqual({
        headline: [{ text: 'BMI holding steady', highlight: true }, { text: ' over the last 7 days' }],
        detail: '24.6 → 24.6 · no change · first and last readings'
      })
    })
  })

  describe('energy', () => {
    it('compares the daily average with the previous span of the same length', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [
        ...Array.from({ length: 30 }, (_, i) => energy(59 - i, 380)),
        ...Array.from({ length: 30 }, (_, i) => energy(29 - i, 430))
      ]
      expect(buildRangeChange('energy', sources({ energy: entries }), 30, 'metric', NOW)).toEqual({
        headline: [{ text: 'Averaging 430 kcal a day', highlight: true }, { text: ' over the last 30 days' }],
        detail: 'Up from 380 in the 30 days before · 12,900 kcal total'
      })
    })

    it('reports a lower average than the previous span', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [energy(10, 600), energy(9, 600), energy(5, 400), energy(2, 400)]
      expect(buildRangeChange('energy', sources({ energy: entries }), 7, 'metric', NOW).detail)
        .toBe('Down from 600 in the 7 days before · 800 kcal total')
    })

    it('says "same as" when the averages match', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [energy(10, 400), energy(9, 400), energy(5, 400), energy(2, 400)]
      expect(buildRangeChange('energy', sources({ energy: entries }), 7, 'metric', NOW).detail)
        .toBe('Same as the 7 days before · 800 kcal total')
    })

    it('falls back to start vs end when the previous span has no data', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [energy(6, 300), energy(4, 400), energy(1, 500)]
      expect(buildRangeChange('energy', sources({ energy: entries }), 7, 'metric', NOW)).toEqual({
        headline: [{ text: 'Averaging 400 kcal a day', highlight: true }, { text: ' over the last 7 days' }],
        detail: 'Up from 300 at the start · 1,200 kcal total'
      })
    })

    it('falls back to start vs end for All', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [
        ...Array.from({ length: 7 }, (_, i) => energy(40 - i, 500)),
        ...Array.from({ length: 7 }, (_, i) => energy(6 - i, 300))
      ]
      expect(buildRangeChange('energy', sources({ energy: entries }), null, 'metric', NOW).detail)
        .toBe('Down from 500 at the start · 5,600 kcal total')
    })

    it('falls back to start vs end when the previous span has a single reading', () => {
      const { buildRangeChange } = useRangeChange()
      // What a from=-filtered fetch returns: just the boundary day before the window.
      const entries = [energy(7, 900), energy(6, 300), energy(1, 500)]
      expect(buildRangeChange('energy', sources({ energy: entries }), 7, 'metric', NOW).detail)
        .toBe('Up from 300 at the start · 800 kcal total')
    })

    it('treats day keys as local calendar days', () => {
      const { buildRangeChange } = useRangeChange()
      // The 7-day window is today and the 6 days before it, so 7 days back is the previous span.
      const entries = [energy(8, 900), energy(7, 900), energy(6, 400), energy(1, 400)]
      expect(buildRangeChange('energy', sources({ energy: entries }), 7, 'metric', NOW).detail)
        .toBe('Down from 900 in the 7 days before · 800 kcal total')
    })
  })

  describe('overwhelm', () => {
    it('reports the average against the baseline and the start', () => {
      const { buildRangeChange } = useRangeChange()
      const entries = [
        ...Array.from({ length: 7 }, (_, i) => overwhelm(29 - i, 4.5)),
        ...Array.from({ length: 16 }, (_, i) => overwhelm(22 - i, 3.8)),
        ...Array.from({ length: 7 }, (_, i) => overwhelm(6 - i, 3.2))
      ]
      // mean = (7 * 4.5 + 16 * 3.8 + 7 * 3.2) / 30 ≈ 3.82
      expect(buildRangeChange('overwhelm', sources({ overwhelm: entries }), 30, 'metric', NOW)).toEqual({
        headline: [{ text: 'Averaging 3.8', highlight: true }, { text: ' over the last 30 days' }],
        detail: '0.8 above your baseline of 3 · down from 4.5 at the start'
      })
    })

    it('reads below and at the baseline', () => {
      const { buildRangeChange } = useRangeChange()
      const below = [overwhelm(6, 2), overwhelm(1, 3)]
      expect(buildRangeChange('overwhelm', sources({ overwhelm: below }), 7, 'metric', NOW).detail)
        .toBe('0.5 below your baseline of 3 · up from 2.0 at the start')
      const at = [overwhelm(6, 3), overwhelm(1, 3)]
      expect(buildRangeChange('overwhelm', sources({ overwhelm: at }), 7, 'metric', NOW).detail)
        .toBe('At your baseline of 3 · same as at the start')
    })
  })

  it('does not depend on units for anything but weight', () => {
    const { buildRangeChange } = useRangeChange()
    const all = sources({ weights: ninetyDaySlide(), energy: [energy(6, 300), energy(1, 500)], overwhelm: [overwhelm(6, 2), overwhelm(1, 3)] })
    for (const metric of ['bmi', 'energy', 'overwhelm'] as const) {
      expect(buildRangeChange(metric, all, 90, 'imperial', NOW)).toEqual(buildRangeChange(metric, all, 90, 'metric', NOW))
    }
  })
})
