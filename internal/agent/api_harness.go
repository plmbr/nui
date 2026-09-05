// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"nui/internal/llm"
	"nui/internal/hitl"
	"nui/internal/mcpclient"
	"nui/internal/model"
	"nui/internal/viz"
)

const maxAPIToolIterations = 32

// APIHarnessAgent runs an in-process LLM conversation via internal/llm HTTP clients.
type APIHarnessAgent struct {
	Harness model.ADLHarness
	Manager *Manager
}

func (a *APIHarnessAgent) Name() string { return "api" }

func (a *APIHarnessAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	provider, err := NewLLMProvider(a.Harness, req.Env)
	if err != nil {
		return err
	}

	var mcpClient *mcpclient.Client
	var tools []llm.Tool
	var mcpTools []mcpclient.Tool
	if !a.Harness.DisableTools {
		if len(req.MCPServers) > 0 {
			var failures []string
			if a.Manager != nil {
				mcpClient, failures = a.Manager.GetOrConnectSessionMCP(ctx, req.NuiSessionID, req.MCPServers)
			} else {
				mcpClient = mcpclient.New()
				failures = mcpClient.ConnectServers(ctx, req.MCPServers)
				defer mcpClient.Close()
			}
			for _, msg := range failures {
				events <- Event{Type: EventText, Content: msg + "\n"}
			}
		}
		if mcpClient != nil {
			mcpTools = mcpClient.Tools()
			tools = mcpToolsToLLM(mcpTools)
		}
		if len(req.ExtraTools) > 0 {
			tools = append(tools, req.ExtraTools...)
		}
	}

	if catalog := mcpToolCatalogSystemPrompt(mcpTools); catalog != "" {
		req.SystemPrompt = appendSystemPromptBlock(req.SystemPrompt, catalog)
	}

	messages := buildAPIMessages(req)

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

	for iteration := 0; iteration < maxAPIToolIterations; iteration++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		assistant, streamedText, err := a.streamCompletion(ctx, provider, modelName, normalizeAPIMessages(messages), tools, req.Message, events)
		if err != nil {
			if errors.Is(err, llm.ErrModelNotFound) && candidateIdx < len(modelCandidates)-1 {
				candidateIdx++
				modelName = modelCandidates[candidateIdx]
				continue
			}
			return err
		}
		if len(assistant.ToolCalls) == 0 {
			text := strings.TrimSpace(streamedText)
			if text == "" {
				if c, ok := assistant.Content.(string); ok {
					text = strings.TrimSpace(c)
				}
			}
			if text == "" && strings.TrimSpace(a.Harness.Provider) == "ollama" && shouldAnswerInPlainText(req.Message) {
				if _, err := a.streamPlainTextCompletion(ctx, provider, modelName, normalizeAPIMessages(messages), events); err != nil {
					return err
				}
			}
			events <- Event{Type: EventDone}
			return nil
		}

		filtered, _ := filterSpuriousAskUser(
			filterExecutableToolCalls(normalizeAPIToolCalls(assistant.ToolCalls)),
			req.Message,
			a.Harness.Provider,
		)
		assistant.ToolCalls = filtered
		if len(assistant.ToolCalls) == 0 {
			text := strings.TrimSpace(streamedText)
			if text == "" {
				if c, ok := assistant.Content.(string); ok {
					text = strings.TrimSpace(c)
				}
			}
			if text == "" && strings.TrimSpace(a.Harness.Provider) == "ollama" && shouldAnswerInPlainText(req.Message) {
				if _, err := a.streamPlainTextCompletion(ctx, provider, modelName, normalizeAPIMessages(messages), events); err != nil {
					return err
				}
			}
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
				messages = append(messages, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: tc.ID,
					ToolName:   toolName,
					Content:    resultText,
				})
				continue
			}

			resultText := ""
			var callErr error
			handled := false
			if req.HandleExtraTool != nil {
				var ok bool
				resultText, ok, callErr = req.HandleExtraTool(ctx, toolName, args)
				handled = ok
			}
			if !handled {
				if mcpClient == nil {
					resultText = "error: no tool backend for " + toolName
				} else {
					resultText, callErr = mcpClient.CallTool(ctx, toolName, args)
					if callErr != nil {
						resultText = "error: " + callErr.Error()
					}
				}
			} else if callErr != nil {
				resultText = "error: " + callErr.Error()
			}
			events <- Event{Type: EventToolCallResult, ToolCallID: tc.ID, ToolName: toolName, Content: resultText}
			if viz.IsVisualizationTool(toolName) && callErr == nil && handled {
				executedViz = true
			}
			if viz.IsVisualizationTool(toolName) && !handled && callErr == nil {
				executedViz = true
			}
			messages = append(messages, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: tc.ID,
				ToolName:   toolName,
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
	provider llm.Provider,
	modelName string,
	messages []llm.Message,
	tools []llm.Tool,
	userMessage string,
	events chan<- Event,
) (llm.Message, string, error) {
	if a.Harness.DisableTools {
		tools = nil
	}
	params := llm.CompletionParams{
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
	deferText := len(tools) > 0
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
			if deferText {
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
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Function.Name != "" {
				acc.name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				updated := accumulateToolCallArgs(acc.args.String(), tc.Function.Arguments)
				acc.args.Reset()
				acc.args.WriteString(updated)
			}
		}
		if choice.FinishReason != "" {
			finishReason = choice.FinishReason
		}
	}
	if err := <-errs; err != nil {
		return llm.Message{}, streamed.String(), err
	}

	var calls []llm.ToolCall
	for _, acc := range toolCalls {
		if acc.name == "" {
			continue
		}
		argsStr := acc.args.String()
		calls = append(calls, llm.ToolCall{
			ID:   acc.id,
			Type: "function",
			Function: llm.FunctionCall{
				Name:      acc.name,
				Arguments: normalizeToolCallArguments(argsStr),
			},
		})
	}

	emitToolCallEvents := func(tc llm.ToolCall) {
		events <- Event{Type: EventToolCallStart, ToolCallID: tc.ID, ToolName: tc.Function.Name}
		events <- Event{Type: EventToolCallArgs, ToolCallID: tc.ID, ToolName: tc.Function.Name, ToolArgs: tc.Function.Arguments}
		events <- Event{Type: EventToolCallEnd, ToolCallID: tc.ID, ToolName: tc.Function.Name, ToolArgs: tc.Function.Arguments}
	}

	streamedContent := streamed.String()
	if bufferedText.Len() > 0 {
		streamedContent += bufferedText.String()
	}
	if len(calls) == 0 {
		available := toolNamesFromLLM(tools)
		cleaned, textCalls := extractTextToolCalls(streamedContent, available)
		if len(textCalls) > 0 {
			streamedContent = cleaned
			textCalls = filterExecutableToolCalls(textCalls)
			calls = append(calls, textCalls...)
		} else if looksLikeTextToolJSON(streamedContent) {
			streamedContent = ""
		}
	}

	var removedAskUser []llm.ToolCall
	var removedViz []llm.ToolCall
	calls, removedViz = filterSpuriousVisualization(calls, userMessage, a.Harness.Provider)
	calls = filterExecutableToolCalls(calls)
	calls, removedAskUser = filterSpuriousAskUser(calls, userMessage, a.Harness.Provider)
	if strings.TrimSpace(streamedContent) == "" {
		if strings.TrimSpace(a.Harness.Provider) != "ollama" && len(removedAskUser) > 0 {
			streamedContent = salvageAskUserText(removedAskUser)
		}
		if strings.TrimSpace(streamedContent) == "" && len(removedViz) > 0 {
			streamedContent = salvageVisualizationText(removedViz)
		}
	}

	if len(calls) > 0 || looksLikeTextToolJSON(streamedContent) {
		if obj := findEmbeddedJSONObjectOrEmpty(streamedContent); obj != "" {
			streamedContent = strings.TrimSpace(strings.Replace(streamedContent, obj, "", 1))
		} else if looksLikeTextToolJSON(streamedContent) {
			streamedContent = ""
		}
	}

	for _, tc := range calls {
		emitToolCallEvents(tc)
	}

	if deferText && strings.TrimSpace(streamedContent) != "" {
		emitText(streamedContent)
	}

	assistant := llm.Message{
		Role:      llm.RoleAssistant,
		Content:   streamedContent,
		ToolCalls: calls,
	}
	_ = finishReason
	return assistant, streamedContent, nil
}

func (a *APIHarnessAgent) streamPlainTextCompletion(
	ctx context.Context,
	provider llm.Provider,
	modelName string,
	messages []llm.Message,
	events chan<- Event,
) (string, error) {
	chunks, errs := provider.CompletionStream(ctx, llm.CompletionParams{
		Model:    modelName,
		Messages: messages,
	})
	var b strings.Builder
	for chunk := range chunks {
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		b.WriteString(delta)
		events <- Event{Type: EventText, Content: delta}
	}
	if err := <-errs; err != nil {
		return b.String(), err
	}
	return b.String(), nil
}

type accumulatedToolCall struct {
	id   string
	name string
	args strings.Builder
}

func buildAPIMessages(req RunRequest) []llm.Message {
	var messages []llm.Message
	if sp := strings.TrimSpace(req.SystemPrompt); sp != "" {
		messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: sp})
	}
	for _, msg := range req.History {
		role := strings.TrimSpace(msg.Role)
		switch role {
		case "user":
			messages = append(messages, llm.Message{Role: llm.RoleUser, Content: msg.Content})
		case "assistant":
			content := assistantHistoryContent(msg)
			if content == "" {
				continue
			}
			messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: content})
		}
	}
	if msg := strings.TrimSpace(req.Message); msg != "" {
		messages = append(messages, llm.Message{Role: llm.RoleUser, Content: msg})
	}
	return messages
}

func assistantHistoryContent(msg model.ChatMessage) string {
	if content := strings.TrimSpace(msg.Content); content != "" {
		if looksLikeTextToolJSON(content) {
			return ""
		}
		return content
	}
	var b strings.Builder
	for _, part := range msg.Parts {
		switch part.Type {
		case "text":
			if t := strings.TrimSpace(part.Content); t != "" {
				if b.Len() > 0 {
					b.WriteString("\n\n")
				}
				b.WriteString(t)
			}
		case "tool":
			if viz.IsVisualizationTool(part.ToolName) {
				title := strings.TrimSpace(part.VisualizationTitle)
				if title == "" {
					title = "chart"
				}
				if b.Len() > 0 {
					b.WriteString("\n\n")
				}
				fmt.Fprintf(&b, "[Rendered visualization: %s]", title)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func mcpToolsToLLM(tools []mcpclient.Tool) []llm.Tool {
	out := make([]llm.Tool, 0, len(tools))
	for _, t := range tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		out = append(out, llm.Tool{
			Type: "function",
			Function: llm.Function{
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
		SessionID: req.NuiSessionID,
		RunID:     req.RunID,
		Kind:      hitl.KindApproval,
		Routing:   hitl.Routing{Channels: []string{hitl.ChannelnuiUI}},
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
