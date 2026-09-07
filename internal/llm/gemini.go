// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const geminiDefaultBaseURL = "https://generativelanguage.googleapis.com"

// Dummy signature accepted by Gemini when transferring history without a real signature.
// Prefer preserving the real thoughtSignature from model responses.
const geminiSkipThoughtSignature = "skip_thought_signature_validator"

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
				"parameters":   sanitizeGeminiSchema(params),
			})
		}
		req["tools"] = []map[string]any{{
			"functionDeclarations": tools,
		}}
	}
	return req
}

// geminiSchemaAllowedKeys is the Schema subset accepted by Gemini functionDeclarations.parameters.
// See https://ai.google.dev/api/caching#Schema — additionalProperties and most JSON Schema
// draft keywords are rejected with INVALID_ARGUMENT.
var geminiSchemaAllowedKeys = map[string]struct{}{
	"type":             {},
	"format":           {},
	"description":      {},
	"nullable":         {},
	"enum":             {},
	"items":            {},
	"properties":       {},
	"required":         {},
	"anyOf":            {},
	"minItems":         {},
	"maxItems":         {},
	"minLength":        {},
	"maxLength":        {},
	"minimum":          {},
	"maximum":          {},
	"propertyOrdering": {},
}

// sanitizeGeminiSchema recursively strips JSON Schema fields Gemini rejects (notably
// additionalProperties from MCP/Zod schemas) and normalizes common shapes.
func sanitizeGeminiSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	out := sanitizeGeminiSchemaValue(schema)
	if m, ok := out.(map[string]any); ok && m != nil {
		return m
	}
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func sanitizeGeminiSchemaValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return sanitizeGeminiSchemaObject(t)
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, sanitizeGeminiSchemaValue(item))
		}
		return out
	default:
		return v
	}
}

func sanitizeGeminiSchemaObject(schema map[string]any) map[string]any {
	out := make(map[string]any, len(schema))
	for key, value := range schema {
		if _, ok := geminiSchemaAllowedKeys[key]; !ok {
			continue
		}
		switch key {
		case "properties":
			props, ok := value.(map[string]any)
			if !ok {
				continue
			}
			cleaned := make(map[string]any, len(props))
			for name, prop := range props {
				cleaned[name] = sanitizeGeminiSchemaValue(prop)
			}
			out[key] = cleaned
		case "items":
			out[key] = sanitizeGeminiSchemaValue(value)
		case "anyOf":
			list, ok := value.([]any)
			if !ok {
				continue
			}
			cleaned := make([]any, 0, len(list))
			for _, item := range list {
				cleaned = append(cleaned, sanitizeGeminiSchemaValue(item))
			}
			if len(cleaned) > 0 {
				out[key] = cleaned
			}
		case "required":
			out[key] = sanitizeStringList(value)
		case "enum":
			if list, ok := value.([]any); ok {
				out[key] = list
			}
		case "type":
			out[key] = normalizeGeminiType(value, out)
		default:
			out[key] = value
		}
	}
	if _, ok := out["type"]; !ok {
		if _, hasProps := out["properties"]; hasProps {
			out["type"] = "object"
		}
	}
	return out
}

func normalizeGeminiType(value any, out map[string]any) any {
	switch t := value.(type) {
	case string:
		return t
	case []any:
		nonNull, nullable := splitNullType(t)
		if nullable {
			out["nullable"] = true
		}
		if nonNull != "" {
			return nonNull
		}
		return "string"
	default:
		return value
	}
}

func splitNullType(types []any) (nonNull string, nullable bool) {
	for _, item := range types {
		s, ok := item.(string)
		if !ok {
			continue
		}
		if s == "null" {
			nullable = true
			continue
		}
		if nonNull == "" {
			nonNull = s
		}
	}
	return nonNull, nullable
}

func sanitizeStringList(value any) []string {
	switch t := value.(type) {
	case []string:
		return append([]string(nil), t...)
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

type geminiPart struct {
	Text             string `json:"text"`
	Thought          bool   `json:"thought"`
	ThoughtSignature string `json:"thoughtSignature"`
	FunctionCall     *struct {
		Name string         `json:"name"`
		Args map[string]any `json:"args"`
	} `json:"functionCall"`
}

func geminiToCompletion(raw *geminiGenerateResponse, model string) *ChatCompletion {
	var content string
	var toolCalls []ToolCall
	finish := FinishReasonStop
	if len(raw.Candidates) > 0 {
		c := raw.Candidates[0]
		fcIndex := 0
		for _, part := range c.Content.Parts {
			if part.FunctionCall != nil {
				args, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, ToolCall{
					ID:    fmt.Sprintf("call_gemini_%d", fcIndex),
					Type:  "function",
					Index: fcIndex,
					Function: FunctionCall{
						Name:      part.FunctionCall.Name,
						Arguments: string(args),
					},
					ThoughtSignature: strings.TrimSpace(part.ThoughtSignature),
				})
				fcIndex++
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
	fcIndex      int
	sawToolCall  bool
}

func (s *geminiStreamState) process(raw *geminiGenerateResponse) []ChatCompletionChunk {
	var out []ChatCompletionChunk
	if len(raw.Candidates) == 0 {
		return out
	}
	c := raw.Candidates[0]
	if c.FinishReason != "" {
		s.finishReason = c.FinishReason
	}
	for _, part := range c.Content.Parts {
		if part.FunctionCall != nil {
			s.sawToolCall = true
			args, _ := json.Marshal(part.FunctionCall.Args)
			idx := s.fcIndex
			s.fcIndex++
			tc := ToolCall{
				ID:    fmt.Sprintf("call_gemini_%d", idx),
				Type:  "function",
				Index: idx,
				Function: FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				},
				ThoughtSignature: strings.TrimSpace(part.ThoughtSignature),
			}
			out = append(out, ChatCompletionChunk{
				Model: s.model,
				Choices: []ChunkChoice{{
					Delta: ChunkDelta{ToolCalls: []ToolCall{tc}},
				}},
			})
		} else if sig := strings.TrimSpace(part.ThoughtSignature); sig != "" && part.Text == "" {
			// Signature may arrive in a trailing empty part during streaming.
			out = append(out, ChatCompletionChunk{
				Model: s.model,
				Choices: []ChunkChoice{{
					Delta: ChunkDelta{ToolCalls: []ToolCall{{
						Index:            signatureTargetIndex(s.fcIndex),
						ThoughtSignature: sig,
					}}},
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
		} else if s.sawToolCall {
			finish = FinishReasonToolCalls
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
			for i, tc := range msg.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				if args == nil {
					args = map[string]any{}
				}
				part := map[string]any{
					"functionCall": map[string]any{
						"name": tc.Function.Name,
						"args": args,
					},
				}
				sig := strings.TrimSpace(tc.ThoughtSignature)
				if sig == "" && i == 0 {
					// Gemini 3 requires a signature on the first functionCall of each step.
					sig = geminiSkipThoughtSignature
				}
				if sig != "" {
					part["thoughtSignature"] = sig
				}
				parts = append(parts, part)
			}
			if len(parts) == 0 {
				parts = append(parts, map[string]any{"text": ""})
			}
			out = append(out, map[string]any{"role": "model", "parts": parts})
		case RoleTool:
			name := strings.TrimSpace(msg.ToolName)
			if name == "" {
				name = strings.TrimSpace(msg.ToolCallID)
			}
			response := geminiToolResponse(msg.ContentString())
			out = append(out, map[string]any{
				"role": "user",
				"parts": []map[string]any{{
					"functionResponse": map[string]any{
						"name":     name,
						"response": response,
					},
				}},
			})
		}
	}
	return out, strings.Join(systemParts, "\n")
}

func geminiToolResponse(content string) map[string]any {
	content = strings.TrimSpace(content)
	if content == "" {
		return map[string]any{"result": ""}
	}
	var asMap map[string]any
	if err := json.Unmarshal([]byte(content), &asMap); err == nil && asMap != nil {
		return asMap
	}
	var asAny any
	if err := json.Unmarshal([]byte(content), &asAny); err == nil {
		return map[string]any{"result": asAny}
	}
	return map[string]any{"result": content}
}

func signatureTargetIndex(fcIndex int) int {
	if fcIndex <= 0 {
		return 0
	}
	return fcIndex - 1
}
