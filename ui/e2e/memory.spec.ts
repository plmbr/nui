// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { waitForAppReady, openCustomize } from './helpers'

test('memory tab loads user memory controls', async ({ page }) => {
  await waitForAppReady(page)
  await openCustomize(page)
  const panel = page.locator('.customize-panel')
  await panel.locator('.customize-tabs').getByRole('button', { name: 'Memory', exact: true }).click()
  await expect(page.getByText('Persistent markdown memory')).toBeVisible({ timeout: 15_000 })
  await expect(page.getByText('User memory', { exact: true })).toBeVisible()
  await expect(page.getByLabel('Memory mode').first()).toBeVisible()
})
