// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { waitForAppReady, openCustomize } from './helpers'

test('extensions tab loads', async ({ page }) => {
  await waitForAppReady(page)
  await openCustomize(page)
  await page.locator('.customize-tabs').getByRole('button', { name: 'Extensions', exact: true }).click()
  await expect(page.getByText(/no extensions installed|installed extensions/i)).toBeVisible()
})
