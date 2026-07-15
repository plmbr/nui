// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

// Message roles.
const (
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
	RoleUser      = "user"
)

// Finish reasons.
const (
	FinishReasonContentFilter = "content_filter"
	FinishReasonLength        = "length"
	FinishReasonStop          = "stop"
	FinishReasonToolCalls     = "tool_calls"
)

// Provider is the core interface for LLM API clients.
type Provider interface {
	Name() string
	Completion(ctx Context, params CompletionParams) (*ChatCompletion, error)
	CompletionStream(ctx Context, params CompletionParams) (<-chan ChatCompletionChunk, <-chan error)
}

// ModelLister is implemented by providers that can list models (e.g. Ollama).
type ModelLister interface {
	Provider
	ListModels(ctx Context) (*ModelsResponse, error)
}

// ChatCompletion represents a chat completion response in OpenAI format.
type ChatCompletion struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// ChatCompletionChunk represents a streaming chunk in OpenAI format.
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
	Usage   *Usage        `json:"usage,omitempty"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

// ChunkChoice represents a choice in a streaming chunk.
type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChunkDelta `json:"delta"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

// ChunkDelta represents the delta content in a streaming chunk.
type ChunkDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Reasoning *Reasoning `json:"reasoning,omitempty"`
}

// CompletionParams represents normalized parameters for chat completion requests.
type CompletionParams struct {
	Model      string   `json:"model"`
	Messages   []Message `json:"messages"`
	Tools      []Tool   `json:"tools,omitempty"`
	ToolChoice any      `json:"tool_choice,omitempty"`
}

// Message represents a chat message in OpenAI format.
type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolName   string     `json:"tool_name,omitempty"`
}

// ContentString extracts string content from a message.
func (m *Message) ContentString() string {
	if s, ok := m.Content.(string); ok {
		return s
	}
	return ""
}

// Function represents a function definition for tool calling.
type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// FunctionCall represents the function being called.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool represents a tool/function that can be called.
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// ToolCall represents a tool call made by the assistant.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// Reasoning represents extended thinking/reasoning content.
type Reasoning struct {
	Content string `json:"content,omitempty"`
}

// Model represents a model from a list models API.
type Model struct {
	ID string `json:"id"`
}

// ModelsResponse represents a list models response.
type ModelsResponse struct {
	Data []Model `json:"data"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
