// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { editDistance, fuzzyMatchScore, fuzzyMatches, fuzzyTokenScore } from '@/lib/fuzzyMatch'

describe('editDistance', () => {
  it('counts substitutions and inserts', () => {
    expect(editDistance('analyzer', 'analyzer')).toBe(0)
    expect(editDistance('analzyer', 'analyzer')).toBe(2)
    expect(editDistance('anlyzer', 'analyzer')).toBe(1)
  })
})

describe('fuzzyTokenScore', () => {
  it('scores exact, prefix, and typo matches', () => {
    expect(fuzzyTokenScore('postgres', 'postgres')).toBe(100)
    expect(fuzzyTokenScore('post', 'postgres')).toBeGreaterThan(80)
    expect(fuzzyTokenScore('postgre', 'postgres')).toBeGreaterThan(0)
    expect(fuzzyTokenScore('xyz', 'postgres')).toBe(0)
  })
})

describe('fuzzyMatchScore', () => {
  it('matches non-contiguous query tokens', () => {
    const text = 'Postgres Query Analyzer'
    expect(fuzzyMatches('postgres analyzer', text)).toBe(true)
    expect(fuzzyMatchScore('postgres analyzer', text)).toBeGreaterThan(
      fuzzyMatchScore('postgres analyzer', 'Random Coding Agent'),
    )
  })

  it('matches smashed queries without spaces', () => {
    const text = 'Postgres Query Analyzer'
    expect(fuzzyMatches('postgresanalyzer', text)).toBe(true)
    expect(fuzzyMatchScore('postgresanalyzer', text)).toBeGreaterThan(0)
  })

  it('tolerates small typos', () => {
    expect(fuzzyMatches('postgre analizer', 'Postgres Query Analyzer')).toBe(true)
    expect(fuzzyMatches('postgresanalizer', 'Postgres Query Analyzer')).toBe(true)
  })

  it('rejects unrelated queries', () => {
    expect(fuzzyMatches('kubernetes deploy', 'Postgres Query Analyzer')).toBe(false)
    expect(fuzzyMatches('kubernetesdeploy', 'Postgres Query Analyzer')).toBe(false)
  })
})
