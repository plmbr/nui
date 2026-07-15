// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const geminiDefaultBaseURL = "https://generativelanguage.googleapis.com"

type geminiProvider struct {
	apiKey  string
	baseURL string
}

func newGeminiProvider(apiKey, baseURL string) (*geminiProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("gemini: api key required")
	}
	if baseURL == "" {
		baseURL = geminiDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &geminiProvider{apiKey: apiKey, baseURL: baseURL}, nil
}

func (p *geminiProvider) Name() string { return "gemini" }

func (p *geminiProvider) Completion(ctx context.Context, params CompletionParams) (*ChatCompletion, error) {
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, params.Model, p.apiKey)
	body := p.buildRequest(params)
	resp, err := postJSON(ctx, url, nil, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkHTTPError("gemini", resp); err != nil {
		return nil, err
	}
	var raw geminiGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return geminiToCompletion(&raw, params.Model), nil
}

func (p *geminiProvider) CompletionStream(ctx context.Context, params CompletionParams) (<-chan ChatCompletionChunk, <-chan error) {
	chunks := make(chan ChatCompletionChunk)
	errs := make(chan error, 1)
	emitStream(ctx, chunks, errs, func() error {
		url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, params.Model, p.apiKey)
		body := p.buildRequest(params)
		resp, err := postJSON(ctx, url, map[string]string{"Accept": "text/event-stream"}, body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if err := checkHTTPError("gemini", resp); err != nil {
			return err
		}
		state := &geminiStreamState{model: params.Model}
		return streamSSE(ctx, resp.Body, func(data []byte) error {
			var raw geminiGenerateResponse
			if err := json.Unmarshal(data, &raw); err != nil {
				return nil
			}
			for _, chunk := range state.process(&raw) {
				select {
				case chunks <- chunk:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	})
	return chunks, errs
}

func (p *geminiProvider) buildRequest(params CompletionParams) map[string]any {
	contents, system := geminiContents(params.Messages)
	req := map[string]any{
		"contents": contents,
	}
	if system != "" {
		req["systemInstruction"] = map[string]any{
			"parts": []map[string]any{{"text": system}},
		}
	}
	if len(params.Tools) > 0 {
		tools := make([]map[string]any, 0, len(params.Tools))
		for _, t := range params.Tools {
			params := t.Function.Parameters
			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			tools = append(tools, map[string]any{
				"name":        t.Function.Name,
				"description": t.Function.Description,
				"parameters":  params,
			})
		}
		req["tools"] = []map[string]any{{
			"functionDeclarations": tools,
		}}
	}
	return req
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text         string `json:"text"`
				Thought      bool   `json:"thought"`
				FunctionCall *struct {
					Name string         `json:"name"`
					Args map[string]any `json:"args"`
				} `json:"functionCall"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func geminiToCompletion(raw *geminiGenerateResponse, model string) *ChatCompletion {
	var content string
	var toolCalls []ToolCall
	finish := FinishReasonStop
	if len(raw.Candidates) > 0 {
		c := raw.Candidates[0]
		for _, part := range c.Content.Parts {
			if part.FunctionCall != nil {
				args, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, ToolCall{
					ID:   "call_gemini",
					Type: "function",
					Function: FunctionCall{
						Name:      part.FunctionCall.Name,
						Arguments: string(args),
					},
				})
			} else if part.Text != "" && !part.Thought {
				content += part.Text
			}
		}
		if c.FinishReason == "MAX_TOKENS" {
			finish = FinishReasonLength
		} else if len(toolCalls) > 0 {
			finish = FinishReasonToolCalls
		}
	}
	usage := (*Usage)(nil)
	if raw.UsageMetadata != nil {
		usage = &Usage{
			PromptTokens:     raw.UsageMetadata.PromptTokenCount,
			CompletionTokens: raw.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      raw.UsageMetadata.PromptTokenCount + raw.UsageMetadata.CandidatesTokenCount,
		}
	}
	return &ChatCompletion{
		Model: model,
		Choices: []Choice{{
			Message: Message{
				Role:      RoleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finish,
		}},
		Usage: usage,
	}
}

type geminiStreamState struct {
	model        string
	finishReason string
}

func (s *geminiStreamState) process(raw *geminiGenerateResponse) []ChatCompletionChunk {
	var out []ChatCompletionChunk
	if raw.UsageMetadata != nil {
		_ = raw.UsageMetadata
	}
	if len(raw.Candidates) == 0 {
		return out
	}
	c := raw.Candidates[0]
	if c.FinishReason != "" {
		s.finishReason = c.FinishReason
	}
	for _, part := range c.Content.Parts {
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			tc := ToolCall{
				ID:   "call_gemini",
				Type: "function",
				Function: FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				},
			}
			out = append(out, ChatCompletionChunk{
				Model: s.model,
				Choices: []ChunkChoice{{
					Delta: ChunkDelta{ToolCalls: []ToolCall{tc}},
				}},
			})
		} else if part.Text != "" && !part.Thought {
			out = append(out, ChatCompletionChunk{
				Model: s.model,
				Choices: []ChunkChoice{{
					Delta: ChunkDelta{Content: part.Text},
				}},
			})
		}
	}
	if s.finishReason != "" {
		finish := FinishReasonStop
		if s.finishReason == "MAX_TOKENS" {
			finish = FinishReasonLength
		}
		out = append(out, ChatCompletionChunk{
			Model: s.model,
			Choices: []ChunkChoice{{
				FinishReason: finish,
			}},
		})
	}
	return out
}

func geminiContents(messages []Message) ([]map[string]any, string) {
	var systemParts []string
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			systemParts = append(systemParts, msg.ContentString())
		case RoleUser:
			out = append(out, map[string]any{
				"role":  "user",
				"parts": []map[string]any{{"text": msg.ContentString()}},
			})
		case RoleAssistant:
			parts := make([]map[string]any, 0)
			if c := msg.ContentString(); c != "" {
				parts = append(parts, map[string]any{"text": c})
			}
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				parts = append(parts, map[string]any{
					"functionCall": map[string]any{
						"name": tc.Function.Name,
						"args": args,
					},
				})
			}
			out = append(out, map[string]any{"role": "model", "parts": parts})
		case RoleTool:
			var response map[string]any
			_ = json.Unmarshal([]byte(msg.ContentString()), &response)
			out = append(out, map[string]any{
				"role": "user",
				"parts": []map[string]any{{
					"functionResponse": map[string]any{
						"name":     msg.ToolCallID,
						"response": response,
					},
				}},
			})
		}
	}
	return out, strings.Join(systemParts, "\n")
}
