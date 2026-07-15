// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { looksLikeDiff } from './diff'

describe('looksLikeDiff', () => {
  it('detects explicit diff language', () => {
    expect(looksLikeDiff('anything', 'language-diff')).toBe(true)
  })

  it('detects unified diff headers', () => {
    const text = [
      '--- a/file.java',
      '+++ b/file.java',
      '@@ -1,3 +1,3 @@',
      '- old',
      '+ new',
    ].join('\n')
    expect(looksLikeDiff(text)).toBe(true)
  })

  it('does not treat indented java as a diff', () => {
    const text = [
      '@Secured("your-gandalf-policy-name")',
      'public ResponseEntity<MyResponse> myEndpoint() {',
      '    // Only members of the authorized group can reach here',
      '}',
    ].join('\n')
    expect(looksLikeDiff(text)).toBe(false)
  })

  it('does not treat list-indented markdown code as a diff', () => {
    const text = [
      '    @Secured("policy")',
      '    public String endpoint() {',
      '        return "ok";',
      '    }',
    ].join('\n')
    expect(looksLikeDiff(text)).toBe(false)
  })
})
