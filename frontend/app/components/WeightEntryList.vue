<script setup lang="ts">
import type { WeightEntry } from '~/stores/weights'

const weights = useWeightsStore()
const settings = useSettingsStore()
const { kgToLb, lbToKg } = useBmi()

const editingId = ref<string | null>(null)
const editWeightInput = ref('')
const editDateInput = ref('')
const savingId = ref<string | null>(null)
const deletingId = ref<string | null>(null)
const rowError = ref<string | null>(null)

const unitLabel = computed(() => settings.settings.unitsPreference === 'imperial' ? 'lb' : 'kg')

const sortedEntries = computed(() => {
  const cutoff = new Date()
  cutoff.setDate(cutoff.getDate() - 7)
  return weights.entries
    .filter(e => new Date(e.recordedAt) >= cutoff)
    .sort((a, b) => b.recordedAt.localeCompare(a.recordedAt))
})

function displayWeight(weightKg: number): string {
  const weight = settings.settings.unitsPreference === 'imperial' ? kgToLb(weightKg) : weightKg
  return weight.toFixed(1)
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString()
}

function toDateTimeInput(value: string): string {
  const date = new Date(value)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function startEdit(entry: WeightEntry) {
  editingId.value = entry.id
  rowError.value = null
  const weight = settings.settings.unitsPreference === 'imperial' ? kgToLb(entry.weightKg) : entry.weightKg
  editWeightInput.value = weight.toFixed(1)
  editDateInput.value = toDateTimeInput(entry.recordedAt)
}

function cancelEdit() {
  editingId.value = null
  rowError.value = null
}

async function saveEdit(id: string) {
  rowError.value = null

  const value = Number(editWeightInput.value)
  if (!editWeightInput.value || !(value > 0)) {
    rowError.value = 'Enter a valid weight.'
    return
  }
  if (!editDateInput.value) {
    rowError.value = 'Enter a date and time.'
    return
  }

  const weightKg = settings.settings.unitsPreference === 'imperial' ? lbToKg(value) : value
  const recordedAt = new Date(editDateInput.value).toISOString()

  savingId.value = id
  try {
    const entry = await weights.updateEntry(id, { weightKg, recordedAt })
    if (entry) {
      editingId.value = null
    } else {
      rowError.value = weights.error
    }
  } finally {
    savingId.value = null
  }
}

async function onDelete(id: string) {
  if (!confirm('Delete this entry?')) {
    return
  }
  deletingId.value = id
  try {
    await weights.deleteEntry(id)
  } finally {
    deletingId.value = null
  }
}
</script>

<template>
  <div class="space-y-2">
    <p
      v-if="!sortedEntries.length"
      class="text-sm text-gray-600"
    >
      No entries in the last 7 days.
    </p>
    <ul
      v-else
      class="divide-y divide-gray-200"
    >
      <li
        v-for="entry in sortedEntries"
        :key="entry.id"
        class="py-2"
      >
        <div
          v-if="editingId === entry.id"
          class="flex flex-wrap items-end gap-2"
        >
          <div class="flex-1">
            <label
              :for="`edit-weight-${entry.id}`"
              class="block text-xs text-gray-500"
            >Weight ({{ unitLabel }})</label>
            <input
              :id="`edit-weight-${entry.id}`"
              v-model="editWeightInput"
              type="number"
              step="0.1"
              min="0"
              class="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
            >
          </div>
          <div class="flex-1">
            <label
              :for="`edit-date-${entry.id}`"
              class="block text-xs text-gray-500"
            >Date and time</label>
            <input
              :id="`edit-date-${entry.id}`"
              v-model="editDateInput"
              type="datetime-local"
              class="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
            >
          </div>
          <div class="flex gap-2">
            <button
              type="button"
              :disabled="savingId === entry.id"
              class="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              @click="saveEdit(entry.id)"
            >
              {{ savingId === entry.id ? 'Saving…' : 'Save' }}
            </button>
            <button
              type="button"
              class="rounded-md bg-gray-200 px-3 py-2 text-sm font-medium text-gray-800 hover:bg-gray-300"
              @click="cancelEdit"
            >
              Cancel
            </button>
          </div>
          <p
            v-if="rowError"
            class="w-full text-sm text-red-600"
          >
            {{ rowError }}
          </p>
        </div>

        <div
          v-else
          class="flex items-center justify-between gap-2"
        >
          <div>
            <p class="text-sm font-medium text-gray-900">
              {{ displayWeight(entry.weightKg) }} {{ unitLabel }}
              <span
                v-if="entry.bmi"
                class="text-gray-500"
              >· BMI {{ entry.bmi.toFixed(1) }}</span>
            </p>
            <p class="text-xs text-gray-500">
              {{ formatDateTime(entry.recordedAt) }}
              <span v-if="entry.source === 'google'">· Google Health</span>
            </p>
          </div>
          <div class="flex gap-2">
            <button
              type="button"
              class="rounded-md bg-gray-200 px-3 py-1 text-sm font-medium text-gray-800 hover:bg-gray-300"
              @click="startEdit(entry)"
            >
              Edit
            </button>
            <button
              type="button"
              :disabled="deletingId === entry.id"
              class="rounded-md bg-red-100 px-3 py-1 text-sm font-medium text-red-700 hover:bg-red-200 disabled:opacity-50"
              @click="onDelete(entry.id)"
            >
              {{ deletingId === entry.id ? 'Deleting…' : 'Delete' }}
            </button>
          </div>
        </div>
      </li>
    </ul>
  </div>
</template>
