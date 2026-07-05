// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { waitForAppReady, openCustomize } from './helpers'

test('customize panel tabs load', async ({ page }) => {
  await waitForAppReady(page)
  await openCustomize(page)
  await expect(page.getByRole('button', { name: 'General' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Extensions' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Skills' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Agents' })).toBeVisible()
})
