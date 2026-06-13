<script setup lang="ts">
const auth = useAuthStore()

const email = ref('')
const status = ref<'idle' | 'sending' | 'sent' | 'error'>('idle')

async function onSubmit() {
  status.value = 'sending'
  try {
    await auth.requestMagicLink(email.value.trim())
    status.value = 'sent'
  } catch {
    status.value = 'error'
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-sm space-y-6">
      <h1 class="text-center text-2xl font-semibold text-gray-900">
        Sign in to Massa
      </h1>

      <form
        v-if="status !== 'sent'"
        class="space-y-4"
        @submit.prevent="onSubmit"
      >
        <input
          v-model="email"
          type="email"
          required
          placeholder="you@example.com"
          autocomplete="email"
          class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
        >
        <button
          type="submit"
          :disabled="status === 'sending'"
          class="w-full rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {{ status === 'sending' ? 'Sending…' : 'Send sign-in link' }}
        </button>

        <p
          v-if="status === 'error'"
          class="text-sm text-red-600"
        >
          Something went wrong. Please try again.
        </p>
      </form>

      <p
        v-else
        class="text-center text-sm text-gray-600"
      >
        If that email is allowed to sign in, a link has been sent. Check your
        inbox and click the link to continue.
      </p>
    </div>
  </div>
</template>
