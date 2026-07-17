function pad(n: number): string {
  return String(n).padStart(2, '0')
}

/**
 * Formats a Date for an HTML datetime-local input (YYYY-MM-DDTHH:mm) in local time.
 */
export function toDateTimeLocalInput(value: Date | string = new Date()): string {
  const date = typeof value === 'string' ? new Date(value) : value
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

/**
 * Formats a Date for an HTML date input (YYYY-MM-DD) in local time.
 */
export function toDateLocalInput(value: Date | string = new Date()): string {
  const date = typeof value === 'string' ? new Date(value) : value
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}
