// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import {
  createSessionWithAgent,
  defaultE2EAgentLabel,
  ensureAgentVisibleInNewSession,
  newSessionPanel,
  openNewSession,
  waitForAppReady,
} from './helpers'

test('new session wizard opens and lists agents', async ({ page }) => {
  await waitForAppReady(page)
  await openNewSession(page)
  const agentLabel = defaultE2EAgentLabel()
  await ensureAgentVisibleInNewSession(page, agentLabel)
  const panel = newSessionPanel(page)
  await expect(panel.getByRole('button', { name: agentLabel })).toBeVisible()
})

test('create default agent session and land on chat route', async ({ page }) => {
  await waitForAppReady(page)
  await createSessionWithAgent(page, defaultE2EAgentLabel())
  await expect(page.getByPlaceholder(/Message your agent/)).toBeVisible()
})

test('schedules route loads via direct navigation', async ({ page }) => {
  await page.goto('/schedules')
  await expect(page.getByRole('heading', { name: /Schedules/i })).toBeVisible()
})
