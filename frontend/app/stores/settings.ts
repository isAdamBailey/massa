export type UnitsPreference = 'metric' | 'imperial'

export interface Settings {
  manualHeightCm?: number
  unitsPreference: UnitsPreference
}

export const useSettingsStore = defineStore('settings', () => {
  const settings = ref<Settings>({ unitsPreference: 'metric' })
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<string | null>(null)

  /** fetchSettings loads the current user's settings. */
  async function fetchSettings() {
    loading.value = true
    error.value = null
    try {
      settings.value = await apiFetch<Settings>('/api/settings')
    } catch {
      error.value = 'Failed to load settings.'
    } finally {
      loading.value = false
    }
  }

  /** updateSettings saves the user's manual height override and units preference. */
  async function updateSettings(input: Settings) {
    saving.value = true
    error.value = null
    try {
      settings.value = await apiFetch<Settings>('/api/settings', { method: 'PUT', body: input })
      return true
    } catch {
      error.value = 'Failed to save settings.'
      return false
    } finally {
      saving.value = false
    }
  }

  return { settings, loading, saving, error, fetchSettings, updateSettings }
})
