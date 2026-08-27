// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { waitForAppReady, openCustomize } from './helpers'

test('customize panel tabs load', async ({ page }) => {
  await waitForAppReady(page)
  await openCustomize(page)
  const tabs = page.locator('.customize-tabs')
  await expect(tabs.getByRole('button', { name: 'General', exact: true })).toBeVisible()
  await expect(tabs.getByRole('button', { name: 'Extensions', exact: true })).toBeVisible()
  await expect(tabs.getByRole('button', { name: 'Skills', exact: true })).toBeVisible()
  await expect(tabs.getByRole('button', { name: 'Agents', exact: true })).toBeVisible()
})
