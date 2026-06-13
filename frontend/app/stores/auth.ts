interface User {
  id: string
  email: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)

  const isAuthenticated = computed(() => user.value !== null)

  /** fetchMe loads the current user from the backend, if any session exists. */
  async function fetchMe() {
    try {
      user.value = await apiFetch<User>('/api/me')
    } catch {
      user.value = null
    } finally {
      initialized.value = true
    }
  }

  /** requestMagicLink emails a sign-in link to the given address. */
  async function requestMagicLink(email: string) {
    await apiFetch('/api/auth/magic-link', { method: 'POST', body: { email } })
  }

  /** verifyMagicLink exchanges a magic-link token for a session. */
  async function verifyMagicLink(token: string) {
    await apiFetch('/api/auth/verify', { method: 'POST', body: { token } })
    await fetchMe()
  }

  /** logout ends the current session. */
  async function logout() {
    await apiFetch('/api/auth/logout', { method: 'POST' })
    user.value = null
  }

  return { user, initialized, isAuthenticated, fetchMe, requestMagicLink, verifyMagicLink, logout }
})
