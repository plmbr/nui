// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	anthropicDefaultBaseURL = "https://api.anthropic.com"
	anthropicAPIVersion     = "2023-06-01"
	anthropicDefaultMaxTok  = 4096
)

type anthropicProvider struct {
	apiKey  string
	baseURL string
}

func newAnthropicProvider(apiKey, baseURL string) (*anthropicProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("anthropic: api key required")
	}
	if baseURL == "" {
		baseURL = anthropicDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &anthropicProvider{apiKey: apiKey, baseURL: baseURL}, nil
}

func (p *anthropicProvider) Name() string { return "anthropic" }

func (p *anthropicProvider) Completion(ctx context.Context, params CompletionParams) (*ChatCompletion, error) {
	body := p.buildRequest(params, false)
	resp, err := postJSON(ctx, p.baseURL+"/v1/messages", p.headers(), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkHTTPError("anthropic", resp); err != nil {
		return nil, err
	}
	var raw anthropicMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return anthropicToCompletion(&raw), nil
}

func (p *anthropicProvider) CompletionStream(ctx context.Context, params CompletionParams) (<-chan ChatCompletionChunk, <-chan error) {
	chunks := make(chan ChatCompletionChunk)
	errs := make(chan error, 1)
	emitStream(ctx, chunks, errs, func() error {
		body := p.buildRequest(params, true)
		resp, err := postJSON(ctx, p.baseURL+"/v1/messages", p.headers(), body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if err := checkHTTPError("anthropic", resp); err != nil {
			return err
		}
		state := &anthropicStreamState{currentToolIdx: -1}
		return streamSSE(ctx, resp.Body, func(data []byte) error {
			var event anthropicStreamEvent
			if err := json.Unmarshal(data, &event); err != nil {
				return nil
			}
			chunk := state.handleEvent(event)
			if chunk == nil {
				return nil
			}
			select {
			case chunks <- *chunk:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	})
	return chunks, errs
}

func (p *anthropicProvider) headers() map[string]string {
	return map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": anthropicAPIVersion,
		"Accept":            "text/event-stream",
	}
}

func (p *anthropicProvider) buildRequest(params CompletionParams, stream bool) map[string]any {
	msgs, system := anthropicMessages(params.Messages)
	req := map[string]any{
		"model":      params.Model,
		"max_tokens": anthropicDefaultMaxTok,
		"messages":   msgs,
		"stream":     stream,
	}
	if system != "" {
		req["system"] = system
	}
	if len(params.Tools) > 0 {
		req["tools"] = anthropicTools(params.Tools)
		choice := "auto"
		if params.ToolChoice != nil {
			if s, ok := params.ToolChoice.(string); ok {
				choice = s
			}
		}
		switch choice {
		case "none":
			req["tool_choice"] = map[string]any{"type": "none"}
		case "required", "any":
			req["tool_choice"] = map[string]any{"type": "any"}
		default:
			req["tool_choice"] = map[string]any{"type": "auto"}
		}
	}
	return req
}

type anthropicMessageResponse struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func anthropicToCompletion(raw *anthropicMessageResponse) *ChatCompletion {
	var content string
	var toolCalls []ToolCall
	for _, block := range raw.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			args := string(block.Input)
			if args == "" || args == "null" {
				args = "{}"
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: args,
				},
			})
		}
	}
	finish := FinishReasonStop
	if raw.StopReason == "tool_use" {
		finish = FinishReasonToolCalls
	} else if raw.StopReason == "max_tokens" {
		finish = FinishReasonLength
	}
	return &ChatCompletion{
		ID:    raw.ID,
		Model: raw.Model,
		Choices: []Choice{{
			Message: Message{
				Role:      RoleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finish,
		}},
		Usage: &Usage{
			PromptTokens:     raw.Usage.InputTokens,
			CompletionTokens: raw.Usage.OutputTokens,
			TotalTokens:      raw.Usage.InputTokens + raw.Usage.OutputTokens,
		},
	}
}

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type         string `json:"type"`
		Text         string `json:"text"`
		PartialJSON  string `json:"partial_json"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
	} `json:"delta"`
	Message struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamState struct {
	messageID      string
	model          string
	inputUsage     int
	toolCalls      []ToolCall
	currentToolIdx int
}

func (s *anthropicStreamState) handleEvent(event anthropicStreamEvent) *ChatCompletionChunk {
	switch event.Type {
	case "message_start":
		s.messageID = event.Message.ID
		s.model = event.Message.Model
		s.inputUsage = event.Message.Usage.InputTokens
		chunk := s.chunk(ChunkDelta{Role: RoleAssistant})
		return &chunk
	case "content_block_start":
		if event.ContentBlock.Type == "tool_use" {
			s.currentToolIdx++
			tc := ToolCall{
				ID:   event.ContentBlock.ID,
				Type: "function",
				Function: FunctionCall{
					Name: event.ContentBlock.Name,
				},
			}
			s.toolCalls = append(s.toolCalls, tc)
		}
		return nil
	case "content_block_delta":
		switch event.Delta.Type {
		case "text_delta":
			chunk := s.chunk(ChunkDelta{Content: event.Delta.Text})
			return &chunk
		case "input_json_delta":
			if s.currentToolIdx < 0 || s.currentToolIdx >= len(s.toolCalls) {
				return nil
			}
			s.toolCalls[s.currentToolIdx].Function.Arguments += event.Delta.PartialJSON
			tc := s.toolCalls[s.currentToolIdx]
			chunk := s.chunk(ChunkDelta{ToolCalls: []ToolCall{tc}})
			return &chunk
		}
	case "message_delta":
		finish := FinishReasonStop
		if event.Delta.StopReason == "tool_use" {
			finish = FinishReasonToolCalls
		} else if event.Delta.StopReason == "max_tokens" {
			finish = FinishReasonLength
		}
		chunk := s.chunk(ChunkDelta{})
		chunk.Choices[0].FinishReason = finish
		chunk.Usage = &Usage{
			PromptTokens:     s.inputUsage,
			CompletionTokens: event.Usage.OutputTokens,
			TotalTokens:      s.inputUsage + event.Usage.OutputTokens,
		}
		return &chunk
	}
	return nil
}

func (s *anthropicStreamState) chunk(delta ChunkDelta) ChatCompletionChunk {
	return ChatCompletionChunk{
		ID:    s.messageID,
		Model: s.model,
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: delta,
		}},
	}
}

func anthropicMessages(messages []Message) ([]map[string]any, string) {
	var systemParts []string
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			systemParts = append(systemParts, msg.ContentString())
		case RoleUser:
			out = append(out, map[string]any{
				"role":    "user",
				"content": msg.ContentString(),
			})
		case RoleAssistant:
			blocks := make([]map[string]any, 0)
			if c := msg.ContentString(); c != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": c})
			}
			for _, tc := range msg.ToolCalls {
				var input map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
				blocks = append(blocks, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": input,
				})
			}
			out = append(out, map[string]any{"role": "assistant", "content": blocks})
		case RoleTool:
			out = append(out, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": msg.ToolCallID,
					"content":     msg.ContentString(),
				}},
			})
		}
	}
	return out, strings.Join(systemParts, "\n")
}

func anthropicTools(tools []Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		schema := map[string]any{"type": "object"}
		if t.Function.Parameters != nil {
			if props, ok := t.Function.Parameters["properties"]; ok {
				schema["properties"] = props
			}
			if req, ok := t.Function.Parameters["required"]; ok {
				schema["required"] = req
			}
		}
		out = append(out, map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": schema,
		})
	}
	return out
}
