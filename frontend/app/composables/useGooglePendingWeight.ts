const STORAGE_KEY = 'massa_pending_weight'

interface PendingWeight {
  weightKg: number
  recordedAt: string
}

/**
 * useGooglePendingWeight stashes a weight entry the user tried to save right
 * as Google Health needed reconnecting, so it survives the full-page OAuth
 * redirect and can be resumed once the connection is restored (see
 * app/plugins/google-resume.client.ts).
 */
export function useGooglePendingWeight() {
  function set(entry: PendingWeight) {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(entry))
  }

  function get(): PendingWeight | null {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return null
    }
    try {
      return JSON.parse(raw) as PendingWeight
    } catch {
      return null
    }
  }

  function clear() {
    sessionStorage.removeItem(STORAGE_KEY)
  }

  return { set, get, clear }
}
