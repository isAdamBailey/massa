<script setup lang="ts" generic="T extends string">
import type { MetricAccent } from '~/composables/useMetricAccent'
import { accentActiveClasses } from '~/composables/useMetricAccent'

export interface SegmentedOption<T extends string = string> {
  value: T
  label: string
}

const model = defineModel<T>({ required: true })

withDefaults(defineProps<{
  options: SegmentedOption<T>[]
  groupLabel: string
  stretch?: boolean
  /** Primary uses the metric accent; quiet is for secondary toggles on the same surface. */
  emphasis?: 'primary' | 'quiet'
  accent?: MetricAccent
}>(), {
  emphasis: 'primary',
  accent: 'verdigris'
})
</script>

<template>
  <div
    role="group"
    :aria-label="groupLabel"
    class="flex gap-1 rounded-sm bg-graphite p-1"
  >
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
      :class="[
        stretch ? 'flex-1' : '',
        model === option.value
          ? (emphasis === 'quiet' ? 'bg-hairline text-mist' : accentActiveClasses(accent))
          : 'text-mist hover:bg-graphite-hover'
      ]"
      @click="model = option.value"
    >
      {{ option.label }}
    </button>
  </div>
</template>
