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
const { computeWeeklyAverages } = useWeeklyAverages()

type ViewMode = 'daily' | 'weekly'

const viewMode = defineModel<ViewMode>('viewMode', { default: 'daily' })

const chartData = computed(() => {
  const toDisplay = (kg: number) => props.unitsPreference === 'imperial' ? kgToLb(kg) : kg
  const unitLabel = props.unitsPreference === 'imperial' ? 'lb' : 'kg'

  const data = viewMode.value === 'weekly'
    ? computeWeeklyAverages(props.entries).map(w => ({ x: new Date(w.weekStart).getTime(), y: toDisplay(w.averageKg) }))
    : props.entries.map(e => ({ x: new Date(e.recordedAt).getTime(), y: toDisplay(e.weightKg) }))

  return {
    datasets: [
      {
        label: viewMode.value === 'weekly' ? `Weekly avg (${unitLabel})` : `Weight (${unitLabel})`,
        data,
        borderColor: '#2563eb',
        backgroundColor: '#2563eb',
        tension: 0.2,
        pointRadius: 3
      }
    ]
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    x: {
      type: 'time' as const,
      time: { unit: viewMode.value === 'weekly' ? 'week' as const : 'day' as const }
    },
    y: {
      beginAtZero: false
    }
  }
}))
</script>

<template>
  <div>
    <div class="mb-2 flex justify-end gap-2">
      <button
        type="button"
        class="rounded-md px-3 py-1 text-sm font-medium"
        :class="viewMode === 'daily' ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-800 hover:bg-gray-300'"
        @click="viewMode = 'daily'"
      >
        Daily
      </button>
      <button
        type="button"
        class="rounded-md px-3 py-1 text-sm font-medium"
        :class="viewMode === 'weekly' ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-800 hover:bg-gray-300'"
        @click="viewMode = 'weekly'"
      >
        Weekly
      </button>
    </div>
    <div class="h-64">
      <p
        v-if="!entries.length"
        class="flex h-full items-center justify-center text-sm text-gray-600"
      >
        No weight entries yet.
      </p>
      <Line
        v-else
        :data="chartData"
        :options="chartOptions"
      />
    </div>
  </div>
</template>
