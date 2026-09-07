// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const ollamaDefaultBaseURL = "http://localhost:11434"

type ollamaProvider struct {
	baseURL string
}

func newOllamaProvider(baseURL string) (*ollamaProvider, error) {
	if baseURL == "" {
		baseURL = ollamaDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("ollama: invalid base URL: %w", err)
	}
	return &ollamaProvider{baseURL: baseURL}, nil
}

func (p *ollamaProvider) Name() string { return "ollama" }

func (p *ollamaProvider) Completion(ctx context.Context, params CompletionParams) (*ChatCompletion, error) {
	body := p.buildRequest(params, false)
	resp, err := postJSON(ctx, p.baseURL+"/api/chat", nil, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkHTTPError("ollama", resp); err != nil {
		return nil, err
	}
	var raw ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return ollamaToCompletion(&raw), nil
}

func (p *ollamaProvider) CompletionStream(ctx context.Context, params CompletionParams) (<-chan ChatCompletionChunk, <-chan error) {
	chunks := make(chan ChatCompletionChunk)
	errs := make(chan error, 1)
	emitStream(ctx, chunks, errs, func() error {
		body := p.buildRequest(params, true)
		resp, err := postJSON(ctx, p.baseURL+"/api/chat", nil, body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if err := checkHTTPError("ollama", resp); err != nil {
			return err
		}
		state := &ollamaStreamState{created: time.Now().Unix(), id: "ollama-" + params.Model}
		return streamNDJSON(ctx, resp.Body, func(data []byte) error {
			var raw ollamaChatResponse
			if err := json.Unmarshal(data, &raw); err != nil {
				return nil
			}
			chunk := state.handle(&raw)
			select {
			case chunks <- chunk:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	})
	return chunks, errs
}

func (p *ollamaProvider) ListModels(ctx context.Context) (*ModelsResponse, error) {
	resp, err := getJSON(ctx, p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkHTTPError("ollama", resp); err != nil {
		return nil, err
	}
	var raw struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(raw.Models))
	for _, m := range raw.Models {
		if id := strings.TrimSpace(m.Name); id != "" {
			out = append(out, Model{ID: id})
		}
	}
	return &ModelsResponse{Data: out}, nil
}

func (p *ollamaProvider) buildRequest(params CompletionParams, stream bool) map[string]any {
	req := map[string]any{
		"model":    params.Model,
		"messages": ollamaMessages(params.Messages),
		"stream":   stream,
		"options":  map[string]any{"num_ctx": 32000},
	}
	if len(params.Tools) > 0 {
		tools := make([]map[string]any, 0, len(params.Tools))
		for _, t := range params.Tools {
			params := t.Function.Parameters
			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Function.Name,
					"description": t.Function.Description,
					"parameters":  params,
				},
			})
		}
		req["tools"] = tools
	}
	return req
}

type ollamaChatResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		Thinking  string           `json:"thinking"`
		ToolCalls []ollamaToolCall `json:"tool_calls"`
	} `json:"message"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
}

// ollamaToolCall matches current Ollama /api/chat tool_calls (id + function.index + object args).
type ollamaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Index     int             `json:"index"`
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

func ollamaToCompletion(raw *ollamaChatResponse) *ChatCompletion {
	toolCalls := ollamaToolCalls(raw.Message.ToolCalls)
	finish := FinishReasonStop
	if len(toolCalls) > 0 {
		finish = FinishReasonToolCalls
	}
	return &ChatCompletion{
		Model: raw.Model,
		Choices: []Choice{{
			Message: Message{
				Role:      RoleAssistant,
				Content:   raw.Message.Content,
				ToolCalls: toolCalls,
			},
			FinishReason: finish,
		}},
	}
}

type ollamaStreamState struct {
	id      string
	model   string
	created int64
	sawTools bool
}

func (s *ollamaStreamState) handle(raw *ollamaChatResponse) ChatCompletionChunk {
	if s.model == "" && raw.Model != "" {
		s.model = raw.Model
	}
	delta := ChunkDelta{Content: raw.Message.Content}
	if raw.Message.Thinking != "" {
		delta.Reasoning = &Reasoning{Content: raw.Message.Thinking}
	}
	if len(raw.Message.ToolCalls) > 0 {
		s.sawTools = true
		delta.ToolCalls = ollamaToolCalls(raw.Message.ToolCalls)
	}
	chunk := ChatCompletionChunk{
		ID:      s.id,
		Created: s.created,
		Model:   s.model,
		Choices: []ChunkChoice{{Index: 0, Delta: delta}},
	}
	if raw.Done {
		finish := FinishReasonStop
		if s.sawTools || len(delta.ToolCalls) > 0 {
			// Ollama often ends with done_reason=stop even after emitting tool_calls.
			finish = FinishReasonToolCalls
		} else if raw.DoneReason == "length" {
			finish = FinishReasonLength
		}
		chunk.Choices[0].FinishReason = finish
	}
	return chunk
}

func ollamaToolCalls(calls []ollamaToolCall) []ToolCall {
	out := make([]ToolCall, 0, len(calls))
	for i, tc := range calls {
		idx := tc.Function.Index
		if idx == 0 && i > 0 {
			idx = i
		}
		id := strings.TrimSpace(tc.ID)
		if id == "" {
			id = fmt.Sprintf("call_%d", idx)
		}
		callType := strings.TrimSpace(tc.Type)
		if callType == "" {
			callType = "function"
		}
		out = append(out, ToolCall{
			ID:    id,
			Type:  callType,
			Index: idx,
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: ollamaArgumentsJSON(tc.Function.Arguments),
			},
		})
	}
	return out
}

func ollamaArgumentsJSON(raw json.RawMessage) string {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return "{}"
	}
	// Object/array args — keep as JSON object string.
	if raw[0] == '{' || raw[0] == '[' {
		if json.Valid(raw) {
			return string(raw)
		}
		return "{}"
	}
	// JSON-encoded string containing JSON object.
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" {
			return "{}"
		}
		if json.Valid([]byte(asString)) {
			return asString
		}
		return "{}"
	}
	return "{}"
}

func ollamaMessages(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == RoleSystem {
			out = append(out, map[string]any{"role": "system", "content": msg.ContentString()})
			continue
		}
		if msg.Role == RoleTool {
			m := map[string]any{
				"role":    "tool",
				"content": msg.ContentString(),
			}
			if name := strings.TrimSpace(msg.ToolName); name != "" {
				m["tool_name"] = name
			}
			out = append(out, m)
			continue
		}
		m := map[string]any{
			"role":    msg.Role,
			"content": msg.ContentString(),
		}
		if len(msg.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				var args any
				rawArgs := strings.TrimSpace(tc.Function.Arguments)
				if rawArgs == "" {
					args = map[string]any{}
				} else if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
					args = map[string]any{}
				}
				callType := strings.TrimSpace(tc.Type)
				if callType == "" {
					callType = "function"
				}
				fn := map[string]any{
					"name":      tc.Function.Name,
					"arguments": args,
				}
				// Preserve index when present so multi-call rounds round-trip cleanly.
				if tc.Index > 0 || tc.ID != "" {
					fn["index"] = tc.Index
				}
				call := map[string]any{
					"type":     callType,
					"function": fn,
				}
				if id := strings.TrimSpace(tc.ID); id != "" {
					call["id"] = id
				}
				calls = append(calls, call)
			}
			m["tool_calls"] = calls
		}
		out = append(out, m)
	}
	return out
}
