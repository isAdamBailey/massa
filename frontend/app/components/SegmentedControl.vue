<script setup lang="ts" generic="T extends string">
import type { MetricAccent } from '~/composables/useMetricAccent'
import { accentActiveClasses } from '~/composables/useMetricAccent'

export interface SegmentedOption<T extends string = string> {
  value: T
  label: string
}

const model = defineModel<T>({ required: true })

const props = withDefaults(defineProps<{
  options: SegmentedOption<T>[]
  groupLabel: string
  /** Primary only: fill the container width instead of sizing to the labels. */
  stretch?: boolean
  /** Primary is the metric focus; quiet is for secondary toggles on the same surface. */
  emphasis?: 'primary' | 'quiet'
  accent?: MetricAccent
}>(), {
  emphasis: 'primary',
  accent: 'verdigris'
})

const activeIndex = computed(() => props.options.findIndex(o => o.value === model.value))

/**
 * Primary is a single Graphite track of equal columns with one accent
 * indicator that slides under the active option. The track has 4px of
 * padding and 4px gaps (p-1, gap-1), so the indicator is one column wide and
 * moves by its own width plus a gap per step. Equal columns at every width
 * keep that arithmetic exact - and keep the metric switcher a single row.
 */
const trackStyle = computed(() => ({
  gridTemplateColumns: `repeat(${props.options.length}, minmax(0, 1fr))`
}))

const indicatorStyle = computed(() => ({
  width: `calc((100% - 8px - ${props.options.length - 1} * 4px) / ${props.options.length})`,
  transform: `translateX(calc(${activeIndex.value} * (100% + 4px)))`
}))
</script>

<template>
  <div
    v-if="emphasis === 'quiet'"
    role="group"
    :aria-label="groupLabel"
    class="flex flex-wrap items-center gap-1"
  >
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      :aria-pressed="model === option.value"
      class="min-h-11 rounded-sm px-3 text-label transition-colors duration-150"
      :class="model === option.value ? 'bg-hairline text-mist' : 'text-fog hover:text-mist'"
      @click="model = option.value"
    >
      {{ option.label }}
    </button>
  </div>

  <div
    v-else
    role="group"
    :aria-label="groupLabel"
    class="relative grid gap-1 rounded-sm bg-graphite p-1"
    :class="stretch ? 'w-full' : 'w-max max-w-full'"
    :style="trackStyle"
  >
    <!-- Decorative: the active button carries aria-pressed. Slides between
         options and cross-fades accent when the metric changes; both are
         instant under prefers-reduced-motion. -->
    <span
      v-if="activeIndex >= 0"
      aria-hidden="true"
      class="pointer-events-none absolute inset-y-1 left-1 rounded-sm transition-[transform,background-color] duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] motion-reduce:transition-none"
      :class="accentActiveClasses(accent)"
      :style="indicatorStyle"
    />
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      :aria-pressed="model === option.value"
      class="relative min-h-11 min-w-0 rounded-sm text-label transition-colors duration-150 motion-reduce:transition-none max-[359px]:text-[0.6875rem] max-[359px]:tracking-tight sm:px-1"
      :class="model === option.value ? 'text-carbon' : 'text-mist hover:bg-graphite-hover'"
      @click="model = option.value"
    >
      {{ option.label }}
    </button>
  </div>
</template>
