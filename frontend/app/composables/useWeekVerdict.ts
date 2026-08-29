import type { ActiveEnergyEntry } from '~/stores/activeEnergy'
import type { WeightEntry } from '~/stores/weights'

export type TrendDirection = 'down' | 'up' | 'steady' | null
export type WeekVerdict = 'better' | 'worse' | 'steady' | null

interface WeeklyPoint {
  weekStart: string
  value: number
}

const MS_PER_WEEK = 7 * 24 * 60 * 60 * 1000

/** How many weekly buckets the trend reads back over, current week included. */
const TREND_WEEKS = 4

/**
 * The weight deadband is one-sided, and only guards the *up* side: a rising
 * slope has to clear it before the sentence says so, because a week of scale
 * noise shouldn't be reported as a gain. Falling weight gets no such band -
 * the chart sits directly below this sentence, and a line that slopes down
 * while the words say "holding steady" reads as the app arguing with its own
 * graph. Two weeks is a single difference rather than a fit, so noise can
 * carry it on its own; that case keeps the wider band. Energy stays
 * two-sided and relative to its own average, since what counts as a real
 * change in output depends on how much a person burns to begin with.
 */
const WEIGHT_GAIN_KG_PER_WEEK = 0.15
const WEIGHT_GAIN_KG_TWO_WEEKS = 0.3
const ENERGY_STEADY_FRACTION = 0.05

/**
 * Least-squares slope through weekly buckets, in units per week. Weeks are
 * placed by their actual start date, so a week with no entries leaves a real
 * gap rather than silently pulling the next bucket forward.
 */
function slopePerWeek(points: WeeklyPoint[]): number {
  const origin = new Date(points[0]!.weekStart).getTime()
  const xs = points.map(p => Math.round((new Date(p.weekStart).getTime() - origin) / MS_PER_WEEK))
  const meanX = xs.reduce((sum, x) => sum + x, 0) / xs.length
  const meanY = points.reduce((sum, p) => sum + p.value, 0) / points.length

  let covariance = 0
  let variance = 0
  points.forEach((point, index) => {
    const dx = xs[index]! - meanX
    covariance += dx * (point.value - meanY)
    variance += dx * dx
  })
  return variance === 0 ? 0 : covariance / variance
}

/**
 * useWeekVerdict distills recent weight and active energy data into a single
 * glanceable verdict, so the dashboard can answer "am I doing better or worse
 * lately?" without the user reading a grid of numbers.
 *
 * Everything is read week by week, never day by day: daily weight swings on
 * hydration and time of day, and the running week is always a partial one, so
 * a straight last-reading-vs-previous comparison mostly reports noise. Weekly
 * buckets damp that, and taking the slope across the last few of them means a
 * slow, consistent loss registers as progress instead of disappearing into
 * the steady band the way a single week-over-week difference would.
 *
 * Where the two directions are treated differently, the tie goes to the
 * user: any downward weight slope counts, while a rising one has to earn it.
 */
export function useWeekVerdict() {
  const { computeWeeklyAverages, computeWeeklyAverageBy } = useWeeklyAverages()

  function computeWeightTrend(entries: WeightEntry[]): TrendDirection {
    const weeks = computeWeeklyAverages(entries).slice(-TREND_WEEKS)
    if (weeks.length < 2) {
      return null
    }
    const slope = slopePerWeek(weeks.map(w => ({ weekStart: w.weekStart, value: w.average })))
    if (slope < 0) {
      return 'down'
    }
    const gainThreshold = weeks.length > 2 ? WEIGHT_GAIN_KG_PER_WEEK : WEIGHT_GAIN_KG_TWO_WEEKS
    return slope < gainThreshold ? 'steady' : 'up'
  }

  /**
   * Averaged per logged day rather than summed, because the current week is
   * partial: a running week's total is always short of a finished one, which
   * would read as falling effort every day but Sunday.
   */
  function computeEnergyTrend(entries: ActiveEnergyEntry[]): TrendDirection {
    const weeks = computeWeeklyAverageBy(entries, e => e.day, e => e.activeEnergyKcal).slice(-TREND_WEEKS)
    if (weeks.length < 2) {
      return null
    }
    const slope = slopePerWeek(weeks.map(w => ({ weekStart: w.weekStart, value: w.average })))
    const mean = weeks.reduce((sum, w) => sum + w.average, 0) / weeks.length
    if (mean <= 0 || Math.abs(slope) / mean < ENERGY_STEADY_FRACTION) {
      return 'steady'
    }
    return slope > 0 ? 'up' : 'down'
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
