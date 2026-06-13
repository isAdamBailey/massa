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

const chartData = computed(() => ({
  datasets: [
    {
      label: props.unitsPreference === 'imperial' ? 'Weight (lb)' : 'Weight (kg)',
      data: props.entries.map(e => ({
        x: new Date(e.recordedAt).getTime(),
        y: props.unitsPreference === 'imperial' ? kgToLb(e.weightKg) : e.weightKg
      })),
      borderColor: '#2563eb',
      backgroundColor: '#2563eb',
      tension: 0.2,
      pointRadius: 3
    }
  ]
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    x: {
      type: 'time' as const,
      time: { unit: 'day' as const }
    },
    y: {
      beginAtZero: false
    }
  }
}
</script>

<template>
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
</template>
