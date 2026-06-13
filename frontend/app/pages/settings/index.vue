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
  <div class="min-h-screen bg-carbon px-4 py-6 text-mist sm:px-6 sm:py-10">
    <div class="mx-auto flex max-w-xl flex-col gap-4">
      <header class="flex items-center justify-between">
        <h1 class="text-headline font-sans">
          Settings
        </h1>
        <NuxtLink
          to="/"
          class="rounded-sm bg-graphite px-4 py-2 text-label text-mist transition-colors duration-150 hover:bg-graphite-hover"
        >
          Back
        </NuxtLink>
      </header>

      <section class="space-y-3 rounded-md bg-slate p-5">
        <h2 class="text-title font-sans">
          Units &amp; height
        </h2>

        <form
          class="space-y-3"
          @submit.prevent="onSaveSettings"
        >
          <div>
            <label
              for="units-preference"
              class="block text-label text-fog"
            >Units</label>
            <select
              id="units-preference"
              v-model="unitsPreference"
              class="mt-1 w-full rounded-sm bg-graphite px-3 py-2 text-body text-mist"
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
              class="block text-label text-fog"
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
              class="mt-1 w-full rounded-sm bg-graphite px-3 py-2 text-body text-mist placeholder:text-[oklch(0.70_0.01_170)]"
            >
          </div>

          <p
            v-if="saveError"
            class="text-body text-ember"
          >
            {{ saveError }}
          </p>
          <p
            v-else-if="saved"
            class="text-body text-fog"
          >
            Settings saved.
          </p>

          <button
            type="submit"
            :disabled="saving"
            class="rounded-sm bg-verdigris px-4 py-2 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover disabled:opacity-50"
          >
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
        </form>
      </section>

      <section class="space-y-3 rounded-md bg-slate p-5">
        <h2 class="text-title font-sans">
          Google Health
        </h2>

        <p
          v-if="google.loading"
          class="text-body text-fog"
        >
          Loading…
        </p>

        <template v-else>
          <p
            class="text-body"
            :class="google.status.connected ? 'text-mist' : 'text-fog'"
          >
            {{ google.status.connected ? 'Connected' : 'Not connected' }}
          </p>

          <dl
            v-if="google.status.connected"
            class="space-y-1"
          >
            <div class="flex justify-between text-body">
              <dt class="text-fog">
                Last full backfill
              </dt>
              <dd class="text-mist">
                {{ formatDate(google.status.lastFullBackfillAt) }}
              </dd>
            </div>
            <div class="flex justify-between text-body">
              <dt class="text-fog">
                Last sync
              </dt>
              <dd class="text-mist">
                {{ formatDate(google.status.lastIncrementalSyncAt) }}
              </dd>
            </div>
          </dl>

          <p
            v-if="google.error"
            class="text-body text-ember"
          >
            {{ google.error }}
          </p>

          <div class="flex gap-2 pt-2">
            <button
              v-if="!google.status.connected"
              type="button"
              class="rounded-sm bg-verdigris px-4 py-2 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover"
              @click="google.connect"
            >
              Connect Google Health
            </button>
            <template v-else>
              <button
                type="button"
                :disabled="google.syncing"
                class="rounded-sm bg-verdigris px-4 py-2 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover disabled:opacity-50"
                @click="google.sync"
              >
                {{ google.syncing ? 'Syncing…' : 'Sync now' }}
              </button>
              <button
                type="button"
                class="rounded-sm bg-graphite px-4 py-2 text-label text-mist transition-colors duration-150 hover:bg-graphite-hover"
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
