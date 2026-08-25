<script setup lang="ts">
import 'chartjs-adapter-date-fns'
import { format, startOfMonth, startOfWeek } from 'date-fns'
import {
  BarController,
  BarElement,
  Chart as ChartJS,
  Filler,
  LinearScale,
  LineController,
  LineElement,
  PointElement,
  TimeScale,
  Tooltip
} from 'chart.js'
import type { Plugin } from 'chart.js'
import { Bar, Line } from 'vue-chartjs'
import { BMI_BOUNDARIES } from '~/composables/useBmi'
import {
  accentForChartMetric,
  METRIC_ACCENT_OKLCH
} from '~/composables/useMetricAccent'
import { OVERWHELM_BASELINE } from '~/composables/useOverwhelm'
import type { SegmentedOption } from '~/components/SegmentedControl.vue'
import type { ActiveEnergyEntry } from '~/stores/activeEnergy'
import type { OverwhelmEntry } from '~/stores/overwhelm'
import type { UnitsPreference } from '~/stores/settings'
import type { WeightEntry } from '~/stores/weights'

ChartJS.register(LineController, LineElement, PointElement, BarController, BarElement, LinearScale, TimeScale, Tooltip, Filler)

const props = defineProps<{
  entries: WeightEntry[]
  activeEnergyEntries: ActiveEnergyEntry[]
  overwhelmEntries: OverwhelmEntry[]
  unitsPreference: UnitsPreference
}>()

const { kgToLb, category } = useBmi()
const { computeWeeklyAverageBy, computeWeeklySumBy, computeMonthlyAverageBy, computeMonthlySumBy, toLocalDate } = useWeeklyAverages()

export type ViewMode = 'daily' | 'weekly' | 'monthly'
type MetricMode = 'weight' | 'bmi' | 'energy' | 'overwhelm'

const viewMode = defineModel<ViewMode>('viewMode', { default: 'weekly' })
const metricMode = defineModel<MetricMode>('metricMode', { default: 'weight' })

const metricOptions: SegmentedOption<MetricMode>[] = [
  { value: 'weight', label: 'Weight', accent: 'verdigris' },
  { value: 'bmi', label: 'BMI', accent: 'verdigris' },
  { value: 'energy', label: 'Energy', accent: 'copper' },
  { value: 'overwhelm', label: 'Overwhelm', accent: 'cobalt' }
]

const viewOptions: SegmentedOption<ViewMode>[] = [
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' }
]

const FOG = 'oklch(0.78 0.015 170)'
const HAIRLINE = 'oklch(0.44 0.014 170)'
const GRAPHITE = 'oklch(0.35 0.018 170)'
const MIST = 'oklch(0.96 0.006 170)'
const AMBER = 'oklch(0.75 0.14 80)'
const AMBER_WASH = 'oklch(0.75 0.14 80 / 0.10)'
const EMBER = 'oklch(0.62 0.17 25)'
const EMBER_WASH = 'oklch(0.62 0.17 25 / 0.12)'

const chartAccent = computed(() => accentForChartMetric(metricMode.value))
const chartPalette = computed(() => METRIC_ACCENT_OKLCH[chartAccent.value])

// Dot size scales down as points get denser, so a 7-day view (few, spaced-out
// points) gets a comfortably tappable dot while a multi-year view (dozens of
// points a few pixels apart) doesn't turn into a solid smear of overlapping
// circles. Hit radius (the invisible tap target) stays generous either way,
// since a small visual dot doesn't need a small touch target.
function pointSizeFor(count: number) {
  if (count <= 10) {
    return { radius: 5, hoverRadius: 7, hitRadius: 14 }
  }
  if (count <= 30) {
    return { radius: 4, hoverRadius: 6, hitRadius: 12 }
  }
  if (count <= 90) {
    return { radius: 2.5, hoverRadius: 5, hitRadius: 10 }
  }
  return { radius: 1.5, hoverRadius: 4.5, hitRadius: 8 }
}

function lineSeries(label: string, data: { x: number, y: number }[]) {
  const { solid, fill } = chartPalette.value
  const { radius, hoverRadius, hitRadius } = pointSizeFor(data.length)
  return {
    datasets: [
      {
        label,
        data,
        borderColor: solid,
        backgroundColor: fill,
        pointBackgroundColor: solid,
        pointBorderColor: solid,
        pointRadius: radius,
        pointHoverRadius: hoverRadius,
        pointHitRadius: hitRadius,
        borderWidth: 2,
        tension: 0.2,
        fill: 'start' as const
      }
    ]
  }
}

function barSeries(label: string, data: { x: number, y: number }[]) {
  const { solid, hover } = chartPalette.value
  return {
    datasets: [
      {
        label,
        data,
        borderColor: solid,
        backgroundColor: solid,
        hoverBackgroundColor: hover,
        borderWidth: 0,
        borderRadius: 3
      }
    ]
  }
}

// BMI reference-range bands, drawn behind the line in BMI mode only.
// Normal/underweight ranges stay untinted; only the elevated ranges get a wash.
const BMI_ZONES: { from: number, to: number, color: string, wash: string, label: string }[] = [
  { from: BMI_BOUNDARIES.overweight, to: BMI_BOUNDARIES.obese, color: AMBER, wash: AMBER_WASH, label: 'Overweight' },
  { from: BMI_BOUNDARIES.obese, to: Infinity, color: EMBER, wash: EMBER_WASH, label: 'Obese' }
]

// The single most recent entry with a BMI value, regardless of view mode -
// used to mark "today" on the chart even when the line itself shows a
// weekly average that can land in a different BMI category.
const latestBmiEntry = computed(() => {
  for (let i = props.entries.length - 1; i >= 0; i--) {
    const entry = props.entries[i]
    if (entry?.bmi != null) {
      return entry
    }
  }
  return null
})

const bmiZonesPlugin: Plugin<'line'> = {
  id: 'bmiZones',
  beforeDraw(chart) {
    if (metricMode.value !== 'bmi') {
      return
    }
    const { ctx, chartArea, scales } = chart
    const yScale = scales.y
    if (!yScale) {
      return
    }

    ctx.save()
    for (const zone of BMI_ZONES) {
      if (zone.from > yScale.max) {
        continue
      }
      const top = Math.max(yScale.getPixelForValue(Math.min(zone.to, yScale.max)), chartArea.top)
      const bottom = Math.min(yScale.getPixelForValue(zone.from), chartArea.bottom)
      if (bottom - top < 1) {
        continue
      }

      ctx.fillStyle = zone.wash
      ctx.fillRect(chartArea.left, top, chartArea.right - chartArea.left, bottom - top)

      if (bottom - top >= 16) {
        ctx.fillStyle = zone.color
        ctx.font = '500 11px "IBM Plex Sans", sans-serif'
        ctx.textAlign = 'right'
        ctx.textBaseline = 'top'
        ctx.fillText(zone.label, chartArea.right - 6, top + 4)
      }
    }
    ctx.restore()
  },
  afterDatasetsDraw(chart) {
    if (metricMode.value !== 'bmi' || !latestBmiEntry.value) {
      return
    }
    const { ctx, chartArea, scales } = chart
    const yScale = scales.y
    if (!yScale) {
      return
    }

    const bmi = latestBmiEntry.value.bmi as number
    const y = Math.min(Math.max(yScale.getPixelForValue(bmi), chartArea.top), chartArea.bottom)

    ctx.save()
    ctx.strokeStyle = MIST
    ctx.globalAlpha = 0.35
    ctx.lineWidth = 1
    ctx.setLineDash([4, 4])
    ctx.beginPath()
    ctx.moveTo(chartArea.left, y)
    ctx.lineTo(chartArea.right, y)
    ctx.stroke()
    ctx.restore()

    const label = `Latest ${bmi.toFixed(1)} · ${category(bmi)}`
    ctx.save()
    ctx.font = '500 11px "IBM Plex Sans", sans-serif'
    ctx.textBaseline = 'middle'
    const textWidth = ctx.measureText(label).width
    const paddingX = 6
    const boxHeight = 18
    const boxY = Math.min(Math.max(y - boxHeight / 2, chartArea.top), chartArea.bottom - boxHeight)
    ctx.fillStyle = 'oklch(0.27 0.016 170 / 0.85)'
    ctx.fillRect(chartArea.left + 4, boxY, textWidth + paddingX * 2, boxHeight)
    ctx.fillStyle = MIST
    ctx.textAlign = 'left'
    ctx.fillText(label, chartArea.left + 4 + paddingX, boxY + boxHeight / 2 + 1)
    ctx.restore()
  }
}

const overwhelmBaselinePlugin: Plugin<'line'> = {
  id: 'overwhelmBaseline',
  beforeDraw(chart) {
    if (metricMode.value !== 'overwhelm') {
      return
    }
    const { ctx, chartArea, scales } = chart
    const yScale = scales.y
    if (!yScale) {
      return
    }

    const y = yScale.getPixelForValue(OVERWHELM_BASELINE)

    ctx.save()
    ctx.strokeStyle = FOG
    ctx.globalAlpha = 0.5
    ctx.lineWidth = 1
    ctx.setLineDash([4, 4])
    ctx.beginPath()
    ctx.moveTo(chartArea.left, y)
    ctx.lineTo(chartArea.right, y)
    ctx.stroke()

    ctx.setLineDash([])
    ctx.globalAlpha = 1
    ctx.fillStyle = FOG
    ctx.font = '500 11px "IBM Plex Sans", sans-serif'
    ctx.textAlign = 'right'
    ctx.textBaseline = 'bottom'
    ctx.fillText('Baseline', chartArea.right - 6, y - 4)
    ctx.restore()
  }
}

// Tag names per day, keyed by the same epoch-ms x value the daily overwhelm
// dataset plots, so the tooltip can look a point's tags up by its x.
const overwhelmDailyTagNames = computed<Record<number, string>>(() => {
  const map: Record<number, string> = {}
  for (const entry of props.overwhelmEntries) {
    if (!entry.tags.length) {
      continue
    }
    const x = toLocalDate(entry.day).getTime()
    map[x] = entry.tags.map(t => t.name).sort((a, b) => a.localeCompare(b)).join(' · ')
  }
  return map
})

// The top 3 tags by frequency per bucket (ties broken alphabetically), keyed
// by the same epoch-ms x value the weekly/monthly overwhelm dataset plots.
// Reasons don't average, so aggregated modes summarize rather than showing
// every tag. Shared by the weekly and monthly computeds below, which only
// differ in which date-fns function starts the bucket.
function topTagsByBucket(entries: OverwhelmEntry[], bucketStart: (date: Date) => Date): Record<number, string> {
  const countsByBucket = new Map<number, Map<string, number>>()
  for (const entry of entries) {
    if (!entry.tags.length) {
      continue
    }
    const bucket = bucketStart(toLocalDate(entry.day)).getTime()
    const counts = countsByBucket.get(bucket) ?? new Map<string, number>()
    for (const tag of entry.tags) {
      counts.set(tag.name, (counts.get(tag.name) ?? 0) + 1)
    }
    countsByBucket.set(bucket, counts)
  }

  const map: Record<number, string> = {}
  for (const [bucket, counts] of countsByBucket) {
    map[bucket] = Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
      .slice(0, 3)
      .map(([name, count]) => `${name} ×${count}`)
      .join(' · ')
  }
  return map
}

const overwhelmWeeklyTopTags = computed(() => topTagsByBucket(props.overwhelmEntries, d => startOfWeek(d, { weekStartsOn: 1 })))
const overwhelmMonthlyTopTags = computed(() => topTagsByBucket(props.overwhelmEntries, startOfMonth))

const viewLabel = computed(() => {
  if (viewMode.value === 'monthly') {
    return 'Monthly avg'
  }
  return viewMode.value === 'weekly' ? 'Weekly avg' : null
})

type Point = { x: number, y: number }

// Shared monthly/weekly/daily dispatch behind every average-based series
// (weight, BMI, overwhelm) - `daily` supplies the metric-specific fallback
// for the one view mode that isn't a bucketed average.
function bucketedAverage<T>(
  entries: T[],
  dateFn: (item: T) => string | Date,
  valueFn: (item: T) => number | null | undefined,
  daily: () => Point[]
): Point[] {
  if (viewMode.value === 'monthly') {
    return computeMonthlyAverageBy(entries, dateFn, valueFn).map(m => ({ x: new Date(m.monthStart).getTime(), y: m.average }))
  }
  if (viewMode.value === 'weekly') {
    return computeWeeklyAverageBy(entries, dateFn, valueFn).map(w => ({ x: new Date(w.weekStart).getTime(), y: w.average }))
  }
  return daily()
}

// Same dispatch for sum-based series (active energy).
function bucketedSum<T>(
  entries: T[],
  dateFn: (item: T) => string | Date,
  valueFn: (item: T) => number | null | undefined,
  daily: () => Point[]
): Point[] {
  if (viewMode.value === 'monthly') {
    return computeMonthlySumBy(entries, dateFn, valueFn).map(m => ({ x: new Date(m.monthStart).getTime(), y: m.total }))
  }
  if (viewMode.value === 'weekly') {
    return computeWeeklySumBy(entries, dateFn, valueFn).map(w => ({ x: new Date(w.weekStart).getTime(), y: w.total }))
  }
  return daily()
}

const chartData = computed(() => {
  if (metricMode.value === 'overwhelm') {
    const data = bucketedAverage(
      props.overwhelmEntries,
      e => e.day,
      e => e.overwhelmLevel,
      () => props.overwhelmEntries.map(e => ({ x: toLocalDate(e.day).getTime(), y: e.overwhelmLevel }))
    )
    return lineSeries(viewLabel.value ? `${viewLabel.value} overwhelm` : 'Overwhelm', data)
  }

  if (metricMode.value === 'energy') {
    const data = bucketedSum(
      props.activeEnergyEntries,
      e => e.day,
      e => e.activeEnergyKcal,
      () => props.activeEnergyEntries.map(e => ({ x: toLocalDate(e.day).getTime(), y: e.activeEnergyKcal }))
    )
    return barSeries(viewLabel.value ? `${viewLabel.value} active energy (kcal)` : 'Active energy (kcal)', data)
  }

  if (metricMode.value === 'bmi') {
    const data = bucketedAverage(
      props.entries,
      e => e.recordedAt,
      e => e.bmi,
      () => props.entries.filter(e => e.bmi != null).map(e => ({ x: new Date(e.recordedAt).getTime(), y: e.bmi as number }))
    )
    return lineSeries(viewLabel.value ? `${viewLabel.value} BMI` : 'BMI', data)
  }

  const toDisplay = (kg: number) => props.unitsPreference === 'imperial' ? kgToLb(kg) : kg
  const unitLabel = props.unitsPreference === 'imperial' ? 'lb' : 'kg'

  const data = bucketedAverage(
    props.entries,
    e => e.recordedAt,
    e => e.weightKg,
    () => props.entries.map(e => ({ x: new Date(e.recordedAt).getTime(), y: e.weightKg }))
  ).map(point => ({ x: point.x, y: toDisplay(point.y) }))

  return lineSeries(viewLabel.value ? `${viewLabel.value} (${unitLabel})` : `Weight (${unitLabel})`, data)
})

const hasData = computed(() => (chartData.value.datasets[0]?.data.length ?? 0) > 0)

const CHART_UNIT: Record<ViewMode, 'day' | 'week' | 'month'> = { daily: 'day', weekly: 'week', monthly: 'month' }

// Every point lands at local midnight, so the adapter's default title
// ("Jan 5, 2026, 12:00:00 AM") is all-clock, no-signal - each view mode
// instead gets a title format that matches the span a point represents.
const TOOLTIP_TITLE: Record<ViewMode, (x: number) => string> = {
  daily: x => format(x, 'MMM d, yyyy'),
  weekly: x => `Week of ${format(x, 'MMM d, yyyy')}`,
  monthly: x => format(x, 'MMMM yyyy')
}

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    x: {
      type: 'time' as const,
      time: { unit: CHART_UNIT[viewMode.value] },
      ticks: { color: FOG },
      grid: { color: HAIRLINE }
    },
    y: metricMode.value === 'overwhelm'
      ? {
          min: 1,
          max: 10,
          ticks: { stepSize: 1, color: FOG },
          grid: { color: HAIRLINE }
        }
      : {
          beginAtZero: metricMode.value === 'energy',
          grace: '5%',
          ...(metricMode.value === 'bmi' && latestBmiEntry.value
            ? { suggestedMin: (latestBmiEntry.value.bmi as number) - 1, suggestedMax: (latestBmiEntry.value.bmi as number) + 1 }
            : {}),
          ticks: { color: FOG },
          grid: { color: HAIRLINE }
        }
  },
  plugins: {
    tooltip: {
      backgroundColor: GRAPHITE,
      titleColor: MIST,
      bodyColor: MIST,
      borderColor: HAIRLINE,
      borderWidth: 1,
      padding: 10,
      cornerRadius: 6,
      displayColors: false,
      titleFont: { family: '"IBM Plex Sans", sans-serif', size: 12, weight: 600 as const },
      bodyFont: { family: '"IBM Plex Sans", sans-serif', size: 13 },
      footerFont: { family: '"IBM Plex Sans", sans-serif', size: 11 },
      titleMarginBottom: 6,
      callbacks: {
        title: (items: { parsed: { x: number | null } }[]) => {
          const x = items[0]?.parsed.x
          return x == null ? '' : TOOLTIP_TITLE[viewMode.value](x)
        },
        footer: (items: { parsed: { x: number | null } }[]) => {
          const x = items[0]?.parsed.x
          if (metricMode.value !== 'overwhelm' || x == null) {
            return undefined
          }
          const tagsByView: Record<ViewMode, Record<number, string>> = {
            daily: overwhelmDailyTagNames.value,
            weekly: overwhelmWeeklyTopTags.value,
            monthly: overwhelmMonthlyTopTags.value
          }
          return tagsByView[viewMode.value][x] || undefined
        }
      }
    }
  }
}))
</script>

<template>
  <div>
    <div class="mb-4 space-y-3">
      <SegmentedControl
        v-model="metricMode"
        :options="metricOptions"
        group-label="Metric"
        :accent="chartAccent"
        stretch
      />
      <div class="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
        <SegmentedControl
          v-model="viewMode"
          :options="viewOptions"
          group-label="Aggregation"
          emphasis="quiet"
        />
        <div class="min-w-0 sm:max-w-full">
          <slot name="range" />
        </div>
      </div>
    </div>
    <div
      class="h-64 rounded-sm p-3 transition-[background-color] duration-200"
      :style="{ backgroundColor: chartPalette.wash }"
    >
      <div class="h-full rounded-sm bg-graphite/55 px-1 pt-2 pb-1">
        <p
          v-if="!hasData"
          class="flex h-full items-center justify-center text-body text-mist"
        >
          <template v-if="metricMode === 'bmi'">
            No BMI data available.
          </template>
          <template v-else-if="metricMode === 'energy'">
            No active energy data yet. Connect Google Health to see it here.
          </template>
          <template v-else-if="metricMode === 'overwhelm'">
            No overwhelm entries yet.
          </template>
          <template v-else>
            No weight entries yet.
          </template>
        </p>
        <Bar
          v-else-if="metricMode === 'energy'"
          :data="chartData"
          :options="chartOptions"
        />
        <Line
          v-else
          :data="chartData"
          :options="chartOptions"
          :plugins="[bmiZonesPlugin, overwhelmBaselinePlugin]"
        />
      </div>
    </div>
    <p
      v-if="hasData && metricMode === 'bmi'"
      class="sr-only"
    >
      Background bands show the WHO BMI reference ranges: Overweight from {{ BMI_BOUNDARIES.overweight }}, Obese from {{ BMI_BOUNDARIES.obese }}.
      <template v-if="latestBmiEntry?.bmi != null">
        A dashed line marks your most recent reading: {{ latestBmiEntry.bmi.toFixed(1) }}, {{ category(latestBmiEntry.bmi) }}.
      </template>
    </p>
    <p
      v-if="hasData && metricMode === 'overwhelm'"
      class="sr-only"
    >
      A dashed line marks your baseline of {{ OVERWHELM_BASELINE }} on a 1 to 10 scale, where 10 is most overwhelmed.
    </p>
  </div>
</template>
