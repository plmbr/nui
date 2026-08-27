// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { expect, test, type Locator, type Page } from '@playwright/test'

export async function waitForAppReady(page: Page) {
  await page.goto('/launch')
  await page.getByRole('textbox', { name: 'Launch prompt' }).waitFor({ state: 'visible' })
}

export async function openNewSession(page: Page) {
  // Prefer landing CTA; avoid matching sidebar / recent-session rows.
  const landingCta = page.locator('.landing-page__actions').getByRole('button', {
    name: 'New Session',
    exact: true,
  })
  if (await landingCta.isVisible().catch(() => false)) {
    await landingCta.click()
  } else {
    await page.getByRole('main').getByRole('button', { name: 'New Session', exact: true }).click()
  }
  await page.getByRole('heading', { name: 'New Session' }).waitFor()
}

export function newSessionPanel(page: Page) {
  return page.locator('.customize-panel').filter({
    has: page.getByRole('heading', { name: 'New Session' }),
  })
}

/** Agent picker cards only — excludes Recents rows in the same panel. */
export function newSessionAgentButton(panel: Locator, agentLabel: string | RegExp) {
  return panel
    .locator('[aria-label="Built-in agents"], [aria-label="Installed agents"]')
    .getByRole('button', { name: agentLabel })
}

export function defaultE2EAgentLabel(): RegExp {
  return echoAgentAvailable() ? /E2E Echo/i : /Claude Code/i
}

export async function ensureAgentVisibleInNewSession(page: Page, agentLabel: string | RegExp) {
  const panel = newSessionPanel(page)
  const agentButton = newSessionAgentButton(panel, agentLabel)
  if (await agentButton.isVisible()) {
    return
  }

  const installedButton = panel.getByRole('button', { name: /Installed agents/i })
  if (await installedButton.isVisible()) {
    await installedButton.click()
    if (await agentButton.isVisible()) {
      return
    }
  }

  const backButton = panel.getByRole('button', { name: /Built-in agents/i })
  if (await backButton.isVisible()) {
    await backButton.click()
    if (await agentButton.isVisible()) {
      return
    }
  }
}

export async function showBuiltinAgentsInNewSession(page: Page) {
  const panel = newSessionPanel(page)
  const backButton = panel.getByRole('button', { name: /Built-in agents/i })
  if (await backButton.isVisible()) {
    await backButton.click()
  }
}

export async function selectAgentInNewSession(page: Page, agentLabel: string | RegExp) {
  const panel = newSessionPanel(page)
  await ensureAgentVisibleInNewSession(page, agentLabel)
  await newSessionAgentButton(panel, agentLabel).click()
}

export async function openCustomize(page: Page) {
  const landingCta = page.locator('.landing-page__actions').getByRole('button', {
    name: 'Customize',
    exact: true,
  })
  if (await landingCta.isVisible().catch(() => false)) {
    await landingCta.click()
  } else {
    await page.getByRole('main').getByRole('button', { name: 'Customize', exact: true }).click()
  }
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

export function localIntegrationAvailable(): boolean {
  return process.env.E2E_AGENT === 'integration'
}

export function skipUnlessLocalIntegration() {
  test.skip(
    !!process.env.CI || !localIntegrationAvailable(),
    'run locally with: E2E_AGENT=integration npm run test:e2e:local',
  )
}

export async function openCustomizeTab(page: Page, tab: string) {
  await openCustomize(page)
  await page.locator('.customize-tabs').getByRole('button', { name: tab, exact: true }).click()
}

export async function waitForAssistantReply(page: Page) {
  await expect(page.locator('.agui-message--assistant .agui-message__bubble').last()).toBeVisible({
    timeout: 45_000,
  })
}
