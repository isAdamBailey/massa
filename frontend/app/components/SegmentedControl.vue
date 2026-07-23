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
  /** Primary is the metric focus; quiet is for secondary toggles on the same surface. */
  emphasis?: 'primary' | 'quiet'
  accent?: MetricAccent
  /**
   * Horizontal scroll instead of wrap — for longer quiet option sets (e.g.
   * time spans) so every option stays a full-size tap target on phones.
   */
  scrollable?: boolean
}>(), {
  emphasis: 'primary',
  accent: 'verdigris'
})
</script>

<template>
  <div
    role="group"
    :aria-label="groupLabel"
    :class="[
      emphasis === 'quiet'
        ? (scrollable
            ? 'flex max-w-full gap-1 overflow-x-auto overscroll-x-contain touch-pan-x scrollbar-none'
            : 'flex flex-wrap items-center gap-1')
        : (stretch
            ? 'grid grid-cols-2 gap-1 rounded-sm bg-graphite p-1 sm:flex'
            : 'flex gap-1 rounded-sm bg-graphite p-1')
    ]"
  >
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="min-h-11 rounded-sm px-3 text-label transition-colors duration-150"
      :class="[
        stretch ? 'sm:flex-1' : '',
        scrollable ? 'shrink-0' : '',
        model === option.value
          ? (emphasis === 'quiet' ? 'bg-hairline text-mist' : accentActiveClasses(accent))
          : (emphasis === 'quiet' ? 'text-fog hover:text-mist' : 'text-mist hover:bg-graphite-hover')
      ]"
      @click="model = option.value"
    >
      {{ option.label }}
    </button>
  </div>
</template>
