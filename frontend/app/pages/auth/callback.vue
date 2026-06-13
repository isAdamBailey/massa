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
  <div class="flex min-h-screen items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-sm space-y-4 text-center">
      <p
        v-if="status === 'verifying'"
        class="text-sm text-gray-600"
      >
        Signing you in…
      </p>
      <template v-else>
        <p class="text-sm text-red-600">
          This sign-in link is invalid or has expired.
        </p>
        <NuxtLink
          to="/login"
          class="text-sm font-medium text-blue-600 hover:underline"
        >
          Back to sign in
        </NuxtLink>
      </template>
    </div>
  </div>
</template>
