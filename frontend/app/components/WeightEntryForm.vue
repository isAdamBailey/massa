<script setup lang="ts">
const weights = useWeightsStore()
const settings = useSettingsStore()
const { lbToKg } = useBmi()

const emit = defineEmits<{ saved: [] }>()

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
      emit('saved')
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
