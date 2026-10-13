export function formatDate(date?: string): string {
  if (!date) return '-'
  return date.slice(0, 19).replace('T', ' ')
}

export function formatTime(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`
  }
  return `${(ms / 1000).toFixed(2)}s`
}