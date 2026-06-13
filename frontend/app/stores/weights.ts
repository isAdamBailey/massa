export interface WeightEntry {
  id: string
  weightKg: number
  recordedAt: string
  bmi?: number
  heightUsedCm?: number
  source: string
  createdAt: string
  updatedAt: string
}

interface WeightEntryInput {
  weightKg: number
  recordedAt: string
}

export const useWeightsStore = defineStore('weights', () => {
  const entries = ref<WeightEntry[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * fetchEntries loads the current user's weight entries, optionally
   * filtered to a recorded_at date range.
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
      entries.value = await apiFetch<WeightEntry[]>(`/api/weights${qs ? `?${qs}` : ''}`)
    } catch {
      error.value = 'Failed to load weight entries.'
    } finally {
      loading.value = false
    }
  }

  /** createEntry records a new weight entry. */
  async function createEntry(input: WeightEntryInput) {
    error.value = null
    try {
      const entry = await apiFetch<WeightEntry>('/api/weights', { method: 'POST', body: input })
      entries.value = [...entries.value, entry].sort((a, b) => a.recordedAt.localeCompare(b.recordedAt))
      return entry
    } catch {
      error.value = 'Failed to save weight entry.'
      return null
    }
  }

  /** updateEntry changes an existing weight entry. */
  async function updateEntry(id: string, input: WeightEntryInput) {
    error.value = null
    try {
      const entry = await apiFetch<WeightEntry>(`/api/weights/${id}`, { method: 'PATCH', body: input })
      entries.value = entries.value
        .map(e => e.id === id ? entry : e)
        .sort((a, b) => a.recordedAt.localeCompare(b.recordedAt))
      return entry
    } catch {
      error.value = 'Failed to update weight entry.'
      return null
    }
  }

  /** deleteEntry removes a weight entry. */
  async function deleteEntry(id: string) {
    error.value = null
    try {
      await apiFetch(`/api/weights/${id}`, { method: 'DELETE' })
      entries.value = entries.value.filter(e => e.id !== id)
      return true
    } catch {
      error.value = 'Failed to delete weight entry.'
      return false
    }
  }

  return { entries, loading, error, fetchEntries, createEntry, updateEntry, deleteEntry }
})
