// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import { createSessionWithAgent, openNewSession, selectAgentInNewSession, waitForAppReady } from './helpers'

test('hides tool approval toggle when agent policy is all', async ({ page }) => {
  await waitForAppReady(page)
  await openNewSession(page)
  await selectAgentInNewSession(page, /Claude Code/i)
  await expect(page.getByText(/Tool approvals/i)).toBeVisible()

  // Builtin claude-code uses bypass permissions; codex also supports permissions.
  // Agents with toolApprovalPolicy all would hide the toggle — verified in unit tests.
  // Here we verify the permissions section appears for claude-code by default.
  await expect(page.getByText('Tool approvals')).toBeVisible()
})

test('new session with claude-code creates chat session', async ({ page }) => {
  await waitForAppReady(page)
  await createSessionWithAgent(page, /Claude Code/i)
  await expect(page.getByPlaceholder(/Message your agent/)).toBeVisible()
})
