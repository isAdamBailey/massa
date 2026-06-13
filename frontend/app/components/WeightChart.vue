<script setup lang="ts">
import 'chartjs-adapter-date-fns'
import {
  Chart as ChartJS,
  LinearScale,
  LineController,
  LineElement,
  PointElement,
  TimeScale,
  Tooltip
} from 'chart.js'
import { Line } from 'vue-chartjs'
import type { UnitsPreference } from '~/stores/settings'
import type { WeightEntry } from '~/stores/weights'

ChartJS.register(LineController, LineElement, PointElement, LinearScale, TimeScale, Tooltip)

const props = defineProps<{
  entries: WeightEntry[]
  unitsPreference: UnitsPreference
}>()

const { kgToLb } = useBmi()
const { computeWeeklyAverageBy, computeWeeklyAverages } = useWeeklyAverages()

type ViewMode = 'daily' | 'weekly'
type MetricMode = 'weight' | 'bmi'

const viewMode = defineModel<ViewMode>('viewMode', { default: 'daily' })
const metricMode = defineModel<MetricMode>('metricMode', { default: 'weight' })

const VERDIGRIS = 'oklch(0.70 0.09 170)'
const FOG = 'oklch(0.64 0.01 170)'
const HAIRLINE = 'oklch(0.32 0.006 170)'
const GRAPHITE = 'oklch(0.28 0.006 170)'
const MIST = 'oklch(0.95 0.003 170)'

const chartData = computed(() => {
  if (metricMode.value === 'bmi') {
    const data = viewMode.value === 'weekly'
      ? computeWeeklyAverageBy(props.entries, e => e.bmi).map(w => ({ x: new Date(w.weekStart).getTime(), y: w.average }))
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
    y: {
      beginAtZero: false,
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
      displayColors: false
    }
  }
}))
</script>

<template>
  <div>
    <div class="mb-3 flex flex-wrap justify-between gap-2">
      <div class="flex gap-2">
        <button
          type="button"
          class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
          :class="metricMode === 'weight' ? 'bg-verdigris text-carbon' : 'bg-graphite text-mist hover:bg-graphite-hover'"
          @click="metricMode = 'weight'"
        >
          Weight
        </button>
        <button
          type="button"
          class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
          :class="metricMode === 'bmi' ? 'bg-verdigris text-carbon' : 'bg-graphite text-mist hover:bg-graphite-hover'"
          @click="metricMode = 'bmi'"
        >
          BMI
        </button>
      </div>
      <div class="flex gap-2">
        <button
          type="button"
          class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
          :class="viewMode === 'daily' ? 'bg-verdigris text-carbon' : 'bg-graphite text-mist hover:bg-graphite-hover'"
          @click="viewMode = 'daily'"
        >
          Daily
        </button>
        <button
          type="button"
          class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
          :class="viewMode === 'weekly' ? 'bg-verdigris text-carbon' : 'bg-graphite text-mist hover:bg-graphite-hover'"
          @click="viewMode = 'weekly'"
        >
          Weekly
        </button>
      </div>
    </div>
    <div class="h-64">
      <p
        v-if="!hasData"
        class="flex h-full items-center justify-center text-body text-fog"
      >
        {{ metricMode === 'bmi' ? 'No BMI data available.' : 'No weight entries yet.' }}
      </p>
      <Line
        v-else
        :data="chartData"
        :options="chartOptions"
      />
    </div>
  </div>
</template>
