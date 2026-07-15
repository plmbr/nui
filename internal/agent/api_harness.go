// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"loop/internal/hitl"
	"loop/internal/model"
	"loop/internal/viz"
)

const maxAPIToolIterations = 32

// APIHarnessAgent runs an in-process LLM conversation via any-llm-go.
type APIHarnessAgent struct {
	Harness model.ADLHarness
}

func (a *APIHarnessAgent) Name() string { return "api" }

func (a *APIHarnessAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	provider, err := NewLLMProvider(a.Harness, req.Env)
	if err != nil {
		return err
	}

	mcpClient := NewSessionMCP()
	defer mcpClient.Close()
	if len(req.MCPServers) > 0 {
		if err := mcpClient.ConnectServers(ctx, req.MCPServers); err != nil {
			return fmt.Errorf("connect mcp servers: %w", err)
		}
	}

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = resolveAPIModel(req, a.Harness)
	}
	if strings.TrimSpace(a.Harness.Provider) == "ollama" {
		var err error
		modelName, err = ensureOllamaModel(ctx, provider, modelName)
		if err != nil {
			return err
		}
	}
	if modelName == "" {
		return fmt.Errorf("api harness: model is required")
	}

	modelCandidates := []string{modelName}
	if strings.TrimSpace(a.Harness.Provider) == "anthropic" {
		if candidates := resolveAnthropicModelCandidates(req, a.Harness, modelName); len(candidates) > 0 {
			modelCandidates = candidates
		}
	}
	candidateIdx := 0
	modelName = modelCandidates[0]

	messages := buildAPIMessages(req)
	tools := sessionToolsToAnyLLMTools(mcpClient.Tools())

	for iteration := 0; iteration < maxAPIToolIterations; iteration++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		assistant, streamedText, err := a.streamCompletion(ctx, provider, modelName, normalizeAPIMessages(messages), tools, events)
		if err != nil {
			if errors.Is(err, anyllm.ErrModelNotFound) && candidateIdx < len(modelCandidates)-1 {
				candidateIdx++
				modelName = modelCandidates[candidateIdx]
				continue
			}
			return err
		}
		if len(assistant.ToolCalls) == 0 {
			if streamedText == "" && assistant.Content != nil {
				if text, ok := assistant.Content.(string); ok {
					streamedText = text
				}
			}
			_ = streamedText
			events <- Event{Type: EventDone}
			return nil
		}

		assistant.ToolCalls = filterExecutableToolCalls(normalizeAPIToolCalls(assistant.ToolCalls))
		if len(assistant.ToolCalls) == 0 {
			events <- Event{Type: EventDone}
			return nil
		}

		messages = append(messages, assistant)
		executedViz := false
		for _, tc := range assistant.ToolCalls {
			toolName := tc.Function.Name
			argsJSON := tc.Function.Arguments
			var args map[string]any
			_ = json.Unmarshal([]byte(argsJSON), &args)

			approved, err := a.approveTool(ctx, req, events, toolName, args)
			if err != nil {
				return err
			}
			if !approved {
				resultText := "tool call denied by user"
				events <- Event{Type: EventToolCallResult, ToolCallID: tc.ID, ToolName: toolName, Content: resultText}
				messages = append(messages, anyllm.Message{
					Role:       anyllm.RoleTool,
					ToolCallID: tc.ID,
					Content:    resultText,
				})
				continue
			}

			resultText, err := mcpClient.CallTool(ctx, toolName, args)
			if err != nil {
				resultText = "error: " + err.Error()
			}
			events <- Event{Type: EventToolCallResult, ToolCallID: tc.ID, ToolName: toolName, Content: resultText}
			if viz.IsVisualizationTool(toolName) && err == nil {
				executedViz = true
			}
			messages = append(messages, anyllm.Message{
				Role:       anyllm.RoleTool,
				ToolCallID: tc.ID,
				Content:    resultText,
			})
		}
		if executedViz {
			events <- Event{Type: EventDone}
			return nil
		}
	}
	return fmt.Errorf("api harness: exceeded max tool iterations (%d)", maxAPIToolIterations)
}

func (a *APIHarnessAgent) streamCompletion(
	ctx context.Context,
	provider anyllm.Provider,
	modelName string,
	messages []anyllm.Message,
	tools []anyllm.Tool,
	events chan<- Event,
) (anyllm.Message, string, error) {
	params := anyllm.CompletionParams{
		Model:    modelName,
		Messages: messages,
	}
	if len(tools) > 0 {
		params.Tools = tools
		params.ToolChoice = "auto"
	}

	chunks, errs := provider.CompletionStream(ctx, params)
	var streamed strings.Builder
	var bufferedText strings.Builder
	bufferingText := false
	toolCalls := map[int]*accumulatedToolCall{}
	finishReason := ""

	emitText := func(delta string) {
		if delta == "" {
			return
		}
		streamed.WriteString(delta)
		events <- Event{Type: EventText, Content: delta}
	}

	for chunk := range chunks {
		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]
		if choice.Delta.Content != "" {
			probe := streamed.String() + bufferedText.String() + choice.Delta.Content
			if bufferingText || shouldBufferTextToolStream(probe) {
				bufferingText = true
				bufferedText.WriteString(choice.Delta.Content)
			} else {
				emitText(choice.Delta.Content)
			}
		}
		for i, tc := range choice.Delta.ToolCalls {
			acc, ok := toolCalls[i]
			if !ok {
				acc = &accumulatedToolCall{id: tc.ID, name: tc.Function.Name}
				toolCalls[i] = acc
				if tc.ID != "" || tc.Function.Name != "" {
					acc.started = true
					events <- Event{Type: EventToolCallStart, ToolCallID: tc.ID, ToolName: tc.Function.Name}
				}
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Function.Name != "" {
				acc.name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				newArgs := tc.Function.Arguments
				prevArgs := acc.args.String()
				acc.args.Reset()
				acc.args.WriteString(newArgs)
				if delta, changed := toolArgsStreamUpdate(prevArgs, newArgs); changed {
					acc.lastEmittedArgs = newArgs
					events <- Event{Type: EventToolCallArgs, ToolCallID: acc.id, ToolName: acc.name, ToolArgs: delta}
				}
			}
		}
		if choice.FinishReason != "" {
			finishReason = choice.FinishReason
		}
	}
	if err := <-errs; err != nil {
		return anyllm.Message{}, streamed.String(), err
	}

	var calls []anyllm.ToolCall
	for _, acc := range toolCalls {
		if acc.name == "" {
			continue
		}
		argsStr := acc.args.String()
		if !acc.started {
			events <- Event{Type: EventToolCallStart, ToolCallID: acc.id, ToolName: acc.name}
		}
		if delta, changed := toolArgsStreamUpdate(acc.lastEmittedArgs, argsStr); changed {
			events <- Event{Type: EventToolCallArgs, ToolCallID: acc.id, ToolName: acc.name, ToolArgs: delta}
		}
		events <- Event{Type: EventToolCallEnd, ToolCallID: acc.id, ToolName: acc.name, ToolArgs: argsStr}
		calls = append(calls, anyllm.ToolCall{
			ID:   acc.id,
			Type: "function",
			Function: anyllm.FunctionCall{
				Name:      acc.name,
				Arguments: normalizeToolCallArguments(argsStr),
			},
		})
	}
	calls = filterExecutableToolCalls(calls)

	streamedContent := streamed.String()
	if bufferingText {
		buffered := bufferedText.String()
		streamedContent = streamedContent + buffered
	}
	if len(calls) == 0 {
		available := toolNamesFromAnyLLM(tools)
		cleaned, textCalls := extractTextToolCalls(streamedContent, available)
		if len(textCalls) > 0 {
			streamedContent = cleaned
			textCalls = filterExecutableToolCalls(textCalls)
			for _, tc := range textCalls {
				events <- Event{Type: EventToolCallStart, ToolCallID: tc.ID, ToolName: tc.Function.Name}
				events <- Event{Type: EventToolCallArgs, ToolCallID: tc.ID, ToolName: tc.Function.Name, ToolArgs: tc.Function.Arguments}
				events <- Event{Type: EventToolCallEnd, ToolCallID: tc.ID, ToolName: tc.Function.Name, ToolArgs: tc.Function.Arguments}
				calls = append(calls, tc)
			}
		} else if bufferingText {
			emitText(bufferedText.String())
			streamedContent = streamed.String()
		}
	} else if bufferingText && bufferedText.Len() > 0 {
		emitText(bufferedText.String())
		streamedContent = streamed.String()
	}

	assistant := anyllm.Message{
		Role:      anyllm.RoleAssistant,
		Content:   streamedContent,
		ToolCalls: calls,
	}
	_ = finishReason
	return assistant, streamedContent, nil
}

type accumulatedToolCall struct {
	id              string
	name            string
	args            strings.Builder
	lastEmittedArgs string
	started         bool
}

func buildAPIMessages(req RunRequest) []anyllm.Message {
	var messages []anyllm.Message
	if sp := strings.TrimSpace(req.SystemPrompt); sp != "" {
		messages = append(messages, anyllm.Message{Role: anyllm.RoleSystem, Content: sp})
	}
	for _, msg := range req.History {
		role := strings.TrimSpace(msg.Role)
		switch role {
		case "user":
			messages = append(messages, anyllm.Message{Role: anyllm.RoleUser, Content: msg.Content})
		case "assistant":
			content := strings.TrimSpace(msg.Content)
			if content == "" {
				content = assistantVizHistorySummary(msg)
			}
			messages = append(messages, anyllm.Message{Role: anyllm.RoleAssistant, Content: content})
		}
	}
	if msg := strings.TrimSpace(req.Message); msg != "" {
		messages = append(messages, anyllm.Message{Role: anyllm.RoleUser, Content: msg})
	}
	return messages
}

func assistantVizHistorySummary(msg model.ChatMessage) string {
	for _, part := range msg.Parts {
		if part.Type != "tool" || !viz.IsVisualizationTool(part.ToolName) {
			continue
		}
		title := strings.TrimSpace(part.VisualizationTitle)
		if title == "" {
			title = "chart"
		}
		return fmt.Sprintf("[Rendered visualization: %s]", title)
	}
	return ""
}

func sessionToolsToAnyLLMTools(tools []sessionMCPTool) []anyllm.Tool {
	out := make([]anyllm.Tool, 0, len(tools))
	for _, t := range tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		out = append(out, anyllm.Tool{
			Type: "function",
			Function: anyllm.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			},
		})
	}
	return out
}

func (a *APIHarnessAgent) approveTool(ctx context.Context, req RunRequest, events chan<- Event, toolName string, args map[string]any) (bool, error) {
	// ask_user creates its own HITL question card; do not gate it behind generic tool approval.
	if hitl.IsQuestionTool(toolName) {
		return true, nil
	}
	if hitl.ShouldAutoApproveTool(toolName, req.ToolApprovalPolicy, req.ToolApprovalTools) {
		return true, nil
	}
	if req.HarnessPermissions == hitl.PermissionsBypass {
		return true, nil
	}
	gate := orchestrationGateFn()
	if gate == nil {
		return false, fmt.Errorf("tool %q requires approval but HITL gate is not configured", toolName)
	}
	created, err := gate.CreateOrchestrationGate(ctx, hitl.CreateInput{
		SessionID: req.LoopSessionID,
		RunID:     req.RunID,
		Kind:      hitl.KindApproval,
		Routing:   hitl.Routing{Channels: []string{hitl.ChannelLoopUI}},
		Payload: map[string]any{
			"title":    "Approve tool call",
			"message":  fmt.Sprintf("Allow tool %q?", toolName),
			"toolName": toolName,
			"toolArgs": args,
		},
	})
	if err != nil {
		return false, err
	}
	events <- Event{Type: EventHITLRequest, Content: created.RequestID}
	resp, err := gate.Wait(ctx, created.RequestID)
	if err != nil {
		return false, err
	}
	switch resp.Status {
	case hitl.StatusAnswered:
		return true, nil
	case hitl.StatusDeclined, hitl.StatusCancelled, hitl.StatusExpired:
		return false, nil
	default:
		return false, fmt.Errorf("tool approval ended with status %q", resp.Status)
	}
}
