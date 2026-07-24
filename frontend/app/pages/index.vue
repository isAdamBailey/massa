<script setup lang="ts">
import type { WeightEntry } from '~/stores/weights'
import type { ChartMetricMode, LogTab } from '~/composables/useMetricAccent'

const auth = useAuthStore()
const weights = useWeightsStore()
const activeEnergy = useActiveEnergyStore()
const overwhelm = useOverwhelmStore()
const settings = useSettingsStore()
const google = useGoogleHealthStore()
const { kgToLb } = useBmi()

type RangePreset = '7d' | '30d' | '90d' | '6m' | '1y' | 'all'
type ChartViewMode = 'daily' | 'weekly'

const rangePreset = ref<RangePreset>('90d')
const chartViewMode = ref<ChartViewMode>('weekly')
const chartMetricMode = ref<ChartMetricMode>('weight')
const logTab = ref<LogTab>('weight')
const recentTab = ref<LogTab>('weight')

// Log is the focus control: when it changes, Trend and Recent follow.
// Those sections can still be toggled independently afterward.
watch(logTab, (tab) => {
  chartMetricMode.value = tab
  recentTab.value = tab
})

const rangePresets: { value: RangePreset, label: string }[] = [
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '90d', label: '90 days' },
  { value: '6m', label: '6 months' },
  { value: '1y', label: '1 year' },
  { value: 'all', label: 'All time' }
]

const presetDays: Record<RangePreset, number | null> = {
  '7d': 7,
  '30d': 30,
  '90d': 90,
  '6m': 182,
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
    activeEnergy.fetchEntries(from ? { from } : {}),
    overwhelm.fetchEntries(from ? { from } : {})
  ])
}

const { computeWeightTrend, computeEnergyTrend, computeVerdict, verdictLabel } = useWeekVerdict()
const { computeCurrentWeekSummary, elevatedTagParts } = useOverwhelmSummary()

/**
 * The weight tab of LogCard already syncs with Google (pulling in fresh
 * active energy data) before it saves the weight entry, so all that's left
 * is to refresh local data to reflect it - and, for the overwhelm tab, the
 * newly saved entry itself.
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

/** Only surfaces when this week's avg overwhelm is over 4. */
const elevatedWeekOverwhelm = computed(() => {
  const summary = computeCurrentWeekSummary(overwhelm.entries)
  return summary?.elevated ? summary : null
})
const elevatedWeekOverwhelmAverage = computed(() => {
  if (!elevatedWeekOverwhelm.value) {
    return null
  }
  return elevatedWeekOverwhelm.value.average.toFixed(1)
})
const elevatedWeekOverwhelmTagParts = computed(() => {
  if (!elevatedWeekOverwhelm.value) {
    return null
  }
  return elevatedTagParts(elevatedWeekOverwhelm.value.topTags)
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
        v-if="latestEntry || elevatedWeekOverwhelm"
        class="grid grid-cols-2 gap-x-6 gap-y-5 rounded-md bg-slate p-5"
      >
        <div v-if="latestEntry">
          <dt class="text-label text-fog">
            Latest weight
          </dt>
          <dd class="text-display font-mono tabular-nums text-verdigris">
            {{ latestWeightDisplay }}<span class="text-label font-sans text-fog"> {{ weightUnitLabel }}</span>
          </dd>
        </div>
        <div v-if="latestEntry">
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

        <div
          v-if="elevatedWeekOverwhelm"
          class="col-span-2 grid grid-cols-2 gap-x-6 gap-y-3 border-t border-hairline pt-5"
        >
          <div>
            <dt class="text-label text-fog">
              Avg overwhelm
            </dt>
            <dd class="text-display font-mono tabular-nums text-cobalt">
              {{ elevatedWeekOverwhelmAverage }}<span class="text-label font-sans text-fog"> / 10</span>
            </dd>
          </div>
          <div v-if="elevatedWeekOverwhelmTagParts">
            <dt class="text-label text-fog">
              Tags
            </dt>
            <dd class="pt-1 text-body text-mist text-pretty">
              {{ elevatedWeekOverwhelmTagParts.lead }}
              <template
                v-for="(tag, index) in elevatedWeekOverwhelmTagParts.tags"
                :key="tag"
              >
                <span v-if="index > 0">,</span>
                {{ ' ' }}<span class="font-medium text-cobalt">{{ tag }}</span>
              </template>
              {{ ' ' }}{{ elevatedWeekOverwhelmTagParts.trail }}
            </dd>
          </div>
        </div>
      </section>

      <section class="space-y-3 rounded-md bg-slate p-5">
        <div class="flex items-baseline justify-between gap-3">
          <h2 class="text-title font-sans">
            Log
          </h2>
          <p class="text-label text-fog">
            Today
          </p>
        </div>
        <LogCard
          v-model="logTab"
          @saved="refreshAfterSave"
        />
      </section>

      <section class="space-y-4 rounded-md bg-slate p-5">
        <h2 class="text-title font-sans">
          Trend
        </h2>

        <div
          :aria-busy="weights.loading || activeEnergy.loading || overwhelm.loading"
        >
          <MetricChart
            v-model:view-mode="chartViewMode"
            v-model:metric-mode="chartMetricMode"
            :entries="displayEntries"
            :active-energy-entries="activeEnergy.entries"
            :overwhelm-entries="overwhelm.entries"
            :units-preference="settings.settings.unitsPreference"
          >
            <template #range>
              <SegmentedControl
                v-model="rangePreset"
                :options="rangePresets"
                group-label="Time span"
                emphasis="quiet"
                scrollable
              />
            </template>
          </MetricChart>
        </div>

        <p
          v-if="weights.error"
          class="text-body text-ember"
        >
          {{ weights.error }}
        </p>
      </section>

      <section class="rounded-md bg-slate p-5">
        <RecentEntries v-model="recentTab" />
      </section>
    </div>
  </div>
</template>
