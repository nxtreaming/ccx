export interface ConversationTurnPreviewOptions {
  width: number
  font: string
  maxLines?: number
  measureText?: (text: string, font: string) => number
}

let conversationTurnCanvas: HTMLCanvasElement | null = null

export function getConversationTurnFont(element: HTMLElement): string {
  const textElement = element.querySelector<HTMLElement>('.main-conversation-turn') ?? element
  const style = window.getComputedStyle(textElement)
  if (style.font) return style.font
  return [
    style.fontStyle,
    style.fontVariant,
    style.fontWeight,
    style.fontStretch,
    style.fontSize,
    style.fontFamily,
  ]
    .filter(Boolean)
    .join(' ')
}

export function buildConversationTurnPreview(text: string, options: ConversationTurnPreviewOptions): string {
  const normalized = text.replace(/\r/g, '').trim()
  if (normalized === '') return ''

  const width = Number.isFinite(options.width) ? options.width : 0
  const maxLines = Math.max(1, Math.floor(options.maxLines ?? 5))
  if (width <= 0) return normalized

  const measureText = options.measureText ?? defaultMeasureText
  const lines: string[] = []
  let truncated = false

  for (const paragraph of normalized.split('\n')) {
    if (lines.length >= maxLines) {
      truncated = true
      break
    }

    const wrapped = wrapConversationParagraph(paragraph, width, options.font, measureText)
    for (const line of wrapped) {
      if (lines.length >= maxLines) {
        truncated = true
        break
      }
      lines.push(line)
    }

    if (truncated) break
  }

  if (lines.length === 0) return ''
  if (truncated) {
    lines[maxLines - 1] = appendEllipsis(lines[maxLines - 1], width, options.font, measureText)
  }
  return lines.join('\n')
}

function defaultMeasureText(text: string, font: string): number {
  if (typeof document === 'undefined') return Math.max(0, text.length * 8)
  conversationTurnCanvas ??= document.createElement('canvas')
  const context = conversationTurnCanvas.getContext('2d')
  if (!context) return Math.max(0, text.length * 8)
  context.font = font
  return context.measureText(text).width
}

function wrapConversationParagraph(
  paragraph: string,
  width: number,
  font: string,
  measureText: (text: string, font: string) => number,
): string[] {
  if (paragraph === '') return ['']

  const tokens = paragraph.match(/\s+|\S+/g) ?? []
  if (tokens.length === 0) return ['']

  const lines: string[] = []
  let current = ''

  const flushCurrent = () => {
    if (current === '') return
    lines.push(current.trimEnd())
    current = ''
  }

  for (const token of tokens) {
    if (/^\s+$/.test(token)) {
      if (current === '') continue
      const candidate = current + token
      if (measureText(candidate, font) <= width) {
        current = candidate
      } else {
        flushCurrent()
      }
      continue
    }

    const candidate = current + token
    if (measureText(candidate, font) <= width) {
      current = candidate
      continue
    }

    flushCurrent()

    if (measureText(token, font) <= width) {
      current = token
      continue
    }

    let segment = ''
    for (const char of splitConversationGraphemes(token)) {
      const next = segment + char
      if (segment === '' || measureText(next, font) <= width) {
        segment = next
      } else {
        lines.push(segment)
        segment = char
      }
    }
    current = segment
  }

  flushCurrent()
  return lines.length > 0 ? lines : ['']
}

function appendEllipsis(
  line: string,
  width: number,
  font: string,
  measureText: (text: string, font: string) => number,
): string {
  const ellipsis = '…'
  const trimmed = line.trimEnd()
  if (measureText(`${trimmed}${ellipsis}`, font) <= width) return `${trimmed}${ellipsis}`

  let candidate = ''
  for (const char of splitConversationGraphemes(trimmed)) {
    const next = candidate + char
    if (measureText(`${next}${ellipsis}`, font) <= width) {
      candidate = next
    } else {
      break
    }
  }

  return candidate ? `${candidate}${ellipsis}` : ellipsis
}

function splitConversationGraphemes(text: string): string[] {
  return Array.from(text)
}
