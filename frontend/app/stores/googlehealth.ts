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
  // Set when a sync fails because Google's test-app refresh token expired
  // (it lasts 7 days until the app is published) or was revoked, so the UI
  // can show a "reconnect" prompt instead of a plain disconnected state.
  const reconnectRequired = ref(false)

  /** fetchStatus loads the current user's Google Health connection status. */
  async function fetchStatus() {
    loading.value = true
    error.value = null
    try {
      status.value = await apiFetch<GoogleHealthStatus>('/api/google/status')
      if (status.value.connected) {
        reconnectRequired.value = false
      }
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
      reconnectRequired.value = false
    } catch {
      error.value = 'Failed to disconnect Google Health. Please try again.'
    }
  }

  /**
   * sync re-runs the Google Health backfill for the current user. Failures
   * are swallowed into `error`/`reconnectRequired` rather than thrown, since
   * sync is often triggered as a best-effort side effect (e.g. after saving
   * a weight entry) where a failure shouldn't interrupt the caller.
   */
  async function sync() {
    syncing.value = true
    error.value = null
    try {
      await apiFetch('/api/google/sync', { method: 'POST' })
      await fetchStatus()
    } catch (err) {
      // A 409 with "reconnect_required" means the stored Google credentials
      // have expired or been revoked. Flip to disconnected so the UI offers a
      // reconnect button instead of a dead "Sync now".
      const body = (err as { data?: { error?: string } })?.data
      if (body?.error === 'reconnect_required') {
        status.value = { connected: false }
        reconnectRequired.value = true
        error.value = 'Your Google Health connection has expired. Please reconnect.'
      } else {
        error.value = 'Sync failed. Please try again.'
      }
    } finally {
      syncing.value = false
    }
  }

  return { status, loading, syncing, error, reconnectRequired, fetchStatus, connect, disconnect, sync }
})
