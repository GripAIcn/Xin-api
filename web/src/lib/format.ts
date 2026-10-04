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

export function maskKey(key: string): string {
  if (key.length > 12) return key.slice(0, 8) + '****' + key.slice(-4)
  return key
}