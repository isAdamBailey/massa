import { startOfMonth, startOfWeek } from 'date-fns'
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

export interface MonthlyAverage {
  monthStart: string
  average: number
  count: number
}

export interface MonthlyTotal {
  monthStart: string
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
 * Shared grouping logic behind computeWeeklyAverageBy/computeMonthlyAverageBy
 * - both bucket entries by a date, then average a chosen value per bucket.
 * `bucketStart` picks the unit (Monday-starting week vs. calendar month).
 */
function bucketAverageBy<T>(
  items: T[],
  dateFn: (item: T) => string | Date,
  valueFn: (item: T) => number | null | undefined,
  bucketStart: (date: Date) => Date
): { bucketStart: string, average: number, count: number }[] {
  const groups = new Map<string, { sum: number, count: number }>()
  for (const item of items) {
    const value = valueFn(item)
    if (value === null || value === undefined) {
      continue
    }
    const key = bucketStart(toLocalDate(dateFn(item))).toISOString()
    const group = groups.get(key) ?? { sum: 0, count: 0 }
    group.sum += value
    group.count += 1
    groups.set(key, group)
  }
  return Array.from(groups.entries())
    .map(([key, group]) => ({ bucketStart: key, average: group.sum / group.count, count: group.count }))
    .sort((a, b) => a.bucketStart.localeCompare(b.bucketStart))
}

/**
 * Shared grouping logic behind computeWeeklySumBy/computeMonthlySumBy, for
 * metrics like active energy where the meaningful per-bucket figure is a
 * total rather than an average.
 */
function bucketSumBy<T>(
  items: T[],
  dateFn: (item: T) => string | Date,
  valueFn: (item: T) => number | null | undefined,
  bucketStart: (date: Date) => Date
): { bucketStart: string, total: number }[] {
  const groups = new Map<string, number>()
  for (const item of items) {
    const value = valueFn(item)
    if (value === null || value === undefined) {
      continue
    }
    const key = bucketStart(toLocalDate(dateFn(item))).toISOString()
    groups.set(key, (groups.get(key) ?? 0) + value)
  }
  return Array.from(groups.entries())
    .map(([key, total]) => ({ bucketStart: key, total }))
    .sort((a, b) => a.bucketStart.localeCompare(b.bucketStart))
}

const weekStart = (date: Date) => startOfWeek(date, { weekStartsOn: 1 })

/**
 * useWeeklyAverages groups weight entries into Monday-starting weeks (or,
 * for the monthly variants, calendar months) and computes per-bucket
 * averages of a chosen value (weight or BMI), shared by the chart and
 * dashboard summary.
 */
export function useWeeklyAverages() {
  function computeWeeklyAverageBy<T>(
    items: T[],
    dateFn: (item: T) => string | Date,
    valueFn: (item: T) => number | null | undefined
  ): WeeklyAverage[] {
    return bucketAverageBy(items, dateFn, valueFn, weekStart)
      .map(({ bucketStart, average, count }) => ({ weekStart: bucketStart, average, count }))
  }

  function computeWeeklyAverages(entries: WeightEntry[]): WeeklyAverage[] {
    return computeWeeklyAverageBy(entries, entry => entry.recordedAt, entry => entry.weightKg)
  }

  function currentWeekAverage(entries: WeightEntry[]): WeeklyAverage | null {
    const currentWeekStart = weekStart(new Date()).toISOString()
    return computeWeeklyAverages(entries).find(w => w.weekStart === currentWeekStart) ?? null
  }

  function computeWeeklySumBy<T>(items: T[], dateFn: (item: T) => string | Date, valueFn: (item: T) => number | null | undefined): WeeklyTotal[] {
    return bucketSumBy(items, dateFn, valueFn, weekStart)
      .map(({ bucketStart, total }) => ({ weekStart: bucketStart, total }))
  }

  function computeMonthlyAverageBy<T>(
    items: T[],
    dateFn: (item: T) => string | Date,
    valueFn: (item: T) => number | null | undefined
  ): MonthlyAverage[] {
    return bucketAverageBy(items, dateFn, valueFn, startOfMonth)
      .map(({ bucketStart, average, count }) => ({ monthStart: bucketStart, average, count }))
  }

  function computeMonthlySumBy<T>(items: T[], dateFn: (item: T) => string | Date, valueFn: (item: T) => number | null | undefined): MonthlyTotal[] {
    return bucketSumBy(items, dateFn, valueFn, startOfMonth)
      .map(({ bucketStart, total }) => ({ monthStart: bucketStart, total }))
  }

  return {
    computeWeeklyAverageBy,
    computeWeeklyAverages,
    currentWeekAverage,
    computeWeeklySumBy,
    computeMonthlyAverageBy,
    computeMonthlySumBy,
    toLocalDate
  }
}
