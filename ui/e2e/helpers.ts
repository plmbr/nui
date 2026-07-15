// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { expect, type Page } from '@playwright/test'

export async function waitForAppReady(page: Page) {
  await page.goto('/launch')
  await page.getByRole('heading', { name: 'The Loop' }).waitFor()
}

export async function openNewSession(page: Page) {
  await page.getByRole('main').getByRole('button', { name: 'New Session' }).click()
  await page.getByRole('heading', { name: 'New Session' }).waitFor()
}

export function newSessionPanel(page: Page) {
  return page.locator('.customize-panel').filter({
    has: page.getByRole('heading', { name: 'New Session' }),
  })
}

export async function selectAgentInNewSession(page: Page, agentLabel: string | RegExp) {
  await newSessionPanel(page).getByRole('button', { name: agentLabel }).click()
}

export async function openCustomize(page: Page) {
  await page.getByRole('main').getByRole('button', { name: 'Customize' }).click()
  await page.waitForURL(/\/customize/)
}

export async function createSessionWithAgent(page: Page, agentLabel: string | RegExp) {
  await openNewSession(page)
  await selectAgentInNewSession(page, agentLabel)
  await page.getByRole('button', { name: 'Create Session' }).click()
  await page.waitForURL(/\/sessions\/[^/]+$/)
  await page.getByPlaceholder(/Message your agent/).waitFor({ state: 'visible' })
}

export async function sendChatMessage(page: Page, text: string) {
  const input = page.getByPlaceholder(/Message your agent/)
  await input.click()
  // fill() alone can miss React onChange; pressSequentially drives the controlled input.
  await input.fill('')
  await input.pressSequentially(text)
  const sendButton = page.getByRole('button', { name: 'Send message' })
  await expect(sendButton).toBeEnabled({ timeout: 10_000 })
  await sendButton.click()
}

export function echoAgentAvailable(): boolean {
  return process.env.E2E_AGENT === 'echo'
}

export function realAgentAvailable(): boolean {
  return !!process.env.ANTHROPIC_API_KEY
}

export function ollamaAgentAvailable(): boolean {
  return process.env.E2E_AGENT === 'ollama'
}

export async function waitForAssistantReply(page: Page) {
  await expect(page.locator('.agui-message--assistant .agui-message__bubble').last()).toBeVisible({
    timeout: 45_000,
  })
}
