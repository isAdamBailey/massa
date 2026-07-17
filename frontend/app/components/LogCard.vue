<script setup lang="ts">
const weights = useWeightsStore()
const overwhelm = useOverwhelmStore()
const overwhelmTags = useOverwhelmTagsStore()
const settings = useSettingsStore()
const google = useGoogleHealthStore()
const { lbToKg } = useBmi()
const { toLocalDate } = useWeeklyAverages()

onMounted(() => {
  overwhelmTags.fetchTags()
})

const emit = defineEmits<{ saved: [] }>()

function nowForInput(): string {
  const date = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function todayKey(): string {
  const date = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

type LogTab = 'weight' | 'overwhelm'
const activeTab = ref<LogTab>('weight')
const tabSeeded = ref(false)

const loggedWeightToday = computed(() =>
  weights.entries.some(e => toLocalDate(e.recordedAt).toDateString() === new Date().toDateString()))
const loggedOverwhelmToday = computed(() =>
  overwhelm.entries.some(e => e.day === todayKey()))

// The form mounts before the page's onMounted fetch resolves, so both stores
// start empty; a computed default would misfire at first paint and again
// flip the tab out from under the user after a save. Seed once, when the
// data first arrives, instead.
watch(
  () => weights.loading || overwhelm.loading,
  (busy) => {
    if (busy || tabSeeded.value) {
      return
    }
    tabSeeded.value = true
    // Weight is logged in the morning and overwhelm in the evening, so open
    // on whichever is still outstanding; weight wins when both or neither are.
    if (loggedWeightToday.value && !loggedOverwhelmToday.value) {
      activeTab.value = 'overwhelm'
    }
  },
  { immediate: true }
)

// --- Weight tab ---

const weightInput = ref('')
const dateInput = ref(nowForInput())
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
    // account for in that day's synced energy data.
    if (google.status.connected && !google.syncing) {
      await google.sync()
      if (google.reconnectRequired) {
        formError.value = google.error ?? 'Google Health needs to reconnect. Please reconnect before logging a new weight.'
        return
      }
    }

    const entry = await weights.createEntry({ weightKg, recordedAt })
    if (entry) {
      weightInput.value = ''
      dateInput.value = nowForInput()
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

const OVERWHELM_BASELINE = 3

const overwhelmLevel = ref<number | null>(null)
const overwhelmDate = ref(todayKey())
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
      // Leave the level and tags selected - it's a rating, not a number
      // field, and blanking it discards the answer and makes a correction
      // re-tap harder.
      overwhelmDate.value = todayKey()
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
    <div
      role="group"
      aria-label="Metric"
      class="flex gap-1 rounded-sm bg-graphite p-1"
    >
      <button
        type="button"
        class="flex-1 rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
        :class="activeTab === 'weight' ? 'bg-verdigris text-carbon' : 'text-mist hover:bg-graphite-hover'"
        @click="activeTab = 'weight'"
      >
        Weight
      </button>
      <button
        type="button"
        class="flex-1 rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
        :class="activeTab === 'overwhelm' ? 'bg-verdigris text-carbon' : 'text-mist hover:bg-graphite-hover'"
        @click="activeTab = 'overwhelm'"
      >
        Overwhelm
      </button>
    </div>

    <form
      v-if="activeTab === 'weight'"
      class="space-y-3"
      @submit.prevent="onSubmit"
    >
      <!-- Primary action row: weight input + submit button, same height, side by side -->
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
              class="w-full rounded-sm bg-graphite py-4 pl-4 pr-12 font-mono text-2xl tabular-nums text-mist placeholder:text-fog/30 focus:outline-2 focus:outline-verdigris focus:outline-offset-2"
            >
            <span
              class="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 font-sans text-label text-fog"
              aria-hidden="true"
            >{{ unitLabel }}</span>
          </div>
        </div>

        <!-- Invisible label-height spacer keeps the button's own height equal
             to the input's, since items-stretch matches column heights but
             the button has no label pushing it down. -->
        <div class="flex shrink-0 flex-col">
          <span
            class="mb-1.5 block text-label opacity-0"
            aria-hidden="true"
          >Log</span>
          <button
            type="submit"
            :disabled="submitting"
            class="flex flex-1 items-center justify-center gap-1.5 rounded-sm bg-verdigris px-5 text-sm font-medium text-carbon transition-colors duration-150 hover:bg-verdigris-hover disabled:cursor-not-allowed disabled:opacity-70"
            style="min-width: 100px"
          >
            <Transition
              name="fade"
              mode="out-in"
            >
              <span
                v-if="justSaved"
                key="saved"
                class="flex items-center gap-1.5"
              >
                <svg
                  class="h-4 w-4 shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="M5 12.5 9.5 17 19 7" />
                </svg>
                Added
              </span>
              <span
                v-else-if="submitting"
                key="submitting"
                class="flex items-center gap-1.5"
              >
                <svg
                  class="h-4 w-4 shrink-0 animate-spinner"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  aria-hidden="true"
                >
                  <circle
                    cx="12"
                    cy="12"
                    r="9"
                    stroke-width="2.5"
                    stroke-linecap="round"
                    stroke-dasharray="40 16"
                  />
                </svg>
                Saving…
              </span>
              <span
                v-else
                key="idle"
                class="flex items-center gap-1.5"
              >
                <svg
                  class="h-4 w-4 shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="M12 5v14M5 12h14" />
                </svg>
                Log
              </span>
            </Transition>
          </button>
        </div>
      </div>

      <!-- Date field: secondary, below the primary row -->
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
      v-else
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
            class="relative min-w-0 flex-1 rounded-sm py-3 font-mono text-body tabular-nums transition-colors duration-150 focus:outline-2 focus:outline-verdigris focus:outline-offset-2"
            :class="overwhelmLevel === n ? 'bg-verdigris text-carbon' : 'bg-graphite text-mist hover:bg-graphite-hover'"
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
          1 = calm · 3 = your baseline · 10 = most overwhelmed
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
            :class="selectedTagIds.includes(tag.id) ? 'bg-verdigris text-carbon' : 'bg-graphite text-mist hover:bg-graphite-hover'"
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
          <button
            type="submit"
            :disabled="overwhelmSubmitting"
            class="flex flex-1 items-center justify-center gap-1.5 rounded-sm bg-verdigris px-5 text-sm font-medium text-carbon transition-colors duration-150 hover:bg-verdigris-hover disabled:cursor-not-allowed disabled:opacity-70"
            style="min-width: 100px"
          >
            <Transition
              name="fade"
              mode="out-in"
            >
              <span
                v-if="overwhelmJustSaved"
                key="saved"
                class="flex items-center gap-1.5"
              >
                <svg
                  class="h-4 w-4 shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="M5 12.5 9.5 17 19 7" />
                </svg>
                Added
              </span>
              <span
                v-else-if="overwhelmSubmitting"
                key="submitting"
                class="flex items-center gap-1.5"
              >
                <svg
                  class="h-4 w-4 shrink-0 animate-spinner"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  aria-hidden="true"
                >
                  <circle
                    cx="12"
                    cy="12"
                    r="9"
                    stroke-width="2.5"
                    stroke-linecap="round"
                    stroke-dasharray="40 16"
                  />
                </svg>
                Saving…
              </span>
              <span
                v-else
                key="idle"
                class="flex items-center gap-1.5"
              >
                <svg
                  class="h-4 w-4 shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="M12 5v14M5 12h14" />
                </svg>
                Log
              </span>
            </Transition>
          </button>
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
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 150ms ease-out;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .fade-enter-active,
  .fade-leave-active {
    transition: none;
  }
}
</style>
