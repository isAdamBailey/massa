<script setup lang="ts">
import type { SegmentedOption } from '~/components/SegmentedControl.vue'
import { accentForLogTab } from '~/composables/useMetricAccent'
import type { LogTab } from '~/composables/useMetricAccent'

const activeTab = defineModel<LogTab>({ default: 'weight' })

const weights = useWeightsStore()
const overwhelm = useOverwhelmStore()

const tabOptions: SegmentedOption<LogTab>[] = [
  { value: 'weight', label: 'Weight' },
  { value: 'overwhelm', label: 'Overwhelm' }
]

const listAccent = computed(() => accentForLogTab(activeTab.value))

// The lists render "nothing in the last 7 days" for an empty store, so a
// failed load needs saying explicitly or it reads as no entries. The store's
// error is shared with row edits/deletes (shown inline on the row), so only
// treat it as a load failure when nothing has loaded.
const loadError = computed(() => {
  const store = activeTab.value === 'weight' ? weights : overwhelm
  return store.entries.length === 0 ? store.error : null
})
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-baseline justify-between gap-3">
      <h2 class="text-title font-sans">
        Entry history
      </h2>
      <p class="text-label text-fog">
        Last 7 days
      </p>
    </div>

    <SegmentedControl
      v-model="activeTab"
      :options="tabOptions"
      group-label="Entry type"
      :accent="listAccent"
      stretch
    />

    <p
      v-if="loadError"
      class="text-body text-ember"
    >
      {{ loadError }}
    </p>
    <WeightEntryList v-else-if="activeTab === 'weight'" />
    <OverwhelmEntryList v-else />
  </div>
</template>
