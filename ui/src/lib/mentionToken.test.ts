// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { mentionToken } from '@/lib/mentionToken'

describe('mentionToken', () => {
  it('leaves values an unquoted token can hold as they are', () => {
    expect(mentionToken('file:docs/readme.md')).toBe('@file:docs/readme.md')
    expect(mentionToken('builtin:files')).toBe('@builtin:files')
  })

  it('quotes values that contain whitespace or @', () => {
    // Unquoted, the server would read these as `data/my` and `img/logo`.
    expect(mentionToken('file:data/my notes.md')).toBe('@file:"data/my notes.md"')
    expect(mentionToken('dir:My Folder')).toBe('@dir:"My Folder"')
    expect(mentionToken('file:img/logo@2x.png')).toBe('@file:"img/logo@2x.png"')
    expect(mentionToken('ext:demo:catalog:Q3 orders')).toBe('@ext:"demo:catalog:Q3 orders"')
    expect(mentionToken('file:a\u001cb')).toBe('@file:"a\u001cb"')
  })

  it('leaves values the quoted form cannot hold unquoted', () => {
    expect(mentionToken('file:say "hi" now.md')).toBe('@file:say "hi" now.md')
    expect(mentionToken('ext:demo:catalog:a\u2028b c')).toBe('@ext:demo:catalog:a\u2028b c')
  })
})
