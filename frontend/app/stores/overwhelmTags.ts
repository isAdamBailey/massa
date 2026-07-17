export interface OverwhelmTag {
  id: string
  name: string
}

export const useOverwhelmTagsStore = defineStore('overwhelmTags', () => {
  const tags = ref<OverwhelmTag[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  /** fetchTags loads the current user's active overwhelm tags. */
  async function fetchTags() {
    loading.value = true
    error.value = null
    try {
      tags.value = await apiFetch<OverwhelmTag[]>('/api/overwhelm/tags')
    } catch {
      error.value = 'Failed to load overwhelm tags.'
    } finally {
      loading.value = false
    }
  }

  /**
   * createTag adds a new tag, or unarchives a previously archived tag with
   * the same name.
   */
  async function createTag(name: string) {
    error.value = null
    try {
      const tag = await apiFetch<OverwhelmTag>('/api/overwhelm/tags', { method: 'POST', body: { name } })
      tags.value = [...tags.value, tag].sort((a, b) => a.name.localeCompare(b.name))
      return tag
    } catch {
      error.value = 'Failed to create tag.'
      return null
    }
  }

  /** renameTag renames an existing tag. */
  async function renameTag(id: string, name: string) {
    error.value = null
    try {
      const tag = await apiFetch<OverwhelmTag>(`/api/overwhelm/tags/${id}`, { method: 'PATCH', body: { name } })
      tags.value = tags.value
        .map(t => t.id === id ? tag : t)
        .sort((a, b) => a.name.localeCompare(b.name))
      return tag
    } catch {
      error.value = 'A tag with that name already exists.'
      return null
    }
  }

  /** archiveTag removes a tag from the picker without deleting its history. */
  async function archiveTag(id: string) {
    error.value = null
    try {
      await apiFetch(`/api/overwhelm/tags/${id}`, { method: 'DELETE' })
      tags.value = tags.value.filter(t => t.id !== id)
      return true
    } catch {
      error.value = 'Failed to remove tag.'
      return false
    }
  }

  return { tags, loading, error, fetchTags, createTag, renameTag, archiveTag }
})
