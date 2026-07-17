<script setup lang="ts" generic="T extends string">
export interface SegmentedOption<T extends string = string> {
  value: T
  label: string
}

const model = defineModel<T>({ required: true })

defineProps<{
  options: SegmentedOption<T>[]
  ariaLabel: string
  stretch?: boolean
}>()
</script>

<template>
  <div
    role="group"
    :aria-label="ariaLabel"
    class="flex gap-1 rounded-sm bg-graphite p-1"
  >
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="rounded-sm px-3 py-1.5 text-label transition-colors duration-150"
      :class="[
        stretch ? 'flex-1' : '',
        model === option.value ? 'bg-verdigris text-carbon' : 'text-mist hover:bg-graphite-hover'
      ]"
      @click="model = option.value"
    >
      {{ option.label }}
    </button>
  </div>
</template>
