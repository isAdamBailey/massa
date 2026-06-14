<script setup lang="ts">
const auth = useAuthStore()
const route = useRoute()

const status = ref<'verifying' | 'error'>('verifying')

onMounted(async () => {
  const token = route.query.token

  if (typeof token !== 'string' || token === '') {
    status.value = 'error'
    return
  }

  try {
    await auth.verifyMagicLink(token)
    await navigateTo('/')
  } catch {
    status.value = 'error'
  }
})
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-carbon px-4 py-6 text-mist sm:px-6 sm:py-10">
    <div class="mx-auto flex w-full max-w-3xl justify-center">
      <div class="w-full max-w-sm space-y-8">
        <div class="space-y-1 text-center">
          <h1 class="text-headline font-sans">
            Massa
          </h1>
          <p class="text-body text-fog">
            Sign in
          </p>
        </div>

        <Transition
          name="fade"
          mode="out-in"
        >
          <p
            v-if="status === 'verifying'"
            key="verifying"
            class="text-center text-body text-fog"
          >
            Signing you in…
          </p>

          <div
            v-else
            key="error"
            class="space-y-4 text-center"
          >
            <p class="text-body text-ember">
              This sign-in link is invalid or has expired.
            </p>
            <NuxtLink
              to="/login"
              class="inline-block rounded-sm bg-verdigris px-4 py-2 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover"
            >
              Back to sign in
            </NuxtLink>
          </div>
        </Transition>
      </div>
    </div>
  </div>
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
