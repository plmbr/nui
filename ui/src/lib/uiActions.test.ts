// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it, vi } from 'vitest'
import {
  applyUIActions,
  parseControlUIToolResult,
  parseUIActionCustomValue,
  parseUIActions,
} from '@/lib/uiActions'

describe('uiActions', () => {
  it('parses navigate and theme actions', () => {
    expect(
      parseUIActions([
        { type: 'navigate', target: 'customize' },
        { type: 'set_theme', theme: 'dark' },
        { type: 'navigate', target: 'nope' },
      ]),
    ).toEqual([
      { type: 'navigate', target: 'customize' },
      { type: 'set_theme', theme: 'dark' },
    ])
  })

  it('parses CUSTOM ui_action value', () => {
    const event = parseUIActionCustomValue(
      { actions: [{ type: 'navigate', target: 'new_session' }], toolCallId: 't1' },
      'src',
    )
    expect(event).toEqual({
      actions: [{ type: 'navigate', target: 'new_session' }],
      toolCallId: 't1',
      sourceSessionId: 'src',
    })
  })

  it('parses control_ui tool result JSON', () => {
    const event = parseControlUIToolResult(
      JSON.stringify({
        ok: true,
        actions: [{ type: 'set_theme', theme: 'light' }],
      }),
      'src',
      'call-1',
    )
    expect(event?.actions).toEqual([{ type: 'set_theme', theme: 'light' }])
  })

  it('applyUIActions dispatches handlers', () => {
    const handlers = {
      navigateCustomize: vi.fn(),
      navigateNewSession: vi.fn(),
      navigateLaunch: vi.fn(),
      navigateSchedules: vi.fn(),
      setTheme: vi.fn(),
      refreshUI: vi.fn(),
    }
    applyUIActions(
      [
        { type: 'navigate', target: 'customize' },
        { type: 'set_theme', theme: 'dark' },
        { type: 'refresh_ui' },
      ],
      handlers,
    )
    expect(handlers.navigateCustomize).toHaveBeenCalledOnce()
    expect(handlers.setTheme).toHaveBeenCalledWith('dark')
    expect(handlers.refreshUI).toHaveBeenCalledOnce()
  })
})
