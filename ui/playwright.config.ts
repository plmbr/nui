// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { defineConfig, devices } from '@playwright/test'

const port = process.env.NUI_E2E_PORT ?? '18080'
const baseURL = `http://127.0.0.1:${port}`

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: 'list',
  timeout: 120_000,
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: '../scripts/e2e-server.sh',
    url: `${baseURL}/health`,
    // Echo/integration/ollama seed agents or mocks; never attach to an unrelated local server.
    reuseExistingServer: !process.env.CI && !process.env.E2E_AGENT,
    timeout: 120_000,
  },
})
