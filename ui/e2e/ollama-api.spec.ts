// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import {
  createSessionWithAgent,
  ollamaAgentAvailable,
  sendChatMessage,
  waitForAppReady,
} from './helpers'

function assistantMessages(page: import('@playwright/test').Page) {
  return page.locator('.agui-message--assistant .agui-message__bubble')
}

test('ollama api does not leak raw tool JSON into chat', async ({ page }) => {
  test.skip(!ollamaAgentAvailable(), 'set E2E_AGENT=ollama to run ollama api tests')

  await waitForAppReady(page)
  await createSessionWithAgent(page, /Ollama/i)
  await sendChatMessage(page, 'what is 2+2')

  const last = assistantMessages(page).last()
  await expect(last).toBeVisible({ timeout: 45_000 })
  await expect(last).not.toContainText(/"name"\s*:\s*"ask_user"/, { timeout: 45_000 })
  await expect(last).not.toContainText(/"parameters"\s*:/, { timeout: 45_000 })
  await expect(page.locator('.agui-message__visualization-part')).toHaveCount(0)
})

test('ollama api answers capability questions without blank visualization', async ({ page }) => {
  test.skip(!ollamaAgentAvailable(), 'set E2E_AGENT=ollama to run ollama api tests')

  await waitForAppReady(page)
  await createSessionWithAgent(page, /Ollama/i)
  await sendChatMessage(page, 'what can you do')

  const last = assistantMessages(page).last()
  await expect(last).toBeVisible({ timeout: 45_000 })
  await expect(page.locator('.visualization-frame')).toHaveCount(0)
  await expect(last).not.toContainText(/"name"\s*:\s*"show_visualization"/, { timeout: 45_000 })
  await expect(page.locator('.hitl-prompt')).toHaveCount(0)
  await expect(last).not.toContainText(/"name"\s*:\s*"ask_user"/)
  await expect(last).not.toContainText(/Choose an action/)
  await expect(last).toContainText(/help|answer|chart|task/i)
})

test('ollama api answers factual questions as text not visualization', async ({ page }) => {
  test.skip(!ollamaAgentAvailable(), 'set E2E_AGENT=ollama to run ollama api tests')

  await waitForAppReady(page)
  await createSessionWithAgent(page, /Ollama/i)
  await sendChatMessage(page, 'What is the capital of France')

  const last = assistantMessages(page).last()
  await expect(last).toBeVisible({ timeout: 45_000 })
  await expect(last).toContainText(/Paris/i)
  await expect(page.locator('.visualization-frame')).toHaveCount(0)
  await expect(page.locator('.hitl-prompt')).toHaveCount(0)
})

test('ollama api responds to the current message, not the previous one', async ({ page }) => {
  test.skip(!ollamaAgentAvailable(), 'set E2E_AGENT=ollama to run ollama api tests')

  await waitForAppReady(page)
  await createSessionWithAgent(page, /Ollama/i)
  await sendChatMessage(page, 'what is 2+2')
  await expect(assistantMessages(page).last()).toBeVisible({ timeout: 45_000 })

  await sendChatMessage(page, 'hi')
  const last = assistantMessages(page).last()
  await expect(last).toContainText(/hello/i, { timeout: 45_000 })
  await expect(last).not.toContainText(/what is 2\+2/i)
  await expect(last).not.toContainText(/"name"\s*:\s*"ask_user"/)
})
