<script setup lang="ts">
import { OVERWHELM_BASELINE } from '~/composables/useOverwhelm'
import { toDateLocalInput } from '~/composables/useLocalDateInput'
import {
  accentActiveClasses,
  accentButtonClasses,
  accentFocusClasses
} from '~/composables/useMetricAccent'
import type { OverwhelmEntry } from '~/stores/overwhelm'
import type { OverwhelmTag } from '~/stores/overwhelmTags'

const overwhelm = useOverwhelmStore()
const overwhelmTags = useOverwhelmTagsStore()

const editingDay = ref<string | null>(null)
const editLevel = ref<number | null>(null)
const editTagIds = ref<string[]>([])
const savingDay = ref<string | null>(null)
const rowError = ref<string | null>(null)

onMounted(() => {
  overwhelmTags.fetchTags()
})

const sortedEntries = computed(() => {
  const cutoff = new Date()
  cutoff.setDate(cutoff.getDate() - 7)
  const cutoffDay = toDateLocalInput(cutoff)
  return overwhelm.entries
    .filter(e => e.day >= cutoffDay)
    .sort((a, b) => b.day.localeCompare(a.day))
})

/** Active vocabulary plus any tags still attached to the entry being edited. */
const editableTags = computed((): OverwhelmTag[] => {
  const byId = new Map(overwhelmTags.tags.map(t => [t.id, t]))
  if (editingDay.value) {
    const entry = sortedEntries.value.find(e => e.day === editingDay.value)
    for (const tag of entry?.tags ?? []) {
      if (!byId.has(tag.id)) {
        byId.set(tag.id, tag)
      }
    }
  }
  return Array.from(byId.values()).sort((a, b) => a.name.localeCompare(b.name))
})

function formatDay(day: string): string {
  const [y, m, d] = day.split('-').map(Number)
  if (y == null || m == null || d == null) {
    return day
  }
  return new Date(y, m - 1, d).toLocaleDateString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric'
  })
}

function tagSummary(entry: OverwhelmEntry): string {
  if (!entry.tags.length) {
    return ''
  }
  return entry.tags.map(t => t.name).sort((a, b) => a.localeCompare(b)).join(' · ')
}

function startEdit(entry: OverwhelmEntry) {
  editingDay.value = entry.day
  editLevel.value = entry.overwhelmLevel
  editTagIds.value = entry.tags.map(t => t.id)
  rowError.value = null
}

function cancelEdit() {
  editingDay.value = null
  editLevel.value = null
  editTagIds.value = []
  rowError.value = null
}

function toggleTag(id: string) {
  editTagIds.value = editTagIds.value.includes(id)
    ? editTagIds.value.filter(tagId => tagId !== id)
    : [...editTagIds.value, id]
}

async function saveEdit(day: string) {
  rowError.value = null

  if (editLevel.value === null) {
    rowError.value = 'Choose a level from 1 to 10.'
    return
  }

  savingDay.value = day
  try {
    const entry = await overwhelm.saveEntry({
      day,
      overwhelmLevel: editLevel.value,
      tagIds: editTagIds.value
    })
    if (entry) {
      cancelEdit()
    } else {
      rowError.value = overwhelm.error
    }
  } finally {
    savingDay.value = null
  }
}
</script>

<template>
  <div>
    <p
      v-if="!sortedEntries.length"
      class="text-body text-fog"
    >
      No overwhelm entries in the last 7 days.
    </p>
    <ul
      v-else
      class="max-h-[50vh] divide-y divide-hairline overflow-y-auto"
    >
      <li
        v-for="entry in sortedEntries"
        :key="entry.day"
        class="py-3 first:pt-0 last:pb-0"
      >
        <div
          v-if="editingDay === entry.day"
          class="space-y-3"
        >
          <div>
            <p class="mb-1.5 text-label text-fog">
              {{ formatDay(entry.day) }}
            </p>
            <div
              role="radiogroup"
              :aria-label="`Overwhelm level for ${formatDay(entry.day)}`"
              class="flex gap-1"
            >
              <button
                v-for="n in 10"
                :key="n"
                type="button"
                role="radio"
                :aria-checked="editLevel === n"
                :aria-label="n === OVERWHELM_BASELINE ? `${n}, baseline` : String(n)"
                class="relative min-w-0 flex-1 rounded-sm py-2.5 font-mono text-body tabular-nums transition-colors duration-150 focus:outline-2 focus:outline-offset-2"
                :class="[
                  accentFocusClasses('cobalt'),
                  editLevel === n ? accentActiveClasses('cobalt') : 'bg-graphite text-mist hover:bg-graphite-hover'
                ]"
                @click="editLevel = n"
              >
                {{ n }}
                <span
                  v-if="n === OVERWHELM_BASELINE"
                  class="absolute bottom-0.5 left-1/2 h-0.5 w-3 -translate-x-1/2 rounded-full"
                  :class="editLevel === n ? 'bg-carbon/60' : 'bg-fog'"
                  aria-hidden="true"
                />
              </button>
            </div>
          </div>

          <div v-if="editableTags.length">
            <span class="mb-1.5 block text-label text-fog">Tags</span>
            <div
              role="group"
              aria-label="Tags"
              class="flex flex-wrap gap-1"
            >
              <button
                v-for="tag in editableTags"
                :key="tag.id"
                type="button"
                :aria-pressed="editTagIds.includes(tag.id)"
                class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
                :class="editTagIds.includes(tag.id) ? accentActiveClasses('cobalt') : 'bg-graphite text-mist hover:bg-graphite-hover'"
                @click="toggleTag(tag.id)"
              >
                {{ tag.name }}
              </button>
            </div>
          </div>

          <div class="flex gap-2">
            <button
              type="button"
              :disabled="savingDay === entry.day"
              class="rounded-sm px-3 py-2 text-label text-carbon transition-colors duration-150 disabled:opacity-50"
              :class="accentButtonClasses('cobalt')"
              @click="saveEdit(entry.day)"
            >
              {{ savingDay === entry.day ? 'Saving…' : 'Save' }}
            </button>
            <button
              type="button"
              class="rounded-sm bg-graphite px-3 py-2 text-label text-mist transition-colors duration-150 hover:bg-graphite-hover"
              @click="cancelEdit"
            >
              Cancel
            </button>
          </div>
          <p
            v-if="rowError"
            class="text-body text-ember"
          >
            {{ rowError }}
          </p>
        </div>

        <div
          v-else
          class="flex items-center justify-between gap-2"
        >
          <div class="min-w-0">
            <p class="text-body text-mist">
              <span class="font-mono tabular-nums">{{ entry.overwhelmLevel }}</span>
              <span class="text-fog"> / 10</span>
              <span
                v-if="tagSummary(entry)"
                class="text-fog"
              > · {{ tagSummary(entry) }}</span>
            </p>
            <p class="text-label text-fog">
              {{ formatDay(entry.day) }}
            </p>
          </div>
          <button
            type="button"
            class="shrink-0 rounded-sm bg-graphite px-3 py-1.5 text-label text-mist transition-colors duration-150 hover:bg-graphite-hover"
            @click="startEdit(entry)"
          >
            Edit
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
