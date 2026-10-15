import { addDays, format, startOfDay, subDays } from 'date-fns'
import type { ActiveEnergyEntry } from '~/stores/activeEnergy'
import type { OverwhelmEntry } from '~/stores/overwhelm'
import type { UnitsPreference } from '~/stores/settings'
import type { WeightEntry } from '~/stores/weights'
import type { ChartMetricMode } from '~/composables/useMetricAccent'
import type { StatusSentenceSegment } from '~/composables/useStatusSentence'
import { OVERWHELM_BASELINE, OVERWHELM_ELEVATED_THRESHOLD } from '~/composables/useOverwhelm'

export interface RangeChangeSummary {
  /** The Trend card's hero sentence; only the number phrase is `highlight`. */
  headline: StatusSentenceSegment[]
  /** Fog detail line under the headline, or null when there's nothing to back it. */
  detail: string | null
}

export interface RangeChangeSources {
  weights: WeightEntry[]
  energy: ActiveEnergyEntry[]
  overwhelm: OverwhelmEntry[]
}

interface Reading {
  at: Date
  value: number
}

interface Window {
  /**
   * Local midnight starting the span's first calendar day (today counts as
   * one), so day-keyed entries land wholly inside or outside it. null for
   * All: the window starts at the first reading.
   */
  start: Date | null
  spanDays: number | null
}

const MS_PER_DAY = 24 * 60 * 60 * 1000

/** Readings averaged at each end of the span, so one morning's weigh-in can't swing the headline. */
const SMOOTHING_DAYS = 7

/** Spans shorter than this compare the first and last readings directly. */
const SMOOTHED_MIN_SPAN_DAYS = 30

/**
 * Below this coverage the first and last 7-day groups would overlap and
 * mostly average the same readings, reporting a fake "steady" - so fall back
 * to the raw first/last readings there too.
 */
const SMOOTHED_MIN_COVERAGE_DAYS = 2 * SMOOTHING_DAYS

/** A window whose first reading lands later than this reads "since <date>" instead. */
const SINCE_GRACE_DAYS = 7

/** |Δ| below these (in display units) reads as holding steady. */
const STEADY_WEIGHT_LB = 0.5
const STEADY_WEIGHT_KG = 0.2
const STEADY_BMI = 0.1

/**
 * Overwhelm is a felt 1-10 scale, so the summary names a band rather than
 * a decimal: within this distance of the baseline reads as "around" it.
 */
const OVERWHELM_AROUND_BASELINE = 0.5

/** Start-to-end overwhelm changes smaller than this read as "about the same". */
const OVERWHELM_STEADY = 0.5

const NOT_ENOUGH: RangeChangeSummary = {
  headline: [{ text: 'Not enough readings in this span yet.' }],
  detail: null
}

const integer = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 })

const DATE_ONLY = /^\d{4}-\d{2}-\d{2}$/

/** Day-keyed entries are local calendar days, not UTC midnights (see useWeeklyAverages). */
function toLocalDate(input: string): Date {
  if (!DATE_ONLY.test(input)) {
    return new Date(input)
  }
  const [year, month, day] = input.split('-').map(Number)
  return new Date(year!, month! - 1, day!)
}

function localDayKey(date: Date): string {
  return format(date, 'yyyy-MM-dd')
}

function mean(readings: Reading[]): number {
  return readings.reduce((sum, r) => sum + r.value, 0) / readings.length
}

function round1(value: number): number {
  return Math.round(value * 10) / 10
}

/** Fixed one-decimal display that never shows "-0.0". */
function fixed1(value: number): string {
  return (round1(value) || 0).toFixed(1)
}

function spanName(spanDays: number): string {
  if (spanDays === 182) {
    return '6 months'
  }
  if (spanDays === 365) {
    return 'year'
  }
  return `${spanDays} days`
}

/** Phrase that closes the headline: "over the last 90 days" or "since Mar 4". */
function windowPhrase(window: Window, first: Reading, now: Date): string {
  if (window.start !== null && window.spanDays !== null
    && first.at.getTime() - window.start.getTime() <= SINCE_GRACE_DAYS * MS_PER_DAY) {
    return `over the last ${spanName(window.spanDays)}`
  }
  const pattern = first.at.getFullYear() === now.getFullYear() ? 'MMM d' : 'MMM d, yyyy'
  return `since ${format(first.at, pattern)}`
}

interface Endpoints {
  start: number
  end: number
  /** How the endpoints were computed, for the detail line. */
  method: string
}

/**
 * Start and end values for a window's readings (sorted, ≥ 2 distinct days).
 * Spans of 30 days or more average the first and last 7 days of available
 * data; shorter spans, or data too sparse for two separate 7-day groups,
 * compare the first and last readings.
 */
function endpoints(readings: Reading[], spanDays: number | null): Endpoints {
  const first = readings[0]!
  const last = readings.at(-1)!
  const coverageDays = (last.at.getTime() - first.at.getTime()) / MS_PER_DAY
  const smoothed = (spanDays === null || spanDays >= SMOOTHED_MIN_SPAN_DAYS)
    && coverageDays >= SMOOTHED_MIN_COVERAGE_DAYS
  if (!smoothed) {
    return { start: first.value, end: last.value, method: 'first and last readings' }
  }
  const startCutoff = first.at.getTime() + SMOOTHING_DAYS * MS_PER_DAY
  const endCutoff = last.at.getTime() - SMOOTHING_DAYS * MS_PER_DAY
  return {
    start: mean(readings.filter(r => r.at.getTime() < startCutoff)),
    end: mean(readings.filter(r => r.at.getTime() > endCutoff)),
    method: `${SMOOTHING_DAYS}-day averages`
  }
}

function hasEnoughReadings(readings: Reading[]): boolean {
  return readings.length >= 2 && new Set(readings.map(r => localDayKey(r.at))).size >= 2
}

/**
 * spanStart is local midnight on the first calendar day of a `spanDays`
 * span ending today (today counts as one day), or null for All. Shared with
 * the chart, so the line and the summary always cover the same days.
 */
export function spanStart(spanDays: number | null, now: Date = new Date()): Date | null {
  return spanDays === null ? null : startOfDay(subDays(now, spanDays - 1))
}

function resolveWindow(spanDays: number | null, now: Date): Window {
  return { start: spanStart(spanDays, now), spanDays }
}

function inWindow(readings: Reading[], window: Window, now: Date): Reading[] {
  return readings.filter(r => (window.start === null || r.at >= window.start) && r.at <= now)
}

/** The span of the same length right before the window; empty for All. */
function previousSpan(readings: Reading[], window: Window): Reading[] {
  if (window.start === null || window.spanDays === null) {
    return []
  }
  const start = window.start
  const previousStart = subDays(start, window.spanDays)
  return readings.filter(r => r.at >= previousStart && r.at < start)
}

function sorted(readings: Reading[]): Reading[] {
  return [...readings].sort((a, b) => a.at.getTime() - b.at.getTime())
}

/**
 * "Down 3.2 lb" / "BMI up 0.4" / "Holding steady" for start-vs-end metrics.
 * The direction and steady check use the rounded endpoints, so the headline
 * always agrees with the "182.4 → 179.2" arithmetic in the detail line.
 */
function changeSummary(
  readings: Reading[],
  window: Window,
  now: Date,
  opts: { label: string, unit: string, steady: number, convert: (v: number) => number }
): RangeChangeSummary {
  const { start, end, method } = endpoints(readings, window.spanDays)
  const from = round1(opts.convert(start))
  const to = round1(opts.convert(end))
  const delta = round1(to - from)
  const unitSuffix = opts.unit ? ` ${opts.unit}` : ''
  const phrase = ` ${windowPhrase(window, readings[0]!, now)}`
  const range = `${fixed1(from)} → ${fixed1(to)}${unitSuffix}`

  if (Math.abs(delta) < opts.steady) {
    const lead = opts.label ? `${opts.label} holding steady` : 'Holding steady'
    const change = delta === 0 ? 'no change' : `${delta < 0 ? 'down' : 'up'} ${fixed1(Math.abs(delta))}`
    return {
      headline: [{ text: lead, highlight: true }, { text: phrase }],
      detail: `${range} · ${change} · ${method}`
    }
  }

  const direction = delta < 0 ? 'down' : 'up'
  const amount = `${fixed1(Math.abs(delta))}${unitSuffix}`
  const lead = opts.label
    ? `${opts.label} ${direction} ${amount}`
    : `${direction === 'down' ? 'Down' : 'Up'} ${amount}`
  return {
    headline: [{ text: lead, highlight: true }, { text: phrase }],
    detail: `${range} · ${method}`
  }
}

/** "up from 380" / "down from 4.5" / "same as" - lowercase, for mid-sentence use. */
function relativeTo(current: string, previous: string, previousNumber: number, currentNumber: number): string {
  if (current === previous) {
    return 'same as'
  }
  return `${currentNumber < previousNumber ? 'down' : 'up'} from ${previous}`
}

function capitalize(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1)
}

/**
 * useRangeChange turns the Trend card's time span into a calm sentence about
 * how much the selected metric moved over it, plus a detail line saying how
 * that was worked out. It's pure logic over the full, unfiltered store
 * entries: it resolves the window itself (and, for energy, the span right
 * before it), so the caller never refetches per span.
 *
 * Numbers are reported without judgment - "down"/"up", never "lost"/
 * "gained" - and the result is independent of the chart's daily/weekly/
 * monthly aggregation, so toggling that never changes the headline.
 */
export function useRangeChange() {
  const { kgToLb } = useBmi()

  function summarizeWeight(entries: WeightEntry[], spanDays: number | null, units: UnitsPreference, now: Date): RangeChangeSummary {
    const window = resolveWindow(spanDays, now)
    const readings = sorted(inWindow(entries.map(e => ({ at: new Date(e.recordedAt), value: e.weightKg })), window, now))
    if (!hasEnoughReadings(readings)) {
      return NOT_ENOUGH
    }
    const imperial = units === 'imperial'
    return changeSummary(readings, window, now, {
      label: '',
      unit: imperial ? 'lb' : 'kg',
      steady: imperial ? STEADY_WEIGHT_LB : STEADY_WEIGHT_KG,
      convert: imperial ? kgToLb : v => v
    })
  }

  function summarizeBmi(entries: WeightEntry[], spanDays: number | null, now: Date): RangeChangeSummary {
    const window = resolveWindow(spanDays, now)
    const readings = sorted(inWindow(
      entries.flatMap(e => e.bmi === undefined || e.bmi === null ? [] : [{ at: new Date(e.recordedAt), value: e.bmi }]),
      window,
      now
    ))
    if (!hasEnoughReadings(readings)) {
      return NOT_ENOUGH
    }
    return changeSummary(readings, window, now, { label: 'BMI', unit: '', steady: STEADY_BMI, convert: v => v })
  }

  /**
   * Energy reads as a daily average, compared with the span of the same
   * length right before it. For All, or when that earlier span doesn't have
   * enough readings of its own, it falls back to the smoothed start-vs-end comparison.
   */
  function summarizeEnergy(entries: ActiveEnergyEntry[], spanDays: number | null, now: Date): RangeChangeSummary {
    const window = resolveWindow(spanDays, now)
    const all = sorted(entries.map(e => ({ at: toLocalDate(e.day), value: e.activeEnergyKcal })))
    const readings = inWindow(all, window, now)
    if (!hasEnoughReadings(readings)) {
      return NOT_ENOUGH
    }
    const average = mean(readings)
    const total = readings.reduce((sum, r) => sum + r.value, 0)
    const averageText = integer.format(average)
    const headline: StatusSentenceSegment[] = [
      { text: `Averaging ${averageText} kcal a day`, highlight: true },
      { text: ` ${windowPhrase(window, readings[0]!, now)}` }
    ]
    const totalText = `${integer.format(total)} kcal total`

    const previous = previousSpan(all, window)
    // A lone reading isn't a span to compare against - e.g. the boundary day
    // a `from=`-filtered fetch returns just before the window.
    if (hasEnoughReadings(previous) && window.spanDays !== null) {
      const previousAverage = mean(previous)
      const comparison = relativeTo(averageText, integer.format(previousAverage), previousAverage, average)
      return {
        headline,
        detail: `${capitalize(comparison)} ${comparison === 'same as' ? 'the' : 'in the'} ${spanName(window.spanDays)} before · ${totalText}`
      }
    }

    const { start, end } = endpoints(readings, window.spanDays)
    const comparison = relativeTo(integer.format(end), integer.format(start), start, end)
    return {
      headline,
      detail: `${capitalize(comparison)} at the start · ${totalText}`
    }
  }

  /**
   * Overwhelm is a felt 1-10 scale where a decimal like "3.8" means little,
   * so it reads as a band relative to the baseline - below, around, a bit
   * above (up to the dashboard's elevated threshold), or above - with the
   * detail line saying how the span ended compared with how it started.
   *
   * Hearing nothing is read as a baseline day, not as a calm one: every
   * unlogged day in the span counts as OVERWHELM_BASELINE. So "below" (or
   * "above") takes enough explicit entries to move the whole span's
   * average, and a span with no entries reads as around the baseline.
   */
  function summarizeOverwhelm(entries: OverwhelmEntry[], spanDays: number | null, now: Date): RangeChangeSummary {
    const window = resolveWindow(spanDays, now)
    const logged = sorted(inWindow(entries.map(e => ({ at: toLocalDate(e.day), value: e.overwhelmLevel })), window, now))
    const first = logged[0]
    if (!first && window.start === null) {
      return NOT_ENOUGH
    }

    // The span runs from the window start, or from the first entry when the
    // headline says "since <date>" - days before someone started logging
    // aren't days they were at baseline.
    const phrase = first ? windowPhrase(window, first, now) : `over the last ${spanName(window.spanDays!)}`
    const from = phrase.startsWith('since') ? startOfDay(first!.at) : window.start!
    const byDay = new Map(logged.map(r => [localDayKey(r.at), r.value]))
    const readings: Reading[] = []
    for (let day = from; day <= now; day = addDays(day, 1)) {
      readings.push({ at: day, value: byDay.get(localDayKey(day)) ?? OVERWHELM_BASELINE })
    }
    if (readings.length < 2) {
      return NOT_ENOUGH
    }

    const average = mean(readings)
    let band: string
    if (average < OVERWHELM_BASELINE - OVERWHELM_AROUND_BASELINE) {
      band = 'Below your baseline'
    } else if (average <= OVERWHELM_BASELINE + OVERWHELM_AROUND_BASELINE) {
      band = 'Around your baseline'
    } else if (average <= OVERWHELM_ELEVATED_THRESHOLD) {
      band = 'A bit above your baseline'
    } else {
      band = 'Above your baseline'
    }
    const { start, end } = endpoints(readings, window.spanDays)
    const change = end - start
    let detail: string
    if (!first) {
      detail = 'Nothing logged in this span'
    } else if (Math.abs(change) < OVERWHELM_STEADY) {
      detail = 'About the same as at the start'
    } else {
      detail = change < 0 ? 'Calmer than at the start' : 'More overwhelmed than at the start'
    }
    return {
      headline: [
        { text: band, highlight: true },
        { text: ` ${phrase}` }
      ],
      detail
    }
  }

  /**
   * buildRangeChange summarizes `metric` over the last `spanDays` days
   * (null for All) ending at `now`, from full, unfiltered store entries.
   * Weight entries should already be deduped (see the dashboard's
   * displayEntries).
   */
  function buildRangeChange(
    metric: ChartMetricMode,
    sources: RangeChangeSources,
    spanDays: number | null,
    units: UnitsPreference,
    now: Date = new Date()
  ): RangeChangeSummary {
    switch (metric) {
      case 'weight':
        return summarizeWeight(sources.weights, spanDays, units, now)
      case 'bmi':
        return summarizeBmi(sources.weights, spanDays, now)
      case 'energy':
        return summarizeEnergy(sources.energy, spanDays, now)
      case 'overwhelm':
        return summarizeOverwhelm(sources.overwhelm, spanDays, now)
    }
  }

  return { buildRangeChange }
}
