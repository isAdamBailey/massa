import { startOfWeek } from 'date-fns'
import type { WeightEntry } from '~/stores/weights'

export interface WeeklyAverage {
  weekStart: string
  average: number
  count: number
}

export interface WeeklyTotal {
  weekStart: string
  total: number
}

const DATE_ONLY = /^\d{4}-\d{2}-\d{2}$/

/**
 * Parses a date-only string (e.g. active energy's "day" field) as local
 * midnight rather than UTC midnight. `new Date("2024-01-01")` parses as UTC,
 * which in negative-UTC-offset timezones renders as the previous day
 * locally - shifting it into the wrong week right at a week boundary.
 */
function toLocalDate(input: string | Date): Date {
  if (input instanceof Date) {
    return input
  }
  const match = DATE_ONLY.exec(input)
  if (!match) {
    return new Date(input)
  }
  const [year, month, day] = input.split('-').map(Number)
  return new Date(year!, month! - 1, day!)
}

/**
 * useWeeklyAverages groups weight entries into Monday-starting weeks and
 * computes per-week averages of a chosen value (weight or BMI), shared by
 * the chart and dashboard summary.
 */
export function useWeeklyAverages() {
  function computeWeeklyAverageBy(entries: WeightEntry[], valueFn: (entry: WeightEntry) => number | null | undefined): WeeklyAverage[] {
    const groups = new Map<string, { sum: number, count: number }>()
    for (const entry of entries) {
      const value = valueFn(entry)
      if (value === null || value === undefined) {
        continue
      }
      const weekStart = startOfWeek(new Date(entry.recordedAt), { weekStartsOn: 1 }).toISOString()
      const group = groups.get(weekStart) ?? { sum: 0, count: 0 }
      group.sum += value
      group.count += 1
      groups.set(weekStart, group)
    }
    return Array.from(groups.entries())
      .map(([weekStart, group]) => ({
        weekStart,
        average: group.sum / group.count,
        count: group.count
      }))
      .sort((a, b) => a.weekStart.localeCompare(b.weekStart))
  }

  function computeWeeklyAverages(entries: WeightEntry[]): WeeklyAverage[] {
    return computeWeeklyAverageBy(entries, entry => entry.weightKg)
  }

  function currentWeekAverage(entries: WeightEntry[]): WeeklyAverage | null {
    const currentWeekStart = startOfWeek(new Date(), { weekStartsOn: 1 }).toISOString()
    return computeWeeklyAverages(entries).find(w => w.weekStart === currentWeekStart) ?? null
  }

  /**
   * computeWeeklySumBy groups arbitrary dated items into Monday-starting
   * weeks and sums a chosen value, for metrics like active energy where the
   * meaningful weekly figure is a total rather than an average.
   */
  function computeWeeklySumBy<T>(items: T[], dateFn: (item: T) => string | Date, valueFn: (item: T) => number | null | undefined): WeeklyTotal[] {
    const groups = new Map<string, number>()
    for (const item of items) {
      const value = valueFn(item)
      if (value === null || value === undefined) {
        continue
      }
      const weekStart = startOfWeek(toLocalDate(dateFn(item)), { weekStartsOn: 1 }).toISOString()
      groups.set(weekStart, (groups.get(weekStart) ?? 0) + value)
    }
    return Array.from(groups.entries())
      .map(([weekStart, total]) => ({ weekStart, total }))
      .sort((a, b) => a.weekStart.localeCompare(b.weekStart))
  }

  return { computeWeeklyAverageBy, computeWeeklyAverages, currentWeekAverage, computeWeeklySumBy }
}
