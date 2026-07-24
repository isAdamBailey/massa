<script setup lang="ts">
import type { SegmentedOption } from '~/components/SegmentedControl.vue'
import { accentForLogTab } from '~/composables/useMetricAccent'
import type { LogTab } from '~/composables/useMetricAccent'

const activeTab = defineModel<LogTab>({ default: 'weight' })

const tabOptions: SegmentedOption<LogTab>[] = [
  { value: 'weight', label: 'Weight' },
  { value: 'overwhelm', label: 'Overwhelm' }
]

const listAccent = computed(() => accentForLogTab(activeTab.value))
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-baseline justify-between gap-3">
      <h2 class="text-title font-sans">
        Recent entries
      </h2>
      <p class="text-label text-fog">
        Last 7 days
      </p>
    </div>

    <SegmentedControl
      v-model="activeTab"
      :options="tabOptions"
      group-label="Recent entry type"
      :accent="listAccent"
      stretch
    />

    <WeightEntryList v-if="activeTab === 'weight'" />
    <OverwhelmEntryList v-else />
  </div>
</template>
