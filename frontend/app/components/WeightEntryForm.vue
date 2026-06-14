<script setup lang="ts">
const weights = useWeightsStore()
const settings = useSettingsStore()
const { lbToKg } = useBmi()

function nowForInput(): string {
  const date = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

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
    const entry = await weights.createEntry({ weightKg, recordedAt })
    if (entry) {
      weightInput.value = ''
      dateInput.value = nowForInput()
      clearTimeout(savedTimeout)
      justSaved.value = true
      savedTimeout = setTimeout(() => {
        justSaved.value = false
      }, 1800)
    } else {
      formError.value = weights.error
    }
  } finally {
    submitting.value = false
  }
}

onUnmounted(() => clearTimeout(savedTimeout))
</script>

<template>
  <form
    class="space-y-3"
    @submit.prevent="onSubmit"
  >
    <div class="flex flex-wrap gap-3">
      <div class="flex-1">
        <label
          for="weight-input"
          class="block text-label text-fog"
        >Weight ({{ unitLabel }})</label>
        <input
          id="weight-input"
          v-model="weightInput"
          type="number"
          step="0.1"
          min="0"
          class="mt-1 w-full rounded-sm bg-graphite px-3 py-2 text-body text-mist"
        >
      </div>
      <div class="flex-1">
        <label
          for="date-input"
          class="block text-label text-fog"
        >Date and time</label>
        <input
          id="date-input"
          v-model="dateInput"
          type="datetime-local"
          class="mt-1 w-full rounded-sm bg-graphite px-3 py-2 text-body text-mist"
        >
      </div>
    </div>

    <p
      v-if="formError"
      class="text-body text-ember"
    >
      {{ formError }}
    </p>

    <button
      type="submit"
      :disabled="submitting"
      class="flex w-full items-center justify-center gap-2 rounded-sm bg-verdigris px-5 py-2.5 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover disabled:cursor-not-allowed disabled:opacity-70 sm:w-auto sm:min-w-32"
    >
      <Transition
        name="fade"
        mode="out-in"
      >
        <span
          v-if="justSaved"
          key="saved"
          class="flex items-center gap-2"
        >
          <svg
            class="h-4 w-4"
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
          class="flex items-center gap-2"
        >
          <svg
            class="h-4 w-4 animate-spinner"
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
          class="flex items-center gap-2"
        >
          <svg
            class="h-4 w-4"
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
          Add entry
        </span>
      </Transition>
    </button>
  </form>
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
