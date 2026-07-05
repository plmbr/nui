// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'

test('schedules panel renders', async ({ page }) => {
  await page.goto('/schedules')
  await expect(page.getByRole('heading', { name: /Schedules/i })).toBeVisible()
  await expect(page.getByRole('button', { name: /New schedule/i })).toBeVisible()
})

test('schedules page survives hard refresh', async ({ page }) => {
  await page.goto('/schedules')
  await expect(page.getByRole('heading', { name: /Schedules/i })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('heading', { name: /Schedules/i })).toBeVisible()
})
