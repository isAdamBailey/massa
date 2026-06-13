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
const formError = ref<string | null>(null)

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
    } else {
      formError.value = weights.error
    }
  } finally {
    submitting.value = false
  }
}
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
      class="rounded-sm bg-verdigris px-4 py-2 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover disabled:opacity-50"
    >
      {{ submitting ? 'Saving…' : 'Add entry' }}
    </button>
  </form>
</template>
