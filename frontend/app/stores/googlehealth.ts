/** apiErrorCode extracts the backend's `{"error": "..."}` code from an apiFetch rejection, if present. */
function apiErrorCode(err: unknown): string | undefined {
  return (err as { data?: { error?: string } })?.data?.error
}

interface GoogleHealthStatus {
  connected: boolean
  healthUserId?: string
  syncEnabled?: boolean
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

  /** connect redirects the browser to Google's OAuth consent screen. Returns false if the redirect could not be started. */
  async function connect() {
    error.value = null
    try {
      const { url } = await apiFetch<{ url: string }>('/api/google/auth-url')
      window.location.href = url
      return true
    } catch {
      error.value = 'Failed to start Google connection. Please try again.'
      return false
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
      // have expired or been revoked. Flip to disconnected so the UI can
      // prompt a reconnect instead of leaving a dead connection.
      if (apiErrorCode(err) === 'reconnect_required') {
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

  /**
   * setSyncEnabled pauses or resumes syncing without discarding the stored
   * Google connection. Enabling with no (or expired) credentials starts a
   * fresh OAuth connect; enabling with a live connection runs a sync.
   */
  async function setSyncEnabled(enabled: boolean) {
    error.value = null
    try {
      await apiFetch('/api/google/sync-enabled', { method: 'POST', body: { enabled } })
      await fetchStatus()
      if (enabled) {
        await sync()
        if (reconnectRequired.value) {
          await connect()
        }
      }
    } catch (err) {
      if (enabled && apiErrorCode(err) === 'reconnect_required') {
        await connect()
        return
      }
      error.value = 'Failed to update Google Health sync setting. Please try again.'
    }
  }

  return { status, loading, syncing, error, reconnectRequired, fetchStatus, connect, sync, setSyncEnabled }
})
