export const maskApiKey = (key: string): string => {
  if (!key) return ''

  if (key.length <= 5) {
    return '***'
  }

  if (key.length <= 10) {
    return `${key.slice(0, 3)}***${key.slice(-2)}`
  }

  return `${key.slice(0, 8)}***${key.slice(-5)}`
}
