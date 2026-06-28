const MAX_SUMMARY_LENGTH = 120

function truncate(text: string, max = MAX_SUMMARY_LENGTH): string {
  const single = text.replace(/\s+/g, ' ').trim()
  if (single.length <= max) return single
  return `${single.slice(0, max - 1)}…`
}

function strArg(args: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const val = args[key]
    if (typeof val === 'string' && val.trim()) return val.trim()
  }
  return undefined
}

function basename(path: string): string {
  const normalized = path.replace(/\\/g, '/')
  return normalized.split('/').pop() || path
}

function normalizeToolName(toolName: string | undefined): string {
  const raw = toolName?.split(':').pop() ?? toolName ?? ''
  return raw.split('__').pop()?.toLowerCase() ?? ''
}

function firstStringArg(args: Record<string, unknown>): string | undefined {
  for (const val of Object.values(args)) {
    if (typeof val === 'string' && val.trim()) return val.trim()
  }
  return undefined
}

export function formatToolCallSummary(
  toolName: string | undefined,
  toolArgs: Record<string, unknown> | undefined,
): string | undefined {
  if (!toolArgs || Object.keys(toolArgs).length === 0) return undefined

  const args =
    toolArgs.arguments && typeof toolArgs.arguments === 'object' && !Array.isArray(toolArgs.arguments)
      ? (toolArgs.arguments as Record<string, unknown>)
      : toolArgs

  const name = normalizeToolName(toolName)
  const command = strArg(args, 'command', 'cmd')
  const path = strArg(args, 'path', 'file_path', 'target_file', 'target_directory', 'directory')
  const pattern = strArg(args, 'pattern', 'glob_pattern', 'glob', 'regex')
  const query = strArg(args, 'query', 'search_term', 'searchTerm', 'prompt', 'description')
  const url = strArg(args, 'url', 'uri')
  const oldString = strArg(args, 'old_string', 'oldString')
  const newString = strArg(args, 'new_string', 'newString')

  if (command && (name.includes('bash') || name.includes('shell') || name === 'run_terminal_cmd')) {
    return truncate(command)
  }

  if (url && (name.includes('fetch') || name.includes('web') || name === 'webfetch')) {
    return truncate(url)
  }

  if (query && (name.includes('search') || name.includes('task') || name === 'semanticsearch')) {
    return truncate(query)
  }

  if (pattern && (name.includes('grep') || name.includes('glob') || name.includes('glob_file_search'))) {
    const scope = path ? ` in ${basename(path)}` : ''
    return truncate(`${pattern}${scope}`)
  }

  if (
    path &&
    (name.includes('read') ||
      name.includes('write') ||
      name.includes('delete') ||
      name.includes('edit') ||
      name.includes('strreplace') ||
      name.includes('list') ||
      name === 'ls')
  ) {
    return truncate(path)
  }

  if (oldString && (name.includes('edit') || name.includes('strreplace'))) {
    const target = path ? `${basename(path)}: ` : ''
    return truncate(`${target}${oldString}`)
  }

  if (newString && name.includes('write')) {
    const target = path ? `${basename(path)}` : 'content'
    return truncate(`${target}: ${newString}`)
  }

  if (command) return truncate(command)
  if (path) return truncate(path)
  if (query) return truncate(query)
  if (url) return truncate(url)
  if (pattern) return truncate(pattern)

  const fallback = firstStringArg(args)
  if (fallback) return truncate(fallback)

  const json = JSON.stringify(args)
  if (json.length <= MAX_SUMMARY_LENGTH) return json

  return undefined
}
