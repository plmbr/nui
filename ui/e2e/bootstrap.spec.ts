// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { waitForAppReady } from './helpers'

test('health endpoint responds', async ({ request }) => {
  const res = await request.get('/health')
  expect(res.ok()).toBeTruthy()
  const body = await res.json()
  expect(body).toMatchObject({ status: 'ok' })
  expect(typeof body.version).toBe('string')
  expect(body.version.length).toBeGreaterThan(0)
})

test('launch page loads with prompt input and actions', async ({ page }) => {
  await waitForAppReady(page)
  await expect(page.getByRole('textbox', { name: 'Launch prompt' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Submit prompt' })).toBeVisible()
  await expect(page.getByRole('main').getByRole('button', { name: 'New Session' })).toBeVisible()
  await expect(page.getByRole('main').getByRole('button', { name: 'Customize' })).toBeVisible()
})
