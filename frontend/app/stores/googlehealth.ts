interface GoogleHealthStatus {
  connected: boolean
  healthUserId?: string
  lastFullBackfillAt?: string
  lastIncrementalSyncAt?: string
}

export const useGoogleHealthStore = defineStore('googlehealth', () => {
  const status = ref<GoogleHealthStatus>({ connected: false })
  const loading = ref(false)
  const syncing = ref(false)
  const error = ref<string | null>(null)

  /** fetchStatus loads the current user's Google Health connection status. */
  async function fetchStatus() {
    loading.value = true
    error.value = null
    try {
      status.value = await apiFetch<GoogleHealthStatus>('/api/google/status')
    } catch {
      error.value = 'Failed to load Google Health connection status.'
    } finally {
      loading.value = false
    }
  }

  /** connect redirects the browser to Google's OAuth consent screen. */
  async function connect() {
    error.value = null
    try {
      const { url } = await apiFetch<{ url: string }>('/api/google/auth-url')
      window.location.href = url
    } catch {
      error.value = 'Failed to start Google connection. Please try again.'
    }
  }

  /** disconnect removes the stored Google Health credentials. */
  async function disconnect() {
    error.value = null
    try {
      await apiFetch('/api/google/disconnect', { method: 'POST' })
      status.value = { connected: false }
    } catch {
      error.value = 'Failed to disconnect Google Health. Please try again.'
    }
  }

  /** sync re-runs the Google Health backfill for the current user. */
  async function sync() {
    syncing.value = true
    error.value = null
    try {
      await apiFetch('/api/google/sync', { method: 'POST' })
      await fetchStatus()
    } catch {
      error.value = 'Sync failed. Please try again.'
    } finally {
      syncing.value = false
    }
  }

  return { status, loading, syncing, error, fetchStatus, connect, disconnect, sync }
})
