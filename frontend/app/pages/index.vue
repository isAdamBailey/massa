<script setup lang="ts">
const auth = useAuthStore()

async function onLogout() {
  await auth.logout()
  await navigateTo('/login')
}
</script>

<template>
  <div class="min-h-screen bg-gray-50 px-4 py-8">
    <div class="mx-auto max-w-md space-y-4">
      <h1 class="text-2xl font-semibold text-gray-900">
        Massa
      </h1>
      <p class="text-sm text-gray-600">
        Signed in as {{ auth.user?.email }}
      </p>
      <button
        type="button"
        class="rounded-md bg-gray-200 px-3 py-2 text-sm font-medium text-gray-800 hover:bg-gray-300"
        @click="onLogout"
      >
        Log out
      </button>
    </div>
  </div>
</template>
