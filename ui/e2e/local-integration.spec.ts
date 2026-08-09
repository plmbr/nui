// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { test, expect } from '@playwright/test'
import {
  createSessionWithAgent,
  openCustomizeTab,
  sendChatMessage,
  skipUnlessLocalIntegration,
  waitForAppReady,
} from './helpers'

test.describe('local integration @local-only', () => {
  test.beforeEach(() => {
    skipUnlessLocalIntegration()
  })

  test('MCP servers tab shows seeded stdio server and saves changes', async ({ page, request }) => {
    await waitForAppReady(page)
    await openCustomizeTab(page, 'MCP servers')
    await expect(page.getByText('Loading MCP servers…')).toBeHidden()

    const nameInput = page.locator('.customize-tab-content input').first()
    await expect(nameInput).toHaveValue('e2e-mcp')

    await nameInput.fill('e2e-mcp-renamed')
    await page.getByRole('button', { name: 'Save changes' }).click()

    const listRes = await request.get('/api/mcp-servers')
    expect(listRes.ok()).toBeTruthy()
    const body = await listRes.json() as { mcpServers: Array<{ name: string }> }
    expect(body.mcpServers.some((server) => server.name === 'e2e-mcp-renamed')).toBeTruthy()

    await nameInput.fill('e2e-mcp')
    await page.getByRole('button', { name: 'Save changes' }).click()
  })

  test('MCP ping tool responds through nui proxy', async ({ request }) => {
    const res = await request.post('/mcp-call-tool', {
      data: { server: 'e2e-mcp', name: 'ping', arguments: { message: '' } },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json() as { content?: Array<{ text?: string }> }
    const text = JSON.stringify(body)
    expect(text).toContain('pong')
  })

  test('skills tab lists seeded user skill', async ({ page, request }) => {
    await waitForAppReady(page)
    await openCustomizeTab(page, 'Skills')
    await expect(page.getByText('Loading skills…')).toBeHidden()
    await expect(page.locator('li').filter({ hasText: 'e2e-test-skill' })).toBeVisible()

    const res = await request.get('/api/skills')
    expect(res.ok()).toBeTruthy()
    const skills = await res.json() as Array<{ name: string }>
    expect(skills.some((skill) => skill.name === 'e2e-test-skill')).toBeTruthy()
  })

  test('agent form can attach seeded user skill', async ({ page, request }) => {
    const skillsRes = await request.get('/api/skills')
    expect(skillsRes.ok()).toBeTruthy()
    const skills = await skillsRes.json() as Array<{ name: string }>
    expect(skills.some((skill) => skill.name === 'e2e-test-skill')).toBeTruthy()

    await waitForAppReady(page)
    await openCustomizeTab(page, 'Agents')
    await expect(page.getByText('Loading agents…')).toBeHidden()
    await page.getByRole('button', { name: 'New agent' }).click()
    const form = page.locator('.agent-form')
    await expect(form).toBeVisible()

    await page.getByRole('button', { name: 'YAML' }).click()
    const yamlEditor = page.locator('textarea').first()
    await yamlEditor.fill(`adl: "1.0"
id: e2e-agent
name: E2E Agent
harness:
  type: claude-code
skills:
  - name: e2e-test-skill
    ref: e2e-test-skill
`)
    await expect(yamlEditor).toHaveValue(/e2e-test-skill/)
  })

  test('tool calling surfaces HITL prompt and accepts an answer', async ({ page }) => {
    test.setTimeout(120_000)
    await waitForAppReady(page)
    await createSessionWithAgent(page, /Ollama/i)
    await sendChatMessage(page, 'which do you prefer: is 2+2 equal to 4 or 5?')

    const prompt = page.locator('.hitl-prompt')
    await expect(prompt).toBeVisible({ timeout: 60_000 })
    await expect(prompt).toContainText(/2\+2|Math/i)

    await prompt.locator('.hitl-prompt__option-label', { hasText: '4' }).click()
    await expect(prompt.getByRole('button', { name: 'Submit' })).toBeEnabled()
    await prompt.getByRole('button', { name: 'Submit' }).click()

    await expect(prompt).toBeHidden({ timeout: 60_000 })
    await expect(page.locator('.agui-message--assistant .agui-message__bubble').last()).toContainText(
      /2\+2|equals|4/i,
      { timeout: 60_000 },
    )
  })
})

test('local integration env guard', () => {
  if (process.env.CI) {
    expect(process.env.E2E_AGENT).not.toBe('integration')
  }
})
