/**
 * Resumes a weight save that was interrupted by an automatic Google Health
 * reconnect. LogCard stashes the entry (via useGooglePendingWeight) and
 * redirects to Google's OAuth consent screen when a sync reports
 * reconnect_required; this plugin runs once the app reloads after that
 * full-page redirect, picks the stash back up, pulls the day's calories,
 * and completes the original save.
 */
export default defineNuxtPlugin(() => {
  const nuxtApp = useNuxtApp()

  nuxtApp.hook('app:mounted', async () => {
    const pending = useGooglePendingWeight()
    const entry = pending.get()
    if (!entry) {
      return
    }

    const google = useGoogleHealthStore()
    const weights = useWeightsStore()

    await google.fetchStatus()
    if (!google.status.connected) {
      // Still not connected (e.g. the consent screen was abandoned) — leave
      // the stash in place so a later reconnect can pick it up.
      return
    }

    await google.sync()
    if (google.reconnectRequired) {
      // The reconnect didn't actually restore a usable connection — leave
      // the stash for the next attempt rather than saving a weight entry we
      // can't also account for in that day's synced energy data.
      return
    }

    // Only clear the stash once the entry is actually saved, so a transient
    // failure here (network blip, expired session) leaves it in place for a
    // retry on the next app load instead of silently losing what the user
    // typed.
    if (await weights.createEntry(entry)) {
      pending.clear()
    }
  })
})
