// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { afterEach, describe, expect, it, vi } from 'vitest'
import { scrollToSidebarSession } from './scrollToSidebarSession'

describe('scrollToSidebarSession', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('scrolls to the matching sidebar session item', () => {
    const item = document.createElement('div')
    item.setAttribute('data-sidebar-session-id', 'session-1')
    item.scrollIntoView = () => {}
    const scrollIntoView = vi.spyOn(item, 'scrollIntoView')
    document.body.appendChild(item)

    expect(scrollToSidebarSession('session-1')).toBe(true)
    expect(scrollIntoView).toHaveBeenCalledWith({ block: 'nearest', behavior: 'smooth' })
  })

  it('returns false when the session item is missing', () => {
    expect(scrollToSidebarSession('missing')).toBe(false)
  })
})
