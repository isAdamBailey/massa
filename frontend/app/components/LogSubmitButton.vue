<script setup lang="ts">
defineProps<{
  submitting: boolean
  justSaved: boolean
}>()
</script>

<template>
  <button
    type="submit"
    :disabled="submitting"
    class="flex flex-1 items-center justify-center gap-1.5 rounded-sm bg-verdigris px-5 text-sm font-medium text-carbon transition-colors duration-150 hover:bg-verdigris-hover disabled:cursor-not-allowed disabled:opacity-70"
    style="min-width: 100px"
  >
    <Transition
      name="fade"
      mode="out-in"
    >
      <span
        v-if="justSaved"
        key="saved"
        class="flex items-center gap-1.5"
      >
        <svg
          class="h-4 w-4 shrink-0"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M5 12.5 9.5 17 19 7" />
        </svg>
        Added
      </span>
      <span
        v-else-if="submitting"
        key="submitting"
        class="flex items-center gap-1.5"
      >
        <svg
          class="h-4 w-4 shrink-0 animate-spinner"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          aria-hidden="true"
        >
          <circle
            cx="12"
            cy="12"
            r="9"
            stroke-width="2.5"
            stroke-linecap="round"
            stroke-dasharray="40 16"
          />
        </svg>
        Saving…
      </span>
      <span
        v-else
        key="idle"
        class="flex items-center gap-1.5"
      >
        <svg
          class="h-4 w-4 shrink-0"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M12 5v14M5 12h14" />
        </svg>
        Log
      </span>
    </Transition>
  </button>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 150ms ease-out;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .fade-enter-active,
  .fade-leave-active {
    transition: none;
  }
}
</style>
