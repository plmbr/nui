// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export interface Session {
  id: string
  name: string
  workingDir: string
  agentType: string
  agentConfig?: Record<string, unknown>
  createdAt: string
}

export interface CreateSessionRequest {
  name: string
  workingDir: string
  agentType: string
  agentConfig?: {
    userScopeHarnessConfig?: boolean
    [key: string]: unknown
  }
}

export interface DirectorySuggestions {
  directories: string[]
}

export interface MentionItem {
  label: string
  value: string
  hasChildren: boolean
  icon?: string
}

export interface MentionBreadcrumb {
  label: string
  parent: string
}

export interface MentionListResponse {
  items: MentionItem[]
  breadcrumb: MentionBreadcrumb[]
}

export interface AgentType {
  id: string
  label: string
  description?: string
  harness: 'claude-code' | 'pi' | 'codex' | 'opencode' | 'docker' | 'remote'
  sandbox?: 'none' | 'bubblewrap' | 'docker'
  promptMode?: 'user' | 'auto'
  defaultPrompt?: string
  workingDirInput?: boolean
  isBuiltin: boolean
  source?: 'builtin' | 'user' | 'extension'
  available: boolean
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  createdAt: string
}

export interface Settings {
  theme: 'light' | 'dark'
  defaultAgentType?: string
  lastAgentType?: string
  lastSessionId?: string
  sidebarOpen?: boolean
  disabledExtensions?: string[]
}

export interface ExtensionInfo {
  name: string
  version?: string
  displayName?: string
  description?: string
  disabled: boolean
  harnesses?: string[]
  mcpServers?: string[]
  skills?: string[]
  instructions?: string[]
  agents?: string[]
}

export interface MCPServer {
  name: string
  ref?: string
  url?: string
  command?: string
  args?: string[]
  type?: string
  env?: Record<string, string>
  headers?: Record<string, string>
}

export interface SkillEntry {
  name: string
  source: string
  path?: string
  git?: string
  version?: string
  installedAt: string
}

export interface AgentFileInfo {
  file: string
  id: string
  name: string
  description?: string
}

export interface AgentFileContent {
  file: string
  content: string
}

export interface Bootstrap {
  sessionId?: string
  initialPrompt?: string
  sidebarOpen?: boolean
  hideInput?: boolean
}

export interface BwrapStatus {
  available: boolean
  path?: string
  error?: string
}

export interface Capabilities {
  sandbox: {
    bwrap: BwrapStatus
  }
}
