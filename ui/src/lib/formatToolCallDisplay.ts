function tryParseJson(text: string): unknown | undefined {
  const trimmed = text.trim()
  if (!trimmed) return undefined
  if (trimmed[0] !== '{' && trimmed[0] !== '[' && trimmed[0] !== '"') return undefined
  try {
    return JSON.parse(trimmed)
  } catch {
    return undefined
  }
}

function isMcpContentBlock(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && 'type' in value
}

function isMcpContentArray(value: unknown[]): boolean {
  return value.length > 0 && value.every(isMcpContentBlock)
}

function formatMcpContentBlock(block: Record<string, unknown>): string {
  if (block.type === 'text' && typeof block.text === 'string') {
    return formatToolCallDisplay(block.text)
  }
  if (block.type === 'image') {
    const mime = typeof block.mimeType === 'string' ? block.mimeType : 'image'
    return `[${mime}]`
  }
  return JSON.stringify(normalizeJsonStrings(block), null, 2)
}

function normalizeJsonStrings(value: unknown): unknown {
  if (typeof value === 'string') {
    const parsed = tryParseJson(value)
    if (parsed !== undefined) return normalizeJsonStrings(parsed)
    return value
  }
  if (Array.isArray(value)) {
    return value.map(normalizeJsonStrings)
  }
  if (value && typeof value === 'object') {
    const obj = value as Record<string, unknown>
    const next: Record<string, unknown> = {}
    for (const [key, val] of Object.entries(obj)) {
      next[key] = normalizeJsonStrings(val)
    }
    return next
  }
  return value
}

export function formatToolCallDisplay(value: unknown): string {
  if (value === undefined) return ''
  if (value === null) return 'null'

  if (typeof value === 'string') {
    const parsed = tryParseJson(value)
    if (parsed !== undefined) return formatToolCallDisplay(parsed)
    return value
  }

  if (Array.isArray(value)) {
    if (isMcpContentArray(value)) {
      return value
        .map((block) => formatMcpContentBlock(block as Record<string, unknown>))
        .join('\n\n')
    }
    return JSON.stringify(normalizeJsonStrings(value), null, 2)
  }

  if (typeof value === 'object') {
    const obj = value as Record<string, unknown>
    if (Array.isArray(obj.content)) {
      if (isMcpContentArray(obj.content)) {
        const body = obj.content
          .map((block) => formatMcpContentBlock(block as Record<string, unknown>))
          .join('\n\n')
        if (obj.isError) return `[error]\n${body}`
        return body
      }
      return JSON.stringify(normalizeJsonStrings(obj.content), null, 2)
    }
    return JSON.stringify(normalizeJsonStrings(obj), null, 2)
  }

  return String(value)
}
