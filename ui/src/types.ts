// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export interface Session {
  id: string
  name: string
  workingDir: string
  agentType: string
  agentConfig?: Record<string, unknown>
  createdAt: string
  scheduleId?: string
  scheduleName?: string
  lastRunAt?: string
  mcpAuthWarnings?: string[]
}

export interface CreateSessionRequest {
  name?: string
  workingDir: string
  agentType: string
  agentConfig?: {
    userScopeHarnessConfig?: boolean
    hitlMode?: 'interactive' | 'off' | 'auto'
    harnessPermissions?: 'interactive' | 'bypass'
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

export interface UploadedImage {
  path: string
  url: string
  mediaType: string
  filename: string
}

export interface PromptSuggestion {
  title: string
  prompt: string
  icon?: string
}

export interface AgentType {
  id: string
  label: string
  description?: string
  harness: 'claude-code' | 'pi' | 'codex' | 'opencode' | 'api' | 'docker' | 'devcontainer' | 'remote'
  provider?: 'anthropic' | 'openai' | 'gemini' | 'openrouter' | 'ollama' | string
  sandbox?: 'none' | 'bubblewrap' | 'docker'
  promptMode?: 'user' | 'auto'
  defaultPrompt?: string
  promptSuggestions?: PromptSuggestion[]
  skills?: string[]
  workingDirInput?: boolean
  hitlMode?: 'interactive' | 'off' | 'auto'
  harnessPermissions?: 'interactive' | 'bypass'
  supportsHarnessPermissions?: boolean
  toolApprovalPolicy?: 'default' | 'all' | 'allowlist' | 'denylist'
  toolApprovalTools?: string[]
  isBuiltin: boolean
  source?: 'builtin' | 'user' | 'extension'
  available: boolean
}

export interface HitlQuestionOption {
  label: string
  description?: string
}

export interface HitlQuestion {
  question?: string
  header?: string
  options?: HitlQuestionOption[]
  multiSelect?: boolean
}

export interface HitlAction {
  id: string
  label: string
}

export interface HitlPayload {
  title?: string
  message?: string
  questions?: HitlQuestion[]
  actions?: HitlAction[]
  toolName?: string
  toolInput?: Record<string, unknown>
  description?: string
  [key: string]: unknown
}

export interface HitlRequest {
  requestId: string
  sessionId?: string
  runId?: string
  stepName?: string
  kind: 'question' | 'approval' | 'freeform' | 'review' | string
  payload?: HitlPayload
  status?: string
  expiresAt?: string
  createdAt?: string
}

export interface HitlResponse {
  requestId: string
  status: string
  answers?: Record<string, unknown>
  respondedAt?: string
}

export interface ChatMessagePart {
  type: 'text' | 'tool'
  id?: string
  content?: string
  toolCallId?: string
  toolName?: string
  toolArgs?: Record<string, unknown>
  toolResult?: unknown
  mcpAppResourceUri?: string
  mcpAppServerName?: string
  mcpAppToolInput?: Record<string, unknown>
  visualizationHtml?: string
  visualizationTitle?: string
}

export interface ChatImage {
  id: string
  mediaType: string
  data: string
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  createdAt: string
  parts?: ChatMessagePart[]
  images?: ChatImage[]
  error?: boolean
  routedAgentLabel?: string
}

export interface Settings {
  theme: 'light' | 'dark'
  defaultAgentType?: string
  lastAgentType?: string
  lastSessionId?: string
  sidebarOpen?: boolean
  sidebarWidth?: number
  disabledExtensions?: string[]
  mcpOAuthCallbackUrl?: string
}

export interface ExtensionInfo {
  name: string
  version?: string
  displayName?: string
  description?: string
  disabled: boolean
  harnesses?: string[]
  mcpServers?: string[]
  mcpServerConfigs?: MCPServer[]
  skills?: string[]
  rules?: string[]
  mentionProviders?: string[]
  agents?: string[]
}

export interface MCPServerAuth {
  clientId?: string
  clientSecret?: string
  scopes?: string[]
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
  auth?: MCPServerAuth
}

export type MCPOAuthStatus = 'connected' | 'needs_auth' | 'expired' | 'not_applicable'

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

export interface Schedule {
  id: string
  name: string
  agentType: string
  prompt?: string
  workingDir?: string
  interval?: string
  cron?: string
  runAt?: string
  enabled: boolean
  lastRunAt?: string
  nextRunAt?: string
  lastSessionId?: string
  lastRunId?: string
  createdAt: string
}

export interface CreateScheduleRequest {
  name: string
  agentType: string
  prompt?: string
  workingDir?: string
  interval?: string
  cron?: string
  runAt?: string
}

export interface AgentDeployerInfo {
  id: string
  extension: string
  name: string
  description?: string
}

export interface DeployEndpoint {
  host?: string
  port?: number
  url?: string
}

export interface AgentDeployResult {
  deploymentId?: string
  status?: string
  message?: string
  endpoint?: DeployEndpoint
}

export interface AgentEvalResult {
  name: string
  status: 'pass' | 'fail' | 'error' | 'skip'
  output?: string
  passed?: boolean | null
  message?: string
  error?: string
  duration: string
}

export interface AgentEvalSummary {
  agentId: string
  results: AgentEvalResult[]
  passed: number
  failed: number
  errors: number
  skipped: number
}
