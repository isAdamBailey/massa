<script setup lang="ts">
import 'chartjs-adapter-date-fns'
import { startOfWeek } from 'date-fns'
import {
  BarController,
  BarElement,
  Chart as ChartJS,
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
import { OVERWHELM_BASELINE } from '~/composables/useOverwhelm'
import type { SegmentedOption } from '~/components/SegmentedControl.vue'
import type { ActiveEnergyEntry } from '~/stores/activeEnergy'
import type { OverwhelmEntry } from '~/stores/overwhelm'
import type { UnitsPreference } from '~/stores/settings'
import type { WeightEntry } from '~/stores/weights'

ChartJS.register(LineController, LineElement, PointElement, BarController, BarElement, LinearScale, TimeScale, Tooltip)

const props = defineProps<{
  entries: WeightEntry[]
  activeEnergyEntries: ActiveEnergyEntry[]
  overwhelmEntries: OverwhelmEntry[]
  unitsPreference: UnitsPreference
}>()

const { kgToLb, category } = useBmi()
const { computeWeeklyAverageBy, computeWeeklyAverages, computeWeeklySumBy, toLocalDate } = useWeeklyAverages()

type ViewMode = 'daily' | 'weekly'
type MetricMode = 'weight' | 'bmi' | 'energy' | 'overwhelm'

const viewMode = defineModel<ViewMode>('viewMode', { default: 'weekly' })
const metricMode = defineModel<MetricMode>('metricMode', { default: 'weight' })

const metricOptions: SegmentedOption<MetricMode>[] = [
  { value: 'weight', label: 'Weight' },
  { value: 'bmi', label: 'BMI' },
  { value: 'energy', label: 'Active energy' },
  { value: 'overwhelm', label: 'Overwhelm' }
]

const viewOptions: SegmentedOption<ViewMode>[] = [
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' }
]

const VERDIGRIS = 'oklch(0.70 0.09 170)'
const FOG = 'oklch(0.64 0.01 170)'
const HAIRLINE = 'oklch(0.32 0.006 170)'
const GRAPHITE = 'oklch(0.28 0.006 170)'
const MIST = 'oklch(0.95 0.003 170)'
const AMBER = 'oklch(0.75 0.14 80)'
const AMBER_WASH = 'oklch(0.75 0.14 80 / 0.10)'
const EMBER = 'oklch(0.62 0.17 25)'
const EMBER_WASH = 'oklch(0.62 0.17 25 / 0.12)'

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
    ctx.fillStyle = 'oklch(0.22 0.005 170 / 0.85)'
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

// The top 3 tags by frequency per week (ties broken alphabetically), keyed
// by the same epoch-ms x value the weekly overwhelm dataset plots. Reasons
// don't average, so weekly mode summarizes rather than showing every tag.
const overwhelmWeeklyTopTags = computed<Record<number, string>>(() => {
  const countsByWeek = new Map<number, Map<string, number>>()
  for (const entry of props.overwhelmEntries) {
    if (!entry.tags.length) {
      continue
    }
    const weekStart = startOfWeek(toLocalDate(entry.day), { weekStartsOn: 1 }).getTime()
    const counts = countsByWeek.get(weekStart) ?? new Map<string, number>()
    for (const tag of entry.tags) {
      counts.set(tag.name, (counts.get(tag.name) ?? 0) + 1)
    }
    countsByWeek.set(weekStart, counts)
  }

  const map: Record<number, string> = {}
  for (const [weekStart, counts] of countsByWeek) {
    map[weekStart] = Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
      .slice(0, 3)
      .map(([name, count]) => `${name} ×${count}`)
      .join(' · ')
  }
  return map
})

const chartData = computed(() => {
  if (metricMode.value === 'overwhelm') {
    const data = viewMode.value === 'weekly'
      ? computeWeeklyAverageBy(props.overwhelmEntries, e => e.day, e => e.overwhelmLevel).map(w => ({ x: toLocalDate(w.weekStart).getTime(), y: w.average }))
      : props.overwhelmEntries.map(e => ({ x: toLocalDate(e.day).getTime(), y: e.overwhelmLevel }))

    return {
      datasets: [
        {
          label: viewMode.value === 'weekly' ? 'Weekly avg overwhelm' : 'Overwhelm',
          data,
          borderColor: VERDIGRIS,
          backgroundColor: VERDIGRIS,
          tension: 0.2,
          pointRadius: 3
        }
      ]
    }
  }

  if (metricMode.value === 'energy') {
    const data = viewMode.value === 'weekly'
      ? computeWeeklySumBy(props.activeEnergyEntries, e => e.day, e => e.activeEnergyKcal).map(w => ({ x: toLocalDate(w.weekStart).getTime(), y: w.total }))
      : props.activeEnergyEntries.map(e => ({ x: toLocalDate(e.day).getTime(), y: e.activeEnergyKcal }))

    return {
      datasets: [
        {
          label: viewMode.value === 'weekly' ? 'Weekly active energy (kcal)' : 'Active energy (kcal)',
          data,
          borderColor: VERDIGRIS,
          backgroundColor: VERDIGRIS
        }
      ]
    }
  }

  if (metricMode.value === 'bmi') {
    const data = viewMode.value === 'weekly'
      ? computeWeeklyAverageBy(props.entries, e => e.recordedAt, e => e.bmi).map(w => ({ x: new Date(w.weekStart).getTime(), y: w.average }))
      : props.entries.filter(e => e.bmi != null).map(e => ({ x: new Date(e.recordedAt).getTime(), y: e.bmi as number }))

    return {
      datasets: [
        {
          label: viewMode.value === 'weekly' ? 'Weekly avg BMI' : 'BMI',
          data,
          borderColor: VERDIGRIS,
          backgroundColor: VERDIGRIS,
          tension: 0.2,
          pointRadius: 3
        }
      ]
    }
  }

  const toDisplay = (kg: number) => props.unitsPreference === 'imperial' ? kgToLb(kg) : kg
  const unitLabel = props.unitsPreference === 'imperial' ? 'lb' : 'kg'

  const data = viewMode.value === 'weekly'
    ? computeWeeklyAverages(props.entries).map(w => ({ x: new Date(w.weekStart).getTime(), y: toDisplay(w.average) }))
    : props.entries.map(e => ({ x: new Date(e.recordedAt).getTime(), y: toDisplay(e.weightKg) }))

  return {
    datasets: [
      {
        label: viewMode.value === 'weekly' ? `Weekly avg (${unitLabel})` : `Weight (${unitLabel})`,
        data,
        borderColor: VERDIGRIS,
        backgroundColor: VERDIGRIS,
        tension: 0.2,
        pointRadius: 3
      }
    ]
  }
})

const hasData = computed(() => (chartData.value.datasets[0]?.data.length ?? 0) > 0)

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    x: {
      type: 'time' as const,
      time: { unit: viewMode.value === 'weekly' ? 'week' as const : 'day' as const },
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
      padding: 8,
      cornerRadius: 6,
      displayColors: false,
      callbacks: {
        footer: (items: { parsed: { x: number | null } }[]) => {
          const x = items[0]?.parsed.x
          if (metricMode.value !== 'overwhelm' || x == null) {
            return undefined
          }
          const tags = viewMode.value === 'weekly' ? overwhelmWeeklyTopTags.value[x] : overwhelmDailyTagNames.value[x]
          return tags || undefined
        }
      }
    }
  }
}))
</script>

<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <SegmentedControl
        v-model="metricMode"
        :options="metricOptions"
        aria-label="Metric"
      />
      <SegmentedControl
        v-model="viewMode"
        :options="viewOptions"
        aria-label="Range"
      />
    </div>
    <div class="h-64">
      <p
        v-if="!hasData"
        class="flex h-full items-center justify-center text-body text-fog"
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
