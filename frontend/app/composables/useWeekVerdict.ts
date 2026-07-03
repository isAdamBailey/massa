import type { ActiveEnergyEntry } from '~/stores/activeEnergy'
import type { WeightEntry } from '~/stores/weights'

export type TrendDirection = 'down' | 'up' | 'steady' | null
export type WeekVerdict = 'better' | 'worse' | 'steady' | null

/**
 * useWeekVerdict distills a week of weight and active energy data into a
 * single glanceable verdict, so the dashboard can answer "am I doing better
 * or worse than last week?" without the user reading a grid of numbers.
 */
export function useWeekVerdict() {
  const { computeWeeklyAverages, computeWeeklySumBy } = useWeeklyAverages()

  function computeWeightTrend(entries: WeightEntry[]): TrendDirection {
    const weeks = computeWeeklyAverages(entries)
    if (weeks.length < 2) {
      return null
    }
    const diff = weeks[weeks.length - 1]!.average - weeks[weeks.length - 2]!.average
    if (Math.abs(diff) < 0.3) {
      return 'steady'
    }
    return diff < 0 ? 'down' : 'up'
  }

  function computeEnergyTrend(entries: ActiveEnergyEntry[]): TrendDirection {
    const weeks = computeWeeklySumBy(entries, e => e.day, e => e.activeEnergyKcal)
    if (weeks.length < 2) {
      return null
    }
    const previous = weeks[weeks.length - 2]!.total
    const diff = weeks[weeks.length - 1]!.total - previous
    if (previous <= 0 || Math.abs(diff) / previous < 0.05) {
      return 'steady'
    }
    return diff > 0 ? 'up' : 'down'
  }

  /**
   * A single, forgiving verdict: better if either signal improved (weight
   * down or energy up), so being active still counts as a win on a week the
   * scale doesn't move. Worse only if weight is up and energy didn't pick up
   * the slack.
   */
  function computeVerdict(weightTrend: TrendDirection, energyTrend: TrendDirection): WeekVerdict {
    if (!weightTrend && !energyTrend) {
      return null
    }
    if (weightTrend === 'down' || energyTrend === 'up') {
      return 'better'
    }
    if (weightTrend === 'up') {
      return 'worse'
    }
    return 'steady'
  }

  function verdictLabel(verdict: WeekVerdict): string {
    switch (verdict) {
      case 'better':
        return 'Better this week'
      case 'worse':
        return 'Worse this week'
      case 'steady':
        return 'Steady this week'
      default:
        return 'Not enough data yet'
    }
  }

  return { computeWeightTrend, computeEnergyTrend, computeVerdict, verdictLabel }
}
