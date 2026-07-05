// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import {
  createSessionWithAgent,
  echoAgentAvailable,
  realAgentAvailable,
  sendChatMessage,
  waitForAppReady,
} from './helpers'

test('echo agent chat returns streamed reply', async ({ page }) => {
  test.skip(!echoAgentAvailable(), 'set E2E_AGENT=echo to run echo chat tests')

  await waitForAppReady(page)
  await createSessionWithAgent(page, /E2E Echo/i)
  await sendChatMessage(page, 'hello loop')
  await expect(page.getByText(/Echo: hello loop/i)).toBeVisible({ timeout: 30_000 })

  await page.reload()
  await expect(page.getByText(/Echo: hello loop/i)).toBeVisible({ timeout: 15_000 })
})

test('claude-code chat returns deterministic reply', async ({ page }) => {
  test.skip(!realAgentAvailable(), 'needs ANTHROPIC_API_KEY')

  test.setTimeout(120_000)
  await waitForAppReady(page)
  await createSessionWithAgent(page, /Claude Code/i)
  await sendChatMessage(page, 'Reply with exactly: LOOP_E2E_OK')
  await expect(page.getByText(/LOOP_E2E_OK/)).toBeVisible({ timeout: 90_000 })
})
