export interface OverwhelmEntryTag {
  id: string
  name: string
}

export interface OverwhelmEntry {
  day: string
  overwhelmLevel: number
  tags: OverwhelmEntryTag[]
}

interface OverwhelmEntryInput {
  day: string
  overwhelmLevel: number
  tagIds?: string[]
}

export const useOverwhelmStore = defineStore('overwhelm', () => {
  const entries = ref<OverwhelmEntry[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * fetchEntries loads the current user's overwhelm entries, optionally
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
      entries.value = await apiFetch<OverwhelmEntry[]>(`/api/overwhelm${qs ? `?${qs}` : ''}`)
    } catch {
      error.value = 'Failed to load overwhelm entries.'
    } finally {
      loading.value = false
    }
  }

  /**
   * saveEntry records the overwhelm level for a day, replacing any existing
   * entry for that day (the API upserts by day).
   */
  async function saveEntry(input: OverwhelmEntryInput) {
    error.value = null
    try {
      const entry = await apiFetch<OverwhelmEntry>('/api/overwhelm', { method: 'PUT', body: input })
      entries.value = [...entries.value.filter(e => e.day !== entry.day), entry]
        .sort((a, b) => a.day.localeCompare(b.day))
      return entry
    } catch {
      error.value = 'Failed to save overwhelm entry.'
      return null
    }
  }

  return { entries, loading, error, fetchEntries, saveEntry }
})
