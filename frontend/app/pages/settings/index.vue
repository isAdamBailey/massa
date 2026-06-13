<script setup lang="ts">
import type { UnitsPreference } from '~/stores/settings'

const google = useGoogleHealthStore()
const settings = useSettingsStore()
const { cmToIn, inToCm } = useBmi()
const route = useRoute()
const router = useRouter()

const heightInput = ref('')
const unitsPreference = ref<UnitsPreference>('metric')
const saving = ref(false)
const saveError = ref<string | null>(null)
const saved = ref(false)

onMounted(async () => {
  await Promise.all([google.fetchStatus(), settings.fetchSettings()])

  if (route.query.google === 'connected') {
    await router.replace({ query: {} })
  }

  unitsPreference.value = settings.settings.unitsPreference
  if (settings.settings.manualHeightCm) {
    const height = unitsPreference.value === 'imperial'
      ? cmToIn(settings.settings.manualHeightCm)
      : settings.settings.manualHeightCm
    heightInput.value = height.toFixed(1)
  }
})

function formatDate(value?: string) {
  if (!value) {
    return 'Never'
  }
  return new Date(value).toLocaleString()
}

async function onSaveSettings() {
  saveError.value = null
  saved.value = false

  let manualHeightCm: number | undefined
  if (heightInput.value) {
    const value = Number(heightInput.value)
    if (!(value > 0)) {
      saveError.value = 'Enter a valid height.'
      return
    }
    manualHeightCm = unitsPreference.value === 'imperial' ? inToCm(value) : value
  }

  saving.value = true
  try {
    const ok = await settings.updateSettings({ manualHeightCm, unitsPreference: unitsPreference.value })
    saved.value = ok
    if (!ok) {
      saveError.value = settings.error
    }
  } finally {
    saving.value = false
  }
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
          Units &amp; height
        </h2>

        <form
          class="space-y-3"
          @submit.prevent="onSaveSettings"
        >
          <div>
            <label
              for="units-preference"
              class="block text-xs text-gray-500"
            >Units</label>
            <select
              id="units-preference"
              v-model="unitsPreference"
              class="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
            >
              <option value="metric">
                Metric (kg, cm)
              </option>
              <option value="imperial">
                Imperial (lb, in)
              </option>
            </select>
          </div>

          <div>
            <label
              for="manual-height"
              class="block text-xs text-gray-500"
            >
              Height override ({{ unitsPreference === 'imperial' ? 'in' : 'cm' }})
            </label>
            <input
              id="manual-height"
              v-model="heightInput"
              type="number"
              step="0.1"
              min="0"
              placeholder="Used when no synced height is available"
              class="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
            >
          </div>

          <p
            v-if="saveError"
            class="text-sm text-red-600"
          >
            {{ saveError }}
          </p>
          <p
            v-else-if="saved"
            class="text-sm text-green-700"
          >
            Settings saved.
          </p>

          <button
            type="submit"
            :disabled="saving"
            class="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
        </form>
      </section>

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
