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
		Role      string `json:"role"`
		Content   string `json:"content"`
		Thinking  string `json:"thinking"`
		ToolCalls []struct {
			Function struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"message"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
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
		if len(delta.ToolCalls) > 0 {
			finish = FinishReasonToolCalls
		} else if raw.DoneReason == "length" {
			finish = FinishReasonLength
		}
		chunk.Choices[0].FinishReason = finish
	}
	return chunk
}

func ollamaToolCalls(calls []struct {
	Function struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"function"`
}) []ToolCall {
	out := make([]ToolCall, 0, len(calls))
	for i, tc := range calls {
		args, _ := json.Marshal(tc.Function.Arguments)
		out = append(out, ToolCall{
			ID:   fmt.Sprintf("call_%d", i),
			Type: "function",
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: string(args),
			},
		})
	}
	return out
}

func ollamaMessages(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == RoleSystem {
			out = append(out, map[string]any{"role": "system", "content": msg.ContentString()})
			continue
		}
		m := map[string]any{
			"role":    msg.Role,
			"content": msg.ContentString(),
		}
		if len(msg.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				calls = append(calls, map[string]any{
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": args,
					},
				})
			}
			m["tool_calls"] = calls
		}
		out = append(out, m)
	}
	return out
}
