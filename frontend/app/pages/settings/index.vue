<script setup lang="ts">
const google = useGoogleHealthStore()
const route = useRoute()
const router = useRouter()

onMounted(async () => {
  await google.fetchStatus()

  if (route.query.google === 'connected') {
    await router.replace({ query: {} })
  }
})

function formatDate(value?: string) {
  if (!value) {
    return 'Never'
  }
  return new Date(value).toLocaleString()
}
</script>

<template>
  <div class="min-h-screen bg-gray-50 px-4 py-8">
    <div class="mx-auto max-w-md space-y-6">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-semibold text-gray-900">
          Settings
        </h1>
        <NuxtLink
          to="/"
          class="text-sm font-medium text-blue-600 hover:underline"
        >
          Back
        </NuxtLink>
      </div>

      <section class="space-y-3 rounded-md border border-gray-200 bg-white p-4">
        <h2 class="text-lg font-medium text-gray-900">
          Google Health
        </h2>

        <p
          v-if="google.loading"
          class="text-sm text-gray-600"
        >
          Loading…
        </p>

        <template v-else>
          <p
            v-if="google.status.connected"
            class="text-sm font-medium text-green-700"
          >
            Connected
          </p>
          <p
            v-else
            class="text-sm text-gray-600"
          >
            Not connected
          </p>

          <dl
            v-if="google.status.connected"
            class="space-y-1 text-sm text-gray-600"
          >
            <div class="flex justify-between">
              <dt>Last full backfill</dt>
              <dd>{{ formatDate(google.status.lastFullBackfillAt) }}</dd>
            </div>
            <div class="flex justify-between">
              <dt>Last sync</dt>
              <dd>{{ formatDate(google.status.lastIncrementalSyncAt) }}</dd>
            </div>
          </dl>

          <p
            v-if="google.error"
            class="text-sm text-red-600"
          >
            {{ google.error }}
          </p>

          <div class="flex gap-2 pt-2">
            <button
              v-if="!google.status.connected"
              type="button"
              class="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
              @click="google.connect"
            >
              Connect Google Health
            </button>
            <template v-else>
              <button
                type="button"
                :disabled="google.syncing"
                class="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                @click="google.sync"
              >
                {{ google.syncing ? 'Syncing…' : 'Sync now' }}
              </button>
              <button
                type="button"
                class="rounded-md bg-gray-200 px-3 py-2 text-sm font-medium text-gray-800 hover:bg-gray-300"
                @click="google.disconnect"
              >
                Disconnect
              </button>
            </template>
          </div>
        </template>
      </section>
    </div>
  </div>
</template>
