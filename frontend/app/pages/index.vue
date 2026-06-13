<script setup lang="ts">
const auth = useAuthStore()
const weights = useWeightsStore()
const settings = useSettingsStore()
const google = useGoogleHealthStore()
const { category, kgToLb } = useBmi()

type RangePreset = '7d' | '30d' | '90d' | '1y' | 'all'
type ChartViewMode = 'daily' | 'weekly'
type ChartMetricMode = 'weight' | 'bmi'

const rangePreset = ref<RangePreset>('90d')
const chartViewMode = ref<ChartViewMode>('daily')
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

const latestEntry = computed(() => weights.entries.at(-1) ?? null)

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
  const average = currentWeekAverage(weights.entries)
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

async function onLogout() {
  await auth.logout()
  await navigateTo('/login')
}

function formatDate(value?: string) {
  if (!value) {
    return 'Never'
  }
  return new Date(value).toLocaleString()
}
</script>

<template>
  <div class="min-h-screen bg-gray-50 px-4 py-8">
    <div class="mx-auto max-w-2xl space-y-4">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-semibold text-gray-900">
          Massa
        </h1>
        <div class="flex gap-2">
          <NuxtLink
            to="/settings"
            class="rounded-md bg-gray-200 px-3 py-2 text-sm font-medium text-gray-800 hover:bg-gray-300"
          >
            Settings
          </NuxtLink>
          <button
            type="button"
            class="rounded-md bg-gray-200 px-3 py-2 text-sm font-medium text-gray-800 hover:bg-gray-300"
            @click="onLogout"
          >
            Log out
          </button>
        </div>
      </div>

      <p class="text-sm text-gray-600">
        Signed in as {{ auth.user?.email }}
      </p>

      <div
        v-if="google.status.connected"
        class="rounded-md border border-gray-200 bg-white p-3 text-sm text-gray-600"
      >
        Google Health connected. Last synced {{ formatDate(google.status.lastIncrementalSyncAt) }}.
      </div>

      <section
        v-if="latestEntry"
        class="grid grid-cols-2 gap-3 rounded-md border border-gray-200 bg-white p-4 sm:grid-cols-3"
      >
        <div>
          <dt class="text-xs text-gray-500">
            Latest weight
          </dt>
          <dd class="text-lg font-semibold text-gray-900">
            {{ latestWeightDisplay }} {{ weightUnitLabel }}
          </dd>
        </div>
        <div v-if="weeklyAverageDisplay">
          <dt class="text-xs text-gray-500">
            This week's avg
          </dt>
          <dd class="text-lg font-semibold text-gray-900">
            {{ weeklyAverageDisplay }} {{ weightUnitLabel }}
          </dd>
        </div>
        <div v-if="latestEntry.bmi">
          <dt class="text-xs text-gray-500">
            BMI
          </dt>
          <dd class="text-lg font-semibold text-gray-900">
            {{ latestEntry.bmi.toFixed(1) }}
          </dd>
        </div>
        <div v-if="latestEntry.bmi">
          <dt class="text-xs text-gray-500">
            Category
          </dt>
          <dd class="text-lg font-semibold text-gray-900">
            {{ category(latestEntry.bmi) }}
          </dd>
        </div>
      </section>

      <section class="space-y-3 rounded-md border border-gray-200 bg-white p-4">
        <h2 class="text-lg font-medium text-gray-900">
          Add weight entry
        </h2>
        <WeightEntryForm />
      </section>

      <section class="space-y-3 rounded-md border border-gray-200 bg-white p-4">
        <div class="flex flex-wrap gap-2">
          <button
            v-for="preset in rangePresets"
            :key="preset.value"
            type="button"
            class="rounded-md px-3 py-1 text-sm font-medium"
            :class="rangePreset === preset.value ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-800 hover:bg-gray-300'"
            @click="rangePreset = preset.value"
          >
            {{ preset.label }}
          </button>
        </div>

        <p
          v-if="weights.loading"
          class="text-sm text-gray-600"
        >
          Loading…
        </p>
        <WeightChart
          v-else
          v-model:view-mode="chartViewMode"
          v-model:metric-mode="chartMetricMode"
          :entries="weights.entries"
          :units-preference="settings.settings.unitsPreference"
        />

        <p
          v-if="weights.error"
          class="text-sm text-red-600"
        >
          {{ weights.error }}
        </p>
      </section>

      <section class="space-y-3 rounded-md border border-gray-200 bg-white p-4">
        <h2 class="text-lg font-medium text-gray-900">
          Recent entries (last 7 days)
        </h2>
        <WeightEntryList />
      </section>
    </div>
  </div>
</template>
