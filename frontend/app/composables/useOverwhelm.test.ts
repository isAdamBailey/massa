import { describe, expect, it } from 'vitest'
import type { OverwhelmEntry } from '~/stores/overwhelm'
import { OVERWHELM_ELEVATED_THRESHOLD, useOverwhelmSummary } from './useOverwhelm'

function entry(day: string, overwhelmLevel: number, tags: string[] = []): OverwhelmEntry {
  return {
    day,
    overwhelmLevel,
    tags: tags.map((name, i) => ({ id: `${day}-${i}`, name }))
  }
}

// 2024-01-08 is a Monday; 2024-01-10 is Wednesday same week; 2024-01-01 is
// the previous Monday.
const thisWeekMonday = new Date(2024, 0, 10) // Wed Jan 10 local
const prevWeekDay = '2024-01-02'
const thisWeekDays = ['2024-01-08', '2024-01-09', '2024-01-10'] as const

describe('useOverwhelmSummary', () => {
  it('returns null when there are no entries in the current week', () => {
    const { computeCurrentWeekSummary } = useOverwhelmSummary()
    expect(computeCurrentWeekSummary([entry(prevWeekDay, 8, ['work'])], thisWeekMonday)).toBeNull()
  })

  it('returns the current week average without tags when not elevated', () => {
    const { computeCurrentWeekSummary } = useOverwhelmSummary()
    const summary = computeCurrentWeekSummary(
      [
        entry(thisWeekDays[0], 3, ['work']),
        entry(thisWeekDays[1], 4, ['sleep'])
      ],
      thisWeekMonday
    )

    expect(summary).toEqual({
      average: 3.5,
      count: 2,
      elevated: false,
      topTags: []
    })
  })

  it('treats average equal to the threshold as not elevated', () => {
    const { computeCurrentWeekSummary } = useOverwhelmSummary()
    const summary = computeCurrentWeekSummary(
      [entry(thisWeekDays[0], OVERWHELM_ELEVATED_THRESHOLD, ['work'])],
      thisWeekMonday
    )

    expect(summary?.elevated).toBe(false)
    expect(summary?.topTags).toEqual([])
  })

  it('surfaces the top 2 tags by frequency when average is over the threshold', () => {
    const { computeCurrentWeekSummary } = useOverwhelmSummary()
    const summary = computeCurrentWeekSummary(
      [
        entry(thisWeekDays[0], 6, ['work', 'sleep']),
        entry(thisWeekDays[1], 7, ['work']),
        entry(thisWeekDays[2], 5, ['kids', 'work']),
        // Previous week should not contribute tags or average.
        entry(prevWeekDay, 9, ['travel', 'travel'])
      ],
      thisWeekMonday
    )

    expect(summary?.elevated).toBe(true)
    expect(summary?.average).toBeCloseTo(6, 5)
    expect(summary?.topTags).toEqual([
      { name: 'work', count: 3 },
      { name: 'kids', count: 1 }
    ])
  })

  it('breaks tag frequency ties alphabetically', () => {
    const { computeCurrentWeekSummary } = useOverwhelmSummary()
    const summary = computeCurrentWeekSummary(
      [
        entry(thisWeekDays[0], 8, ['sleep']),
        entry(thisWeekDays[1], 8, ['work'])
      ],
      thisWeekMonday
    )

    expect(summary?.topTags.map(t => t.name)).toEqual(['sleep', 'work'])
  })

  it('elevatedTagParts formats one or two tags for the template', () => {
    const { elevatedTagParts } = useOverwhelmSummary()

    expect(elevatedTagParts([])).toBeNull()
    expect(elevatedTagParts([{ name: 'work', count: 2 }])).toEqual({
      lead: 'Overwhelmed by',
      tags: ['work'],
      trail: 'this week.'
    })
    expect(elevatedTagParts([
      { name: 'work', count: 3 },
      { name: 'sleep', count: 2 }
    ])).toEqual({
      lead: 'Overwhelmed by',
      tags: ['work', 'sleep'],
      trail: 'this week.'
    })
  })
})
