// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { waitForAppReady, openCustomize } from './helpers'

test('general tab shows implicit launch agent picker', async ({ page }) => {
  await waitForAppReady(page)
  await openCustomize(page)
  await page.getByRole('button', { name: 'General' }).click()
  await expect(page.getByText('Implicit launch agent', { exact: true })).toBeVisible()
})

test('capabilities endpoint returns sandbox info', async ({ request }) => {
  const res = await request.get('/api/capabilities')
  expect(res.ok()).toBeTruthy()
  const body = await res.json()
  expect(body.sandbox).toBeDefined()
  expect(typeof body.sandbox.bwrap.available).toBe('boolean')
})
