<script setup lang="ts">
import { OVERWHELM_BASELINE } from '~/composables/useOverwhelm'
import { toDateLocalInput, toDateTimeLocalInput } from '~/composables/useLocalDateInput'
import {
  accentActiveClasses,
  accentFocusClasses,
  accentForLogTab,
  METRIC_ACCENT_OKLCH
} from '~/composables/useMetricAccent'
import type { SegmentedOption } from '~/components/SegmentedControl.vue'

const weights = useWeightsStore()
const overwhelm = useOverwhelmStore()
const overwhelmTags = useOverwhelmTagsStore()
const settings = useSettingsStore()
const google = useGoogleHealthStore()
const pendingWeight = useGooglePendingWeight()
const { lbToKg } = useBmi()
const { toLocalDate } = useWeeklyAverages()

onMounted(() => {
  overwhelmTags.fetchTags()
})

const emit = defineEmits<{ saved: [] }>()

type LogTab = 'weight' | 'overwhelm'

interface LogMetric {
  id: LogTab
  label: string
  loggedToday: () => boolean
}

// Ordered list of loggable metrics. Prefer the first still-outstanding metric
// when seeding the active tab; weight stays first so it wins ties.
const logMetrics: LogMetric[] = [
  {
    id: 'weight',
    label: 'Weight',
    loggedToday: () => weights.entries.some(e => toLocalDate(e.recordedAt).toDateString() === new Date().toDateString())
  },
  {
    id: 'overwhelm',
    label: 'Overwhelm',
    loggedToday: () => overwhelm.entries.some(e => e.day === toDateLocalInput())
  }
]

const tabOptions: SegmentedOption<LogTab>[] = logMetrics.map(m => ({ value: m.id, label: m.label }))

const activeTab = defineModel<LogTab>({ default: 'weight' })
const tabSeeded = ref(false)
const fetchSeen = ref(false)

const logAccent = computed(() => accentForLogTab(activeTab.value))
const logWash = computed(() => METRIC_ACCENT_OKLCH[logAccent.value].wash)

// The form mounts before the page's onMounted fetch resolves, so both stores
// start empty. Wait until a fetch has started and finished before seeding —
// immediate:true on loading alone would latch on the idle-before-fetch state.
watch(
  () => weights.loading || overwhelm.loading,
  (busy) => {
    if (busy) {
      fetchSeen.value = true
      return
    }
    if (!fetchSeen.value || tabSeeded.value) {
      return
    }
    tabSeeded.value = true
    const outstanding = logMetrics.find(m => !m.loggedToday())
    activeTab.value = outstanding?.id ?? 'weight'
  }
)

// --- Weight tab ---

const weightInput = ref('')
const dateInput = ref(toDateTimeLocalInput())
const submitting = ref(false)
const justSaved = ref(false)
const formError = ref<string | null>(null)

let savedTimeout: ReturnType<typeof setTimeout> | undefined

const unitLabel = computed(() => settings.settings.unitsPreference === 'imperial' ? 'lb' : 'kg')

async function onSubmit() {
  formError.value = null

  const value = Number(weightInput.value)
  if (!weightInput.value || !(value > 0)) {
    formError.value = 'Enter a valid weight.'
    return
  }
  if (!dateInput.value) {
    formError.value = 'Enter a date and time.'
    return
  }

  const weightKg = settings.settings.unitsPreference === 'imperial' ? lbToKg(value) : value
  const recordedAt = new Date(dateInput.value).toISOString()

  submitting.value = true
  try {
    // Sync with Google Health before logging the weight, not after: this is
    // where a stale/revoked Google connection surfaces (reconnect_required).
    // Catching it here means we never save a weight entry we can't also
    // account for in that day's synced energy data. Skipped entirely while
    // syncing is paused.
    if (google.status.connected && google.status.syncEnabled && !google.syncing) {
      await google.sync()
      if (google.reconnectRequired) {
        // Stash the entry and hand off to Google's consent screen instead of
        // blocking the save: app/plugins/google-resume.client.ts picks it
        // back up once the app reloads with a restored connection.
        pendingWeight.set({ weightKg, recordedAt })
        if (!(await google.connect())) {
          pendingWeight.clear()
          formError.value = google.error ?? 'Failed to reconnect Google Health. Please try again.'
        }
        return
      }
    }

    const entry = await weights.createEntry({ weightKg, recordedAt })
    if (entry) {
      weightInput.value = ''
      dateInput.value = toDateTimeLocalInput()
      clearTimeout(savedTimeout)
      justSaved.value = true
      savedTimeout = setTimeout(() => {
        justSaved.value = false
      }, 1800)
      emit('saved')
    } else {
      formError.value = weights.error
    }
  } finally {
    submitting.value = false
  }
}

// --- Overwhelm tab ---

const overwhelmLevel = ref<number | null>(null)
const overwhelmDate = ref(toDateLocalInput())
const selectedTagIds = ref<string[]>([])
const overwhelmSubmitting = ref(false)
const overwhelmJustSaved = ref(false)
const overwhelmError = ref<string | null>(null)

let overwhelmSavedTimeout: ReturnType<typeof setTimeout> | undefined

function toggleTag(id: string) {
  selectedTagIds.value = selectedTagIds.value.includes(id)
    ? selectedTagIds.value.filter(tagId => tagId !== id)
    : [...selectedTagIds.value, id]
}

async function onSubmitOverwhelm() {
  overwhelmError.value = null

  if (overwhelmLevel.value === null) {
    overwhelmError.value = 'Choose a level from 1 to 10.'
    return
  }
  if (!overwhelmDate.value) {
    overwhelmError.value = 'Enter a date.'
    return
  }

  overwhelmSubmitting.value = true
  try {
    // No Google Health sync here: overwhelm has no counterpart there.
    const entry = await overwhelm.saveEntry({ day: overwhelmDate.value, overwhelmLevel: overwhelmLevel.value, tagIds: selectedTagIds.value })
    if (entry) {
      // Leave the level, tags, and date selected — it's a rating upsert, not
      // a new row, so blanking the date (or the answer) makes corrections harder.
      clearTimeout(overwhelmSavedTimeout)
      overwhelmJustSaved.value = true
      overwhelmSavedTimeout = setTimeout(() => {
        overwhelmJustSaved.value = false
      }, 1800)
      emit('saved')
    } else {
      overwhelmError.value = overwhelm.error
    }
  } finally {
    overwhelmSubmitting.value = false
  }
}

onUnmounted(() => {
  clearTimeout(savedTimeout)
  clearTimeout(overwhelmSavedTimeout)
})
</script>

<template>
  <div class="space-y-3">
    <SegmentedControl
      v-model="activeTab"
      :options="tabOptions"
      group-label="Metric"
      :accent="logAccent"
      stretch
    />

    <div
      class="rounded-sm p-3 transition-[background-color] duration-200"
      :style="{ backgroundColor: logWash }"
    >
      <form
        v-if="activeTab === 'weight'"
        class="space-y-3"
        @submit.prevent="onSubmit"
      >
        <div class="flex items-stretch gap-2">
          <div class="relative min-w-0 flex-1">
            <label
              for="weight-input"
              class="mb-1.5 block text-label text-fog"
            >Weight ({{ unitLabel }})</label>
            <div class="relative">
              <input
                id="weight-input"
                v-model="weightInput"
                type="number"
                step="0.1"
                min="0"
                inputmode="decimal"
                placeholder="0.0"
                autocomplete="off"
                class="w-full rounded-sm bg-graphite py-4 pl-4 pr-12 font-mono text-2xl tabular-nums text-mist placeholder:text-fog/30 focus:outline-2 focus:outline-offset-2"
                :class="accentFocusClasses(logAccent)"
              >
              <span
                class="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 font-sans text-label text-fog"
                aria-hidden="true"
              >{{ unitLabel }}</span>
            </div>
          </div>

          <div class="flex shrink-0 flex-col">
            <span
              class="mb-1.5 block text-label opacity-0"
              aria-hidden="true"
            >Log</span>
            <LogSubmitButton
              :submitting="submitting"
              :just-saved="justSaved"
              :accent="logAccent"
            />
          </div>
        </div>

        <div>
          <label
            for="date-input"
            class="mb-1.5 block text-label text-fog"
          >Date and time</label>
          <input
            id="date-input"
            v-model="dateInput"
            type="datetime-local"
            class="w-full rounded-sm bg-graphite px-3 py-2 text-body text-mist"
          >
        </div>

        <p
          v-if="formError"
          class="text-body text-ember"
        >
          {{ formError }}
        </p>
      </form>

      <form
        v-else-if="activeTab === 'overwhelm'"
        class="space-y-3"
        @submit.prevent="onSubmitOverwhelm"
      >
        <div>
          <span
            id="overwhelm-label"
            class="mb-1.5 block text-label text-fog"
          >How overwhelmed today?</span>
          <div
            role="radiogroup"
            aria-labelledby="overwhelm-label"
            class="flex gap-1"
          >
            <button
              v-for="n in 10"
              :key="n"
              type="button"
              role="radio"
              :aria-checked="overwhelmLevel === n"
              :aria-label="n === OVERWHELM_BASELINE ? `${n}, baseline` : String(n)"
              class="relative min-w-0 flex-1 rounded-sm py-3 font-mono text-body tabular-nums transition-colors duration-150 focus:outline-2 focus:outline-offset-2"
              :class="[
                accentFocusClasses(logAccent),
                overwhelmLevel === n ? accentActiveClasses(logAccent) : 'bg-graphite text-mist hover:bg-graphite-hover'
              ]"
              @click="overwhelmLevel = n"
            >
              {{ n }}
              <span
                v-if="n === OVERWHELM_BASELINE"
                class="absolute bottom-0.5 left-1/2 h-0.5 w-3 -translate-x-1/2 rounded-full"
                :class="overwhelmLevel === n ? 'bg-carbon/60' : 'bg-fog'"
                aria-hidden="true"
              />
            </button>
          </div>
          <p class="mt-1.5 text-label text-fog">
            1 = calm · {{ OVERWHELM_BASELINE }} = your baseline · 10 = most overwhelmed
          </p>
        </div>

        <div v-if="overwhelmTags.tags.length">
          <span
            id="overwhelm-tags-label"
            class="mb-1.5 block text-label text-fog"
          >Why? (optional)</span>
          <div
            role="group"
            aria-labelledby="overwhelm-tags-label"
            class="flex flex-wrap gap-1"
          >
            <button
              v-for="tag in overwhelmTags.tags"
              :key="tag.id"
              type="button"
              :aria-pressed="selectedTagIds.includes(tag.id)"
              class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
              :class="selectedTagIds.includes(tag.id) ? accentActiveClasses(logAccent) : 'bg-graphite text-mist hover:bg-graphite-hover'"
              @click="toggleTag(tag.id)"
            >
              {{ tag.name }}
            </button>
          </div>
        </div>
        <p
          v-else
          class="text-label text-fog"
        >
          <NuxtLink
            to="/settings"
            class="underline hover:text-mist"
          >Add tags in Settings</NuxtLink> to describe why, if you want to.
        </p>

        <div class="flex items-stretch gap-2">
          <div class="min-w-0 flex-1">
            <label
              for="overwhelm-date-input"
              class="mb-1.5 block text-label text-fog"
            >Date</label>
            <input
              id="overwhelm-date-input"
              v-model="overwhelmDate"
              type="date"
              class="w-full rounded-sm bg-graphite px-3 py-2 text-body text-mist"
            >
          </div>

          <div class="flex shrink-0 flex-col">
            <span
              class="mb-1.5 block text-label opacity-0"
              aria-hidden="true"
            >Log</span>
            <LogSubmitButton
              :submitting="overwhelmSubmitting"
              :just-saved="overwhelmJustSaved"
              :accent="logAccent"
            />
          </div>
        </div>

        <p
          v-if="overwhelmError"
          class="text-body text-ember"
        >
          {{ overwhelmError }}
        </p>
      </form>
    </div>
  </div>
</template>
