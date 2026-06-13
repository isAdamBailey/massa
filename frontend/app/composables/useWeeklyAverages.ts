import { startOfWeek } from 'date-fns'
import type { WeightEntry } from '~/stores/weights'

export interface WeeklyAverage {
  weekStart: string
  averageKg: number
  count: number
}

/**
 * useWeeklyAverages groups weight entries into Monday-starting weeks and
 * computes the average weight (kg) per week, shared by the chart and
 * dashboard summary.
 */
export function useWeeklyAverages() {
  function computeWeeklyAverages(entries: WeightEntry[]): WeeklyAverage[] {
    const groups = new Map<string, { sum: number, count: number }>()
    for (const entry of entries) {
      const weekStart = startOfWeek(new Date(entry.recordedAt), { weekStartsOn: 1 }).toISOString()
      const group = groups.get(weekStart) ?? { sum: 0, count: 0 }
      group.sum += entry.weightKg
      group.count += 1
      groups.set(weekStart, group)
    }
    return Array.from(groups.entries())
      .map(([weekStart, group]) => ({
        weekStart,
        averageKg: group.sum / group.count,
        count: group.count
      }))
      .sort((a, b) => a.weekStart.localeCompare(b.weekStart))
  }

  function currentWeekAverage(entries: WeightEntry[]): WeeklyAverage | null {
    const currentWeekStart = startOfWeek(new Date(), { weekStartsOn: 1 }).toISOString()
    return computeWeeklyAverages(entries).find(w => w.weekStart === currentWeekStart) ?? null
  }

  return { computeWeeklyAverages, currentWeekAverage }
}
