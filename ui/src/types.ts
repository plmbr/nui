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
  agentConfig?: Record<string, unknown>
}

export interface DirectorySuggestions {
  directories: string[]
}

export interface AgentType {
  id: string
  label: string
  description?: string
  harness: 'claude-code' | 'pi' | 'codex' | 'opencode' | 'docker' | 'remote'
  sandbox?: 'none' | 'bubblewrap' | 'docker'
  isBuiltin: boolean
  available: boolean
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  createdAt: string
}

export interface AppConfig {
  copilotKitPublicApiKey: string
  copilotKitRuntimeUrl: string
}

export interface Settings {
  theme: 'light' | 'dark'
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
