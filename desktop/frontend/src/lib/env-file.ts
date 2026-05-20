export type EnvEntry =
  | { type: 'empty'; raw: string }
  | { type: 'comment'; raw: string }
  | { type: 'pair'; raw: string; key: string; value: string; exportPrefix: boolean; quote: 'none' | 'single' | 'double'; inlineComment: string; prefix: string; separator: string }
  | { type: 'raw'; raw: string }

const pairPattern = /^(\s*)(export\s+)?([A-Za-z_][A-Za-z0-9_]*)(\s*=\s*)(.*)$/

export function detectEnvNewline(content: string) {
  return content.includes('\r\n') ? '\r\n' : '\n'
}

export function parseEnvFile(content: string): EnvEntry[] {
  const lines = content.replace(/\r\n/g, '\n').split('\n')
  if (lines.length > 0 && lines[lines.length - 1] === '') {
    lines.pop()
  }

  return lines.map((line) => {
    if (line.trim() === '') return { type: 'empty', raw: line }
    if (line.trimStart().startsWith('#')) return { type: 'comment', raw: line }

    const match = line.match(pairPattern)
    if (!match) return { type: 'raw', raw: line }

    const { valuePart, inlineComment } = splitInlineComment(match[5] ?? '')
    return {
      type: 'pair',
      raw: line,
      prefix: match[1] ?? '',
      key: match[3],
      separator: match[4] ?? '=',
      value: parseEnvValue(valuePart),
      exportPrefix: Boolean(match[2]),
      quote: detectQuote(valuePart),
      inlineComment,
    }
  })
}

export function getEnvFieldValue(entries: EnvEntry[], key: string, fallback = '') {
  for (let index = entries.length - 1; index >= 0; index--) {
    const entry = entries[index]
    if (entry.type === 'pair' && entry.key === key) return entry.value
  }
  return fallback
}

export function serializeEnvFile(
  entries: EnvEntry[],
  values: Record<string, string>,
  supportedKeys: string[],
  newline = '\n',
) {
  const remainingKeys = new Set(supportedKeys)
  const lines = entries.map((entry) => {
    if (entry.type !== 'pair' || !(entry.key in values)) return entry.raw

    remainingKeys.delete(entry.key)
    const exportPrefix = entry.exportPrefix ? 'export ' : ''
    return `${entry.prefix}${exportPrefix}${entry.key}${entry.separator}${formatEnvValue(values[entry.key] ?? '', entry.quote)}${entry.inlineComment}`
  })

  const additions = supportedKeys
    .filter((key) => remainingKeys.has(key))
    .map((key) => `${key}=${formatEnvValue(values[key] ?? '')}`)

  if (additions.length > 0) {
    if (lines.length > 0 && lines[lines.length - 1].trim() !== '') {
      lines.push('')
    }
    lines.push('# Managed by CCX Desktop')
    lines.push(...additions)
  }

  return lines.join(newline).replace(new RegExp(`${escapeRegExp(newline)}*$`), '') + newline
}

function splitInlineComment(rawValue: string) {
  let quote: 'single' | 'double' | null = null
  let escaped = false

  for (let index = 0; index < rawValue.length; index++) {
    const char = rawValue[index]
    if (escaped) {
      escaped = false
      continue
    }
    if (quote === 'double' && char === '\\') {
      escaped = true
      continue
    }
    if (!quote && char === '"') {
      quote = 'double'
      continue
    }
    if (!quote && char === "'") {
      quote = 'single'
      continue
    }
    if (quote === 'double' && char === '"') {
      quote = null
      continue
    }
    if (quote === 'single' && char === "'") {
      quote = null
      continue
    }
    if (!quote && char === '#' && index > 0 && /\s/.test(rawValue[index - 1])) {
      let commentStart = index
      while (commentStart > 0 && /\s/.test(rawValue[commentStart - 1])) {
        commentStart--
      }
      return {
        valuePart: rawValue.slice(0, commentStart),
        inlineComment: rawValue.slice(commentStart),
      }
    }
  }

  return { valuePart: rawValue, inlineComment: '' }
}

function parseEnvValue(raw: string) {
  const trimmed = raw.trim()
  if (trimmed.startsWith('"') && trimmed.endsWith('"')) {
    return trimmed.slice(1, -1).replace(/\\n/g, '\n').replace(/\\"/g, '"').replace(/\\\\/g, '\\')
  }
  if (trimmed.startsWith("'") && trimmed.endsWith("'")) {
    return trimmed.slice(1, -1)
  }
  return trimmed
}

function detectQuote(raw: string): 'none' | 'single' | 'double' {
  const trimmed = raw.trim()
  if (trimmed.startsWith('"') && trimmed.endsWith('"')) return 'double'
  if (trimmed.startsWith("'") && trimmed.endsWith("'")) return 'single'
  return 'none'
}

function formatEnvValue(value: string, preferredQuote: 'none' | 'single' | 'double' = 'none') {
  if (preferredQuote === 'single' && !value.includes("'")) return `'${value}'`
  if (preferredQuote === 'double') return quoteDouble(value)
  if (value === '') return ''
  if (/^[A-Za-z0-9_./:@*-]+$/.test(value)) return value
  return quoteDouble(value)
}

function quoteDouble(value: string) {
  return `"${value.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\n/g, '\\n')}"`
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
