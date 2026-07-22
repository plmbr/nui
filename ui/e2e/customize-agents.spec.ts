// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { waitForAppReady, openCustomize } from './helpers'

test('agents tab supports simplified eval authoring', async ({ page }) => {
  await waitForAppReady(page)
  await openCustomize(page)
  await page.getByRole('button', { name: 'Agents' }).click()
  await page.getByRole('button', { name: 'New agent' }).click()
  await expect(page.locator('.agent-form')).toBeVisible()

  const form = page.locator('.agent-form')
  await form.getByRole('textbox', { name: 'ID' }).fill('e2e-eval-agent')
  await form.getByRole('textbox', { name: 'Name' }).fill('E2E Eval Agent')
  await form.getByRole('button', { name: 'Add eval' }).click()
  await form.getByLabel('Eval name').fill('smoke')
  await form.getByPlaceholder('User message sent to the agent').fill('say hello')
  await form.getByPlaceholder('Substring the response should include').fill('hello')

  await page.getByRole('button', { name: 'Create agent' }).click()
  await expect(page.getByText('E2E Eval Agent')).toBeVisible({ timeout: 15_000 })
  await expect(page.getByRole('button', { name: 'Run evals' })).toBeVisible()
})

test('agents tab supports creating a docker harness agent', async ({ page }) => {
  await waitForAppReady(page)
  await openCustomize(page)
  await page.getByRole('button', { name: 'Agents' }).click()
  await page.getByRole('button', { name: 'New agent' }).click()
  await expect(page.locator('.agent-form')).toBeVisible()

  const harnessSelect = page.locator('.agent-form').getByRole('button', { name: 'Claude Code' })
  await harnessSelect.click()
  await page.getByPlaceholder('Search harnesses…').fill('docker')
  await page.getByRole('option', { name: 'Docker (HTTP/SSE container)' }).click({ force: true })

  await page.getByLabel('Container image').fill('nui-echo-agent')
  await page.getByLabel('Container port').fill('9090')
  const form = page.locator('.agent-form')
  await form.getByRole('textbox', { name: 'ID' }).fill('e2e-docker-agent')
  await form.getByRole('textbox', { name: 'Name' }).fill('E2E Docker Agent')

  await page.getByRole('button', { name: 'Create agent' }).click()
  await expect(page.getByText('E2E Docker Agent')).toBeVisible({ timeout: 15_000 })
})
