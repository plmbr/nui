// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

/** Split into lowercase alphanumeric tokens (spaces, punctuation, camel/kebab boundaries). */
export function fuzzyTokens(text: string): string[] {
  return text
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter(Boolean)
}

/** Lowercase alphanumeric only — treats missing/extra spaces as irrelevant. */
export function compactAlnum(text: string): string {
  return text.toLowerCase().replace(/[^a-z0-9]/g, '')
}

function maxEditDistance(len: number): number {
  if (len <= 2) return 0
  if (len <= 5) return 1
  return 2
}

/** Levenshtein distance for short strings (autocomplete queries). */
export function editDistance(a: string, b: string): number {
  if (a === b) return 0
  if (a.length === 0) return b.length
  if (b.length === 0) return a.length
  const prev = new Array<number>(b.length + 1)
  const cur = new Array<number>(b.length + 1)
  for (let j = 0; j <= b.length; j++) prev[j] = j
  for (let i = 1; i <= a.length; i++) {
    cur[0] = i
    for (let j = 1; j <= b.length; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1
      cur[j] = Math.min(
        (prev[j] ?? 0) + 1,
        (cur[j - 1] ?? 0) + 1,
        (prev[j - 1] ?? 0) + cost,
      )
    }
    for (let j = 0; j <= b.length; j++) prev[j] = cur[j] ?? 0
  }
  return prev[b.length] ?? 0
}

/**
 * Score how well a query token matches a candidate token.
 * Returns 0 for no match; higher is better.
 */
export function fuzzyTokenScore(queryToken: string, candidateToken: string): number {
  const q = queryToken.toLowerCase()
  const c = candidateToken.toLowerCase()
  if (!q || !c) return 0
  if (c === q) return 100
  if (c.startsWith(q)) return 80 + Math.min(19, q.length)
  if (c.includes(q)) return 60
  if (q.length <= 2) return 0

  const maxDist = maxEditDistance(q.length)
  if (Math.abs(c.length - q.length) > maxDist) return 0
  const dist = editDistance(q, c)
  if (dist > maxDist) return 0
  return Math.max(1, 45 - dist * 15)
}

/**
 * Cover a compacted query by concatenating a subsequence of text tokens.
 * For example, "postgresanalyzer" matches tokens ["postgres","query","analyzer"] via postgres+analyzer.
 */
function concatenatedTokenCoverScore(queryCompact: string, textTokens: string[]): number {
  if (queryCompact.length < 3 || textTokens.length === 0) return 0

  const memo = new Map<string, number>()

  const search = (qi: number, ti: number): number => {
    if (qi >= queryCompact.length) return 1
    if (ti >= textTokens.length) return 0

    const key = `${qi}:${ti}`
    const cached = memo.get(key)
    if (cached !== undefined) return cached

    // Skip this text token (allows "postgres" + "analyzer" while skipping "query").
    let best = search(qi, ti + 1)

    const token = textTokens[ti]!
    const remaining = queryCompact.length - qi

    if (queryCompact.startsWith(token, qi)) {
      const next = search(qi + token.length, ti + 1)
      if (next > 0) best = Math.max(best, 100 + next)
    }

    // Partial token at the end of the query (still typing).
    if (remaining > 0 && remaining < token.length) {
      const slice = queryCompact.slice(qi)
      if (token.startsWith(slice)) {
        const next = search(queryCompact.length, ti + 1)
        if (next > 0) best = Math.max(best, 70 + remaining + next)
      }
    }

    // Typo within an approximately token-sized slice.
    if (token.length >= 3 && remaining >= Math.max(3, token.length - 2)) {
      const minLen = Math.max(3, token.length - 2)
      const maxLen = Math.min(remaining, token.length + 2)
      for (let len = minLen; len <= maxLen; len++) {
        const slice = queryCompact.slice(qi, qi + len)
        const tokenScore = fuzzyTokenScore(slice, token)
        if (tokenScore <= 0) continue
        const next = search(qi + len, ti + 1)
        if (next > 0) best = Math.max(best, tokenScore + next)
      }
    }

    memo.set(key, best)
    return best
  }

  const raw = search(0, 0)
  // raw includes a +1 success sentinel; require a real token contribution.
  return raw > 1 ? raw : 0
}

function tokenWiseMatchScore(queryTokens: string[], textTokens: string[], normalizedText: string): number {
  let score = 0
  let lastIndex = -1
  let inOrder = true
  for (const qt of queryTokens) {
    let best = 0
    let bestIndex = -1
    for (let i = 0; i < textTokens.length; i++) {
      const tokenScore = fuzzyTokenScore(qt, textTokens[i]!)
      if (tokenScore > best) {
        best = tokenScore
        bestIndex = i
      }
    }
    // Also allow the query token as a substring of the full text (ids, etc.).
    if (best === 0 && qt.length >= 2 && normalizedText.includes(qt)) {
      best = 50
      bestIndex = lastIndex + 1
    }
    if (best === 0) return 0
    score += best
    if (bestIndex <= lastIndex) inOrder = false
    lastIndex = Math.max(lastIndex, bestIndex)
  }

  if (inOrder) score += 25 * queryTokens.length
  score += Math.max(0, 20 - textTokens.length)
  return score
}

/**
 * Fuzzy / similarity score of `query` against `text`.
 * Multi-word queries match when every query token hits some text token
 * (order preferred but not required), so "postgres analyzer" matches
 * "Postgres Query Analyzer". Missing spaces ("postgresanalyzer") and small
 * typos are also tolerated.
 * Returns 0 when there is no match.
 */
export function fuzzyMatchScore(query: string, text: string): number {
  const normalizedQuery = query.trim().toLowerCase()
  if (!normalizedQuery) return 1
  const normalizedText = text.toLowerCase()
  if (!normalizedText) return 0

  if (normalizedText === normalizedQuery) return 1000
  if (normalizedText.includes(normalizedQuery)) {
    return 500 + Math.min(99, normalizedQuery.length)
  }

  const queryCompact = compactAlnum(normalizedQuery)
  const textCompact = compactAlnum(normalizedText)
  if (queryCompact && textCompact) {
    if (textCompact === queryCompact) return 900
    if (textCompact.includes(queryCompact)) {
      return 450 + Math.min(99, queryCompact.length)
    }
  }

  const queryTokens = fuzzyTokens(normalizedQuery)
  if (queryTokens.length === 0) return 0
  const textTokens = fuzzyTokens(normalizedText)
  if (textTokens.length === 0) return 0

  const tokenScore = tokenWiseMatchScore(queryTokens, textTokens, normalizedText)
  const smashedScore = concatenatedTokenCoverScore(queryCompact, textTokens)
  const score = Math.max(tokenScore, smashedScore)
  return score > 0 ? score : 0
}

export function fuzzyMatches(query: string, text: string): boolean {
  return fuzzyMatchScore(query, text) > 0
}
