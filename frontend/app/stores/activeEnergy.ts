export interface ActiveEnergyEntry {
  day: string
  activeEnergyKcal: number
}

export const useActiveEnergyStore = defineStore('activeEnergy', () => {
  const entries = ref<ActiveEnergyEntry[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * fetchEntries loads the current user's active energy entries, optionally
   * filtered to a day range.
   */
  async function fetchEntries(range: { from?: string, to?: string } = {}) {
    loading.value = true
    error.value = null
    try {
      const query = new URLSearchParams()
      if (range.from) {
        query.set('from', range.from)
      }
      if (range.to) {
        query.set('to', range.to)
      }
      const qs = query.toString()
      entries.value = await apiFetch<ActiveEnergyEntry[]>(`/api/active-energy${qs ? `?${qs}` : ''}`)
    } catch {
      error.value = 'Failed to load active energy entries.'
    } finally {
      loading.value = false
    }
  }

  return { entries, loading, error, fetchEntries }
})
