<script setup lang="ts">
import { format } from 'date-fns'
import type { WeightEntry } from '~/stores/weights'
import type { ChartMetricMode, LogTab } from '~/composables/useMetricAccent'
import type { ViewMode as ChartViewMode } from '~/components/MetricChart.vue'
import type { TimeSpanStop } from '~/components/TimeSpanSlider.vue'
import { accentForChartMetric } from '~/composables/useMetricAccent'
import { spanStart } from '~/composables/useRangeChange'

const auth = useAuthStore()
const weights = useWeightsStore()
const activeEnergy = useActiveEnergyStore()
const overwhelm = useOverwhelmStore()
const settings = useSettingsStore()
const google = useGoogleHealthStore()

type RangePreset = '7d' | '30d' | '90d' | '6m' | '1y' | 'all'

const rangePreset = ref<RangePreset>('90d')
const chartViewMode = ref<ChartViewMode>('weekly')
const chartMetricMode = ref<ChartMetricMode>('weight')
const logTab = ref<LogTab>('weight')

// Log is the focus control: when it changes, Trend follows. Trend can still
// be toggled independently afterward.
watch(logTab, (tab) => {
  chartMetricMode.value = tab
})

const rangePresets: TimeSpanStop<RangePreset>[] = [
  { value: '7d', tick: '7d', label: '7 days' },
  { value: '30d', tick: '30d', label: '30 days' },
  { value: '90d', tick: '90d', label: '90 days' },
  { value: '6m', tick: '6m', label: '6 months' },
  { value: '1y', tick: '1y', label: '1 year' },
  { value: 'all', tick: 'All', label: 'All time' }
]

const presetDays: Record<RangePreset, number | null> = {
  '7d': 7,
  '30d': 30,
  '90d': 90,
  '6m': 182,
  '1y': 365,
  all: null
}

// Short ranges default to a weekly average; long ranges have enough history
// that weekly gets noisy, so monthly is the more readable default. This is
// only the *default* applied when the time span changes - the user can still
// pick a different aggregation afterward, and it sticks until they change
// the time span again.
const defaultViewModeForPreset: Record<RangePreset, ChartViewMode> = {
  '7d': 'daily',
  '30d': 'weekly',
  '90d': 'weekly',
  '6m': 'monthly',
  '1y': 'monthly',
  all: 'monthly'
}

/**
 * All history is fetched once (on mount and after saves) and the time span
 * is applied on the client by MetricChart, so changing span - including
 * dragging the slider - never waits on the network. The list endpoints
 * aren't paginated, and a personal tracker's history is small.
 */
async function loadEntries() {
  await Promise.all([
    weights.fetchEntries(),
    activeEnergy.fetchEntries(),
    overwhelm.fetchEntries()
  ])
}

/** Applied when the user settles on a span (slider release, key press, tapped label). */
function applyDefaultViewMode(preset: RangePreset) {
  chartViewMode.value = defaultViewModeForPreset[preset]
}

const { computeWeightTrend, computeEnergyTrend, computeVerdict } = useWeekVerdict()
const { computeCurrentWeekSummary } = useOverwhelmSummary()
const { buildStatusSentence } = useStatusSentence()

/**
 * The weight tab of LogCard already syncs with Google (pulling in fresh
 * active energy data) before it saves the weight entry, so all that's left
 * is to refresh local data to reflect it - and, for the overwhelm tab, the
 * newly saved entry itself.
 *
 * Once the reload lands, return to the top so the status sentence and the
 * Trend summary - already reflecting the new entry - are what come into
 * view, and move focus there so keyboard and screen-reader users land on it
 * too instead of staying on the Save button. If the reload fails, stay put
 * so the error remains visible where it is.
 */
const statusSection = ref<HTMLElement | null>(null)
const pageHeading = ref<HTMLElement | null>(null)

async function refreshAfterSave() {
  await loadEntries()
  if (weights.error || activeEnergy.error || overwhelm.error) {
    return
  }
  await nextTick()
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  window.scrollTo({ top: 0, behavior: reduceMotion ? 'auto' : 'smooth' })
  // preventScroll so focusing doesn't cut the smooth scroll short.
  ;(statusSection.value ?? pageHeading.value)?.focus({ preventScroll: true })
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

/**
 * The verdict reads the last few weekly buckets that have data. It used to
 * see whatever span the chart had fetched (90 days by default); with all
 * history loaded it keeps that 90-day horizon explicitly, so a long gap
 * doesn't let months-old weeks speak for "this week".
 */
const VERDICT_HORIZON_DAYS = 90
const weekVerdict = computed(() => {
  const start = spanStart(VERDICT_HORIZON_DAYS)!
  return computeVerdict(
    computeWeightTrend(displayEntries.value.filter(e => new Date(e.recordedAt) >= start)),
    computeEnergyTrend(activeEnergy.entries.filter(e => e.day >= format(start, 'yyyy-MM-dd')))
  )
})

/** Only carries a clause in the status sentence when this week's avg overwhelm is over 4. */
const currentWeekOverwhelm = computed(() => computeCurrentWeekSummary(overwhelm.entries))

const statusSentence = computed(() => buildStatusSentence(
  latestEntry.value !== null,
  weekVerdict.value,
  currentWeekOverwhelm.value
))

onMounted(async () => {
  await Promise.all([loadEntries(), settings.fetchSettings(), google.fetchStatus()])
})


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
      <h1
        ref="pageHeading"
        tabindex="-1"
        class="sr-only"
      >
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
        v-if="statusSentence"
        ref="statusSection"
        tabindex="-1"
        class="rounded-md bg-slate p-5 outline-none"
      >
        <p class="text-pretty text-2xl font-semibold leading-snug text-mist sm:text-3xl">
          <template
            v-for="(segment, index) in statusSentence"
            :key="index"
          ><span
            v-if="segment.tag"
            class="text-cobalt"
          >{{ segment.text }}</span><span
            v-else-if="segment.highlight"
            class="text-verdigris"
          >{{ segment.text }}</span><template v-else>{{ segment.text }}</template></template>
        </p>
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
            :span-days="presetDays[rangePreset]"
          >
            <template #range>
              <TimeSpanSlider
                v-model="rangePreset"
                :stops="rangePresets"
                :accent="accentForChartMetric(chartMetricMode)"
                @commit="applyDefaultViewMode"
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
    </div>
  </div>
</template>
