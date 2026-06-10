// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export interface Project {
  id: string
  name: string
  workingDir: string
  agentType: string
  createdAt: string
}

export interface CreateProjectRequest {
  name: string
  workingDir: string
  agentType: string
}

export interface AgentType {
  id: string
  label: string
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
