// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const openAIDefaultBaseURL = "https://api.openai.com/v1"

type openAIProvider struct {
	name    string
	apiKey  string
	baseURL string
}

func newOpenAIProvider(name, apiKey, baseURL string) (*openAIProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("%s: api key required", name)
	}
	if baseURL == "" {
		baseURL = openAIDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &openAIProvider{name: name, apiKey: apiKey, baseURL: baseURL}, nil
}

func (p *openAIProvider) Name() string { return p.name }

func (p *openAIProvider) Completion(ctx context.Context, params CompletionParams) (*ChatCompletion, error) {
	body := p.buildRequest(params, false)
	resp, err := postJSON(ctx, p.baseURL+"/chat/completions", p.headers(), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkHTTPError(p.name, resp); err != nil {
		return nil, err
	}
	var out ChatCompletion
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *openAIProvider) CompletionStream(ctx context.Context, params CompletionParams) (<-chan ChatCompletionChunk, <-chan error) {
	chunks := make(chan ChatCompletionChunk)
	errs := make(chan error, 1)
	emitStream(ctx, chunks, errs, func() error {
		body := p.buildRequest(params, true)
		resp, err := postJSON(ctx, p.baseURL+"/chat/completions", p.headers(), body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if err := checkHTTPError(p.name, resp); err != nil {
			return err
		}
		return streamSSE(ctx, resp.Body, func(data []byte) error {
			var chunk ChatCompletionChunk
			if err := json.Unmarshal(data, &chunk); err != nil {
				return nil
			}
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

func (p *openAIProvider) headers() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + p.apiKey,
		"Accept":        "text/event-stream",
	}
}

func (p *openAIProvider) buildRequest(params CompletionParams, stream bool) map[string]any {
	req := map[string]any{
		"model":    params.Model,
		"messages": openAIMessages(params.Messages),
		"stream":   stream,
	}
	if len(params.Tools) > 0 {
		req["tools"] = openAITools(params.Tools)
		if params.ToolChoice != nil {
			req["tool_choice"] = params.ToolChoice
		} else {
			req["tool_choice"] = "auto"
		}
	}
	return req
}

func openAIMessages(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		m := map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}
		if len(msg.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				calls = append(calls, map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			m["tool_calls"] = calls
		}
		out = append(out, m)
	}
	return out
}

func openAITools(tools []Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		params := t.Function.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Function.Name,
				"description": t.Function.Description,
				"parameters":  params,
			},
		})
	}
	return out
}
