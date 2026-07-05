// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { createSessionWithAgent, openNewSession, waitForAppReady } from './helpers'

test('new session wizard opens and lists agents', async ({ page }) => {
  await waitForAppReady(page)
  await openNewSession(page)
  const panel = page.locator('.customize-panel').filter({
    has: page.getByRole('heading', { name: 'New Session' }),
  })
  await expect(panel.getByRole('button', { name: 'Claude Code', exact: true })).toBeVisible()
})

test('create claude-code session and land on chat route', async ({ page }) => {
  await waitForAppReady(page)
  await createSessionWithAgent(page, /Claude Code/i)
  await expect(page.getByPlaceholder(/Message your agent/)).toBeVisible()
})

test('schedules route loads via direct navigation', async ({ page }) => {
  await page.goto('/schedules')
  await expect(page.getByRole('heading', { name: /Schedules/i })).toBeVisible()
})
