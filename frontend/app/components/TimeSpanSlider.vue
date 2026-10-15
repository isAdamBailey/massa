<script setup lang="ts" generic="T extends string">
import type { MetricAccent } from '~/composables/useMetricAccent'
import { METRIC_ACCENT_OKLCH } from '~/composables/useMetricAccent'

export interface TimeSpanStop<T extends string = string> {
  value: T
  /** Short tick label under the track, e.g. "90d". */
  tick: string
  /** Spoken/visible span name, e.g. "90 days". */
  label: string
}

const model = defineModel<T>({ required: true })

const props = defineProps<{
  stops: TimeSpanStop<T>[]
  accent: MetricAccent
}>()

/**
 * `update:modelValue` fires live on every `input` (while dragging), so the
 * chart and summary follow the thumb; `commit` fires once on `change`
 * (release, a key press, or a tapped label) for anything that should only
 * happen when the user settles on a span.
 */
const emit = defineEmits<{ commit: [value: T] }>()

const index = computed(() => Math.max(0, props.stops.findIndex(s => s.value === model.value)))
const current = computed(() => props.stops[index.value]!)
const last = computed(() => props.stops.length - 1)
const fraction = computed(() => (last.value === 0 ? 0 : index.value / last.value))
const accentSolid = computed(() => METRIC_ACCENT_OKLCH[props.accent].solid)

function valueAt(position: number): T {
  return props.stops[Math.min(last.value, Math.max(0, position))]!.value
}

function onInput(event: Event) {
  model.value = valueAt(Number((event.target as HTMLInputElement).value))
}

function onChange(event: Event) {
  emit('commit', valueAt(Number((event.target as HTMLInputElement).value)))
}

function jumpTo(position: number) {
  const value = valueAt(position)
  model.value = value
  emit('commit', value)
}
</script>

<template>
  <div class="w-full">
    <div class="flex items-baseline justify-between gap-3">
      <span
        id="time-span-label"
        class="text-label text-fog"
      >Time span</span>
      <span
        class="text-label text-mist"
        aria-hidden="true"
      >{{ current.label }}</span>
    </div>
    <!-- The rail, fill and dots are inset by half the thumb (11px) on each
         side, which is where a native range input's thumb centre travels. -->
    <div
      class="time-span relative h-11"
      :style="{ '--accent': accentSolid }"
    >
      <div
        class="pointer-events-none absolute inset-x-[11px] top-1/2 h-1 -translate-y-1/2 rounded-full bg-graphite"
        aria-hidden="true"
      >
        <div
          class="absolute inset-y-0 left-0 rounded-full"
          :style="{ width: `${fraction * 100}%`, backgroundColor: 'var(--accent)' }"
        />
        <span
          v-for="(stop, i) in stops"
          :key="stop.value"
          class="absolute top-1/2 size-1.5 -translate-x-1/2 -translate-y-1/2 rounded-full"
          :class="i <= index ? '' : 'bg-hairline'"
          :style="{ left: `${(i / last) * 100}%`, backgroundColor: i <= index ? 'var(--accent)' : undefined }"
        />
      </div>
      <input
        type="range"
        min="0"
        :max="last"
        step="1"
        :value="index"
        aria-labelledby="time-span-label"
        :aria-valuetext="current.label"
        class="absolute inset-0 h-full w-full cursor-pointer appearance-none bg-transparent"
        @input="onInput"
        @change="onChange"
      >
    </div>
    <div
      class="relative h-6"
      aria-hidden="true"
    >
      <button
        v-for="(stop, i) in stops"
        :key="stop.value"
        type="button"
        tabindex="-1"
        class="absolute top-0 -translate-x-1/2 px-1 text-label transition-colors duration-150"
        :class="i === index ? 'font-semibold text-mist' : 'text-fog hover:text-mist'"
        :style="{ left: `calc(11px + (100% - 22px) * ${i / last})` }"
        @click="jumpTo(i)"
      >
        {{ stop.tick }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.time-span input::-webkit-slider-runnable-track {
  height: 100%;
  background: transparent;
}

.time-span input::-webkit-slider-thumb {
  appearance: none;
  width: 22px;
  height: 22px;
  margin-top: calc((2.75rem - 22px) / 2);
  border-radius: 9999px;
  background: var(--color-mist);
  box-shadow: 0 0 0 3px var(--color-slate);
}

.time-span input::-moz-range-track {
  background: transparent;
}

.time-span input::-moz-range-thumb {
  width: 22px;
  height: 22px;
  border: 0;
  border-radius: 9999px;
  background: var(--color-mist);
  box-shadow: 0 0 0 3px var(--color-slate);
}

.time-span input:focus {
  outline: none;
}

.time-span input:focus-visible::-webkit-slider-thumb {
  box-shadow: 0 0 0 3px var(--color-slate), 0 0 0 5px var(--accent);
}

.time-span input:focus-visible::-moz-range-thumb {
  box-shadow: 0 0 0 3px var(--color-slate), 0 0 0 5px var(--accent);
}
</style>
