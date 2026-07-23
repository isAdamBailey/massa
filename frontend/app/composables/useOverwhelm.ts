import { startOfWeek } from 'date-fns'
import type { OverwhelmEntry } from '~/stores/overwhelm'

/**
 * Overwhelm scale baseline (1-10). Mirrors backend/internal/overwhelm.Baseline.
 * Readings above are more overwhelmed than usual; below, less.
 */
export const OVERWHELM_BASELINE = 3

/**
 * Weekly average above this threshold surfaces the week's top tags on the
 * dashboard — a quiet signal that the week was elevated, not a judgment.
 */
export const OVERWHELM_ELEVATED_THRESHOLD = 4

export interface WeekOverwhelmTag {
  name: string
  count: number
}

export interface WeekOverwhelmSummary {
  average: number
  count: number
  elevated: boolean
  topTags: WeekOverwhelmTag[]
}

/**
 * useOverwhelmSummary distills the current Monday-starting week's overwhelm
 * entries into an average and, when that average is elevated, the top tags
 * by frequency — so the dashboard can show why a hard week felt hard.
 */
export function useOverwhelmSummary() {
  const { computeWeeklyAverageBy, toLocalDate } = useWeeklyAverages()

  function computeCurrentWeekSummary(
    entries: OverwhelmEntry[],
    now: Date = new Date()
  ): WeekOverwhelmSummary | null {
    const weeks = computeWeeklyAverageBy(entries, e => e.day, e => e.overwhelmLevel)
    const currentWeekStart = startOfWeek(now, { weekStartsOn: 1 }).toISOString()
    const week = weeks.find(w => w.weekStart === currentWeekStart)
    if (!week) {
      return null
    }

    const elevated = week.average > OVERWHELM_ELEVATED_THRESHOLD
    if (!elevated) {
      return {
        average: week.average,
        count: week.count,
        elevated: false,
        topTags: []
      }
    }

    const counts = new Map<string, number>()
    for (const entry of entries) {
      if (startOfWeek(toLocalDate(entry.day), { weekStartsOn: 1 }).toISOString() !== currentWeekStart) {
        continue
      }
      for (const tag of entry.tags) {
        counts.set(tag.name, (counts.get(tag.name) ?? 0) + 1)
      }
    }

    const topTags = Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
      .slice(0, 2)
      .map(([name, count]) => ({ name, count }))

    return {
      average: week.average,
      count: week.count,
      elevated: true,
      topTags
    }
  }

  /**
   * Builds the short elevated-week sentence. Tag names are returned separately
   * so the template can color them with the overwhelm accent.
   */
  function elevatedTagParts(tags: WeekOverwhelmTag[]): { lead: string, tags: string[], trail: string } | null {
    if (!tags.length) {
      return null
    }
    return {
      lead: 'Overwhelmed by',
      tags: tags.map(t => t.name),
      trail: 'this week.'
    }
  }

  return { computeCurrentWeekSummary, elevatedTagParts }
}
