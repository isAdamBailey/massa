<script setup lang="ts">
import type { WeightEntry } from '~/stores/weights'

const auth = useAuthStore()
const weights = useWeightsStore()
const settings = useSettingsStore()
const google = useGoogleHealthStore()
const { category, kgToLb } = useBmi()

type RangePreset = '7d' | '30d' | '90d' | '1y' | 'all'
type ChartViewMode = 'daily' | 'weekly'
type ChartMetricMode = 'weight' | 'bmi'

const rangePreset = ref<RangePreset>('90d')
const chartViewMode = ref<ChartViewMode>('weekly')
const chartMetricMode = ref<ChartMetricMode>('weight')

const rangePresets: { value: RangePreset, label: string }[] = [
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '90d', label: '90 days' },
  { value: '1y', label: '1 year' },
  { value: 'all', label: 'All time' }
]

const presetDays: Record<RangePreset, number | null> = {
  '7d': 7,
  '30d': 30,
  '90d': 90,
  '1y': 365,
  all: null
}

async function loadEntries() {
  const days = presetDays[rangePreset.value]
  if (days === null) {
    await weights.fetchEntries()
    return
  }
  const from = new Date()
  from.setDate(from.getDate() - days)
  await weights.fetchEntries({ from: from.toISOString() })
}

const { currentWeekAverage } = useWeeklyAverages()

/**
 * Google sync can backfill a data point for a timestamp that already has a
 * manual entry, recomputed against whatever height was current at sync
 * time. When both exist for the same instant, prefer the manual one so
 * "latest" stats and the chart agree with the user's own input.
 */
const displayEntries = computed(() => {
  const byRecordedAt = new Map<string, WeightEntry>()
  for (const entry of weights.entries) {
    const existing = byRecordedAt.get(entry.recordedAt)
    if (!existing || (existing.source !== 'manual' && entry.source === 'manual')) {
      byRecordedAt.set(entry.recordedAt, entry)
    }
  }
  return Array.from(byRecordedAt.values()).sort((a, b) => a.recordedAt.localeCompare(b.recordedAt))
})

const latestEntry = computed(() => displayEntries.value.at(-1) ?? null)

const weightUnitLabel = computed(() => settings.settings.unitsPreference === 'imperial' ? 'lb' : 'kg')

const latestWeightDisplay = computed(() => {
  if (!latestEntry.value) {
    return null
  }
  const weight = settings.settings.unitsPreference === 'imperial'
    ? kgToLb(latestEntry.value.weightKg)
    : latestEntry.value.weightKg
  return weight.toFixed(1)
})

const weeklyAverageDisplay = computed(() => {
  const average = currentWeekAverage(displayEntries.value)
  if (!average) {
    return null
  }
  const weight = settings.settings.unitsPreference === 'imperial'
    ? kgToLb(average.average)
    : average.average
  return weight.toFixed(1)
})

onMounted(async () => {
  await Promise.all([loadEntries(), settings.fetchSettings(), google.fetchStatus()])
})

watch(rangePreset, loadEntries)

function formatDate(value?: string) {
  if (!value) {
    return 'never'
  }
  return new Date(value).toLocaleString()
}
</script>

<template>
  <div class="min-h-screen bg-carbon px-4 py-6 text-mist sm:px-6 sm:py-10">
    <div class="mx-auto flex max-w-3xl flex-col gap-4">
      <h1 class="sr-only">
        Dashboard
      </h1>
      <AppHeader />

      <p class="text-label text-fog">
        Signed in as {{ auth.user?.email }}
        <template v-if="google.status.connected">
          · Google Health synced {{ formatDate(google.status.lastIncrementalSyncAt) }}
        </template>
      </p>

      <section
        v-if="latestEntry"
        class="grid grid-cols-2 gap-x-6 gap-y-5 rounded-md bg-slate p-5 sm:grid-cols-4"
      >
        <div>
          <dt class="text-label text-fog">
            Latest weight
          </dt>
          <dd class="text-display font-mono tabular-nums text-verdigris">
            {{ latestWeightDisplay }}<span class="text-label font-sans text-fog"> {{ weightUnitLabel }}</span>
          </dd>
        </div>
        <div v-if="weeklyAverageDisplay">
          <dt class="text-label text-fog">
            This week's avg
          </dt>
          <dd class="text-display font-mono tabular-nums text-mist">
            {{ weeklyAverageDisplay }}<span class="text-label font-sans text-fog"> {{ weightUnitLabel }}</span>
          </dd>
        </div>
        <div v-if="latestEntry.bmi">
          <dt class="text-label text-fog">
            BMI
          </dt>
          <dd class="text-display font-mono tabular-nums text-mist">
            {{ latestEntry.bmi.toFixed(1) }}
          </dd>
        </div>
        <div v-if="latestEntry.bmi">
          <dt class="text-label text-fog">
            Category
          </dt>
          <dd class="text-title font-sans text-mist">
            {{ category(latestEntry.bmi) }}
          </dd>
        </div>
      </section>

      <section class="space-y-3 rounded-md bg-slate p-5">
        <h2 class="text-title font-sans">
          Add weight entry
        </h2>
        <WeightEntryForm />
      </section>

      <section class="space-y-4 rounded-md bg-slate p-5">
        <div class="flex flex-wrap gap-2">
          <button
            v-for="preset in rangePresets"
            :key="preset.value"
            type="button"
            class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
            :class="rangePreset === preset.value ? 'bg-verdigris text-carbon' : 'bg-graphite text-mist hover:bg-graphite-hover'"
            @click="rangePreset = preset.value"
          >
            {{ preset.label }}
          </button>
        </div>

        <p
          v-if="weights.loading"
          class="text-body text-fog"
        >
          Loading…
        </p>
        <WeightChart
          v-else
          v-model:view-mode="chartViewMode"
          v-model:metric-mode="chartMetricMode"
          :entries="displayEntries"
          :units-preference="settings.settings.unitsPreference"
        />

        <p
          v-if="weights.error"
          class="text-body text-ember"
        >
          {{ weights.error }}
        </p>
      </section>

      <section class="space-y-3 rounded-md bg-slate p-5">
        <h2 class="text-title font-sans">
          Recent entries (last 7 days)
        </h2>
        <WeightEntryList />
      </section>
    </div>
  </div>
</template>
