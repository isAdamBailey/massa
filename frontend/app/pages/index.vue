<script setup lang="ts">
import type { WeightEntry } from '~/stores/weights'

const auth = useAuthStore()
const weights = useWeightsStore()
const activeEnergy = useActiveEnergyStore()
const settings = useSettingsStore()
const google = useGoogleHealthStore()
const { kgToLb } = useBmi()

type RangePreset = '7d' | '30d' | '90d' | '1y' | 'all'
type ChartViewMode = 'daily' | 'weekly'
type ChartMetricMode = 'weight' | 'bmi' | 'energy'

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

function currentRangeFrom(): string | undefined {
  const days = presetDays[rangePreset.value]
  if (days === null) {
    return undefined
  }
  const from = new Date()
  from.setDate(from.getDate() - days)
  return from.toISOString()
}

async function loadEntries() {
  const from = currentRangeFrom()
  await Promise.all([
    weights.fetchEntries(from ? { from } : {}),
    activeEnergy.fetchEntries(from ? { from } : {})
  ])
}

const { computeWeightTrend, computeEnergyTrend, computeVerdict, verdictLabel } = useWeekVerdict()

/**
 * WeightEntryForm already syncs with Google (pulling in fresh active energy
 * data) before it saves the weight entry, so all that's left is to refresh
 * local data to reflect both.
 */
async function refreshAfterSave() {
  await loadEntries()
}

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

const weekVerdict = computed(() => computeVerdict(computeWeightTrend(displayEntries.value), computeEnergyTrend(activeEnergy.entries)))
const weekVerdictLabel = computed(() => verdictLabel(weekVerdict.value))

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
        v-if="google.reconnectRequired"
        class="flex flex-wrap items-center justify-between gap-3 rounded-md bg-slate p-4"
      >
        <p class="text-body text-mist">
          Google Health needs to reconnect to keep syncing your data.
        </p>
        <button
          type="button"
          class="shrink-0 rounded-sm bg-verdigris px-4 py-2 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover"
          @click="google.connect"
        >
          Reconnect
        </button>
      </section>

      <section
        v-if="latestEntry"
        class="grid grid-cols-2 gap-x-6 gap-y-5 rounded-md bg-slate p-5"
      >
        <div>
          <dt class="text-label text-fog">
            Latest weight
          </dt>
          <dd class="text-display font-mono tabular-nums text-verdigris">
            {{ latestWeightDisplay }}<span class="text-label font-sans text-fog"> {{ weightUnitLabel }}</span>
          </dd>
        </div>
        <div>
          <dt class="text-label text-fog">
            This week
          </dt>
          <dd class="flex items-center gap-2 pt-1">
            <svg
              v-if="weekVerdict === 'better'"
              class="h-6 w-6 shrink-0 text-verdigris"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <path d="M12 19V5M5 12l7-7 7 7" />
            </svg>
            <svg
              v-else-if="weekVerdict === 'worse'"
              class="h-6 w-6 shrink-0 text-fog"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <path d="M12 5v14M19 12l-7 7-7-7" />
            </svg>
            <svg
              v-else
              class="h-6 w-6 shrink-0 text-fog"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <path d="M5 12h14" />
            </svg>
            <span
              class="text-title font-sans"
              :class="weekVerdict === 'better' ? 'text-verdigris' : 'text-mist'"
            >{{ weekVerdictLabel }}</span>
          </dd>
        </div>
      </section>

      <section class="space-y-3 rounded-md bg-slate p-5">
        <h2 class="text-title font-sans">
          Add weight entry
        </h2>
        <WeightEntryForm @saved="refreshAfterSave" />
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
          v-if="weights.loading || activeEnergy.loading"
          class="text-body text-fog"
        >
          Loading…
        </p>
        <WeightChart
          v-else
          v-model:view-mode="chartViewMode"
          v-model:metric-mode="chartMetricMode"
          :entries="displayEntries"
          :active-energy-entries="activeEnergy.entries"
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
