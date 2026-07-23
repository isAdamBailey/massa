<script setup lang="ts">
import type { OverwhelmTag } from '~/stores/overwhelmTags'
import type { UnitsPreference } from '~/stores/settings'

const google = useGoogleHealthStore()
const settings = useSettingsStore()
const overwhelmTags = useOverwhelmTagsStore()
const { cmToIn, inToCm } = useBmi()
const route = useRoute()
const router = useRouter()

const heightInput = ref('')
const unitsPreference = ref<UnitsPreference>('metric')
const saving = ref(false)
const saveError = ref<string | null>(null)
const saved = ref(false)
const syncEnabledUpdating = ref(false)

async function onToggleSyncEnabled() {
  syncEnabledUpdating.value = true
  try {
    await google.setSyncEnabled(!google.status.syncEnabled)
  } finally {
    syncEnabledUpdating.value = false
  }
}

onMounted(async () => {
  await Promise.all([google.fetchStatus(), settings.fetchSettings(), overwhelmTags.fetchTags()])

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

// --- Overwhelm tag vocabulary ---

const newTagName = ref('')
const editingTagId = ref<string | null>(null)
const editingTagName = ref('')

function startEditTag(tag: OverwhelmTag) {
  editingTagId.value = tag.id
  editingTagName.value = tag.name
}

function cancelEditTag() {
  editingTagId.value = null
  editingTagName.value = ''
}

async function onCreateTag() {
  const name = newTagName.value.trim()
  if (!name) {
    return
  }
  const tag = await overwhelmTags.createTag(name)
  if (tag) {
    newTagName.value = ''
  }
}

async function onRenameTag(id: string) {
  const name = editingTagName.value.trim()
  if (!name) {
    return
  }
  const tag = await overwhelmTags.renameTag(id, name)
  if (tag) {
    cancelEditTag()
  }
}

async function onArchiveTag(id: string) {
  if (editingTagId.value === id) {
    cancelEditTag()
  }
  await overwhelmTags.archiveTag(id)
}
</script>

<template>
  <div class="min-h-screen bg-carbon px-4 py-6 text-mist sm:px-6 sm:py-10">
    <div class="mx-auto flex max-w-3xl flex-col gap-4">
      <AppHeader />

      <h1 class="text-headline font-sans">
        Settings
      </h1>

      <section class="space-y-3 rounded-md bg-slate p-5">
        <h2 class="text-title font-sans">
          Units &amp; height
        </h2>

        <form
          class="space-y-3"
          @submit.prevent="onSaveSettings"
        >
          <div class="flex flex-wrap gap-3">
            <div class="flex-1">
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

            <div class="flex-1">
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
                class="mt-1 w-full rounded-sm bg-graphite px-3 py-2 text-body text-mist placeholder:text-fog/55"
              >
            </div>
          </div>

          <p
            v-if="saveError"
            class="text-body text-ember"
          >
            {{ saveError }}
          </p>
          <p
            v-else-if="saved"
            class="text-body text-mist"
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
          class="text-body text-mist"
        >
          Loading…
        </p>

        <template v-else>
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <p
                class="text-body"
                :class="google.status.connected && google.status.syncEnabled ? 'text-mist' : 'text-fog'"
              >
                {{
                  google.syncing
                    ? 'Syncing…'
                    : google.status.connected
                      ? (google.status.syncEnabled ? 'Syncing with Google Health' : 'Paused')
                      : 'Off'
                }}
              </p>
              <p class="mt-0.5 text-label text-fog">
                {{
                  google.status.connected
                    ? `Last sync ${formatDate(google.status.lastIncrementalSyncAt)}`
                    : 'Turn on to connect and sync'
                }}
              </p>
            </div>

            <button
              type="button"
              role="switch"
              :aria-checked="!!google.status.syncEnabled"
              :disabled="syncEnabledUpdating || google.syncing"
              aria-label="Sync with Google Health"
              class="relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors duration-150 disabled:opacity-50"
              :class="google.status.syncEnabled
                ? 'bg-verdigris'
                : 'bg-graphite ring-1 ring-inset ring-hairline'"
              @click="onToggleSyncEnabled"
            >
              <span
                class="pointer-events-none block h-5 w-5 rounded-full transition-transform duration-150 ease-[cubic-bezier(0.16,1,0.3,1)]"
                :class="google.status.syncEnabled
                  ? 'translate-x-6.5 bg-carbon'
                  : 'translate-x-1 bg-mist'"
              />
            </button>
          </div>

          <p
            v-if="google.error"
            class="text-body text-ember"
          >
            {{ google.error }}
          </p>
        </template>
      </section>

      <section class="space-y-3 rounded-md bg-slate p-5">
        <h2 class="text-title font-sans">
          Overwhelm tags
        </h2>
        <p class="text-body text-mist">
          Keywords you can attach to an overwhelm entry to describe why.
          Removing a tag keeps it on any day it was already logged.
        </p>

        <ul
          v-if="overwhelmTags.tags.length"
          class="space-y-2"
        >
          <li
            v-for="tag in overwhelmTags.tags"
            :key="tag.id"
            class="flex items-center gap-2"
          >
            <input
              v-if="editingTagId === tag.id"
              v-model="editingTagName"
              type="text"
              maxlength="30"
              class="min-w-0 flex-1 rounded-sm bg-graphite px-3 py-2 text-body text-mist"
              @keyup.enter="onRenameTag(tag.id)"
              @keyup.esc="cancelEditTag"
            >
            <span
              v-else
              class="min-w-0 flex-1 text-body text-mist"
            >
              {{ tag.name }}
            </span>

            <template v-if="editingTagId === tag.id">
              <button
                type="button"
                class="shrink-0 rounded-sm bg-verdigris px-3 py-1.5 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover"
                @click="onRenameTag(tag.id)"
              >
                Save
              </button>
              <button
                type="button"
                class="shrink-0 rounded-sm bg-graphite px-3 py-1.5 text-label text-mist transition-colors duration-150 hover:bg-graphite-hover"
                @click="cancelEditTag"
              >
                Cancel
              </button>
            </template>
            <template v-else>
              <button
                type="button"
                class="shrink-0 rounded-sm bg-graphite px-3 py-1.5 text-label text-mist transition-colors duration-150 hover:bg-graphite-hover"
                @click="startEditTag(tag)"
              >
                Rename
              </button>
              <button
                type="button"
                class="shrink-0 rounded-sm px-3 py-1.5 text-label text-ember transition-colors duration-150 hover:bg-graphite"
                @click="onArchiveTag(tag.id)"
              >
                Remove
              </button>
            </template>
          </li>
        </ul>
        <p
          v-else
          class="text-body text-mist"
        >
          No tags yet.
        </p>

        <form
          class="flex gap-2"
          @submit.prevent="onCreateTag"
        >
          <input
            v-model="newTagName"
            type="text"
            placeholder="New tag"
            maxlength="30"
            class="min-w-0 flex-1 rounded-sm bg-graphite px-3 py-2 text-body text-mist placeholder:text-fog/55"
          >
          <button
            type="submit"
            class="shrink-0 rounded-sm bg-verdigris px-4 py-2 text-label text-carbon transition-colors duration-150 hover:bg-verdigris-hover"
          >
            Add
          </button>
        </form>

        <p
          v-if="overwhelmTags.error"
          class="text-body text-ember"
        >
          {{ overwhelmTags.error }}
        </p>
      </section>
    </div>
  </div>
</template>
