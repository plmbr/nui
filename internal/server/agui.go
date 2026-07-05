// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"loop/internal/agent"
	"loop/internal/hitl"
	"loop/internal/model"
	"loop/internal/viz"
)

type aguiRunInput struct {
	ThreadID string `json:"threadId"`
	RunID    string `json:"runId"`
	Messages []struct {
		ID      string `json:"id"`
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

var aguiSessionRunLocks sync.Map // sessionID -> *sync.Mutex

func lockSessionAGUIRun(sessionID string) func() {
	v, _ := aguiSessionRunLocks.LoadOrStore(sessionID, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func handleSessionAGUI(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input aguiRunInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var lastUserMessage string
	var lastUserID string
	for i := len(input.Messages) - 1; i >= 0; i-- {
		if input.Messages[i].Role == "user" {
			lastUserMessage = strings.TrimSpace(input.Messages[i].Content)
			lastUserID = strings.TrimSpace(input.Messages[i].ID)
			break
		}
	}
	if lastUserMessage == "" {
		http.Error(w, "last message must be from the user", http.StatusBadRequest)
		return
	}

	mu.RLock()
	session, ok := findSession(sessionID)
	agentSessionID := agentSessions[sessionID]
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	workingDir, err := effectiveWorkingDir(session.WorkingDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resolvedMessage, err := prepareRunMessage(r.Context(), session, workingDir, lastUserMessage, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	unlockSessionRun := lockSessionAGUIRun(sessionID)
	defer unlockSessionRun()

	userID := lastUserID
	if userID == "" {
		userID = uuid.NewString()
	}
	userMsg := model.ChatMessage{
		ID:        userID,
		Role:      "user",
		Content:   resolvedMessage,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	mu.Lock()
	sessionMessages[sessionID] = append(sessionMessages[sessionID], userMsg)
	mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	threadID := input.ThreadID
	if threadID == "" {
		threadID = sessionID
	}
	runID := input.RunID
	if runID == "" {
		runID = uuid.NewString()
	}

	writeAGUIEvent(w, flusher, map[string]any{
		"type":     "RUN_STARTED",
		"threadId": threadID,
		"runId":    runID,
	})

	messageID := uuid.NewString()
	var assistantContent strings.Builder
	var newAgentSessionID string
	acc := newAssistantPartAccumulator()

	createRunRecord(sessionID, runID, resolvedMessage)

	var ag agent.Agent
	var multiStepWorkflow bool
	if def, found := findADLDef(session.AgentType); found {
		ag = agent.NewADLAgent(def, session.ID, extensionManager)
		multiStepWorkflow = model.IsMultiStepWorkflow(def)
	} else {
		var err error
		ag, err = extensionManager.GetAgent(session.ID, session.AgentType, workingDir, session.AgentConfig)
		if err != nil {
			writeAGUIEvent(w, flusher, map[string]any{
				"type":    "RUN_ERROR",
				"message": fmt.Sprintf("agent unavailable: %v", err),
			})
			return
		}
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	registerActiveRun(sessionID, runID, cancelRun)
	defer unregisterActiveRun(sessionID, runID)

	reqCtx := r.Context()

	initHITL()
	hitlCh, unsubHITL := subscribeHITL(sessionID)
	defer unsubHITL()

	events := make(chan agent.Event, 64)
	go func() {
		defer close(events)
		runReq := agent.RunRequest{
			LoopSessionID:    sessionID,
			RunID:            runID,
			WorkingDir:       workingDir,
			Message:          resolvedMessage,
			UserScopeHarness: agent.UserScopeHarnessConfig(session.AgentConfig),
			AgentConfig:      session.AgentConfig,
		}
		if def, ok := findADLDef(session.AgentType); ok {
			runReq.HarnessPermissions = hitl.EffectivePermissions(def, session.AgentConfig)
			runReq.ToolApprovalPolicy, runReq.ToolApprovalTools = hitl.EffectiveToolApprovals(def, session.AgentConfig)
		}
		if !multiStepWorkflow {
			runReq.SessionID = agentSessionID
		}
		err := ag.Run(runCtx, runReq, events)
		if err != nil && runCtx.Err() == nil {
			events <- agent.Event{Type: agent.EventError, Error: err.Error()}
		}
	}()

	runErrored := false
	runCancelled := false
	seq := 0
	mcpLookup := func(toolName string) (string, string, bool) {
		return mcpManager.LookupToolUI(toolName)
	}
	go func() {
		for req := range hitlCh {
			writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
				"type": "CUSTOM",
				"name": "hitl_request",
				"value": map[string]any{
					"requestId": req.RequestID,
					"sessionId": req.SessionID,
					"runId":     req.RunID,
					"stepName":  req.StepName,
					"kind":      req.Kind,
					"payload":   req.Payload,
					"status":    req.Status,
					"expiresAt": req.ExpiresAt,
				},
			})
		}
	}()

	for ev := range events {
		seq++
		_ = appendRunEvent(runID, seq, ev)
		acc.applyEvent(ev, mcpLookup)
		switch ev.Type {
		case agent.EventText:
			if ev.Content == "" {
				continue
			}
			assistantContent.WriteString(ev.Content)
			writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
				"type":      "TEXT_MESSAGE_CHUNK",
				"messageId": messageID,
				"delta":     ev.Content,
			})
		case agent.EventToolCallStart:
			writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
				"type":         "TOOL_CALL_START",
				"toolCallId":   ev.ToolCallID,
				"toolCallName": ev.ToolName,
			})
		case agent.EventToolCallArgs:
			if ev.ToolArgs == "" {
				continue
			}
			writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
				"type":       "TOOL_CALL_ARGS",
				"toolCallId": ev.ToolCallID,
				"delta":      ev.ToolArgs,
			})
			emitVisualizationEvent(reqCtx, w, flusher, acc, ev.ToolCallID, ev.ToolName, ev.ToolArgs)
		case agent.EventToolCallEnd:
			writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
				"type":       "TOOL_CALL_END",
				"toolCallId": ev.ToolCallID,
			})
			emitVisualizationEvent(reqCtx, w, flusher, acc, ev.ToolCallID, ev.ToolName, ev.ToolArgs)
			if uri, server, ok := mcpManager.LookupToolUI(ev.ToolName); ok {
				var toolInput map[string]any
				if ev.ToolArgs != "" {
					json.Unmarshal([]byte(ev.ToolArgs), &toolInput)
				}
				if toolInput == nil {
					toolInput = map[string]any{}
				}
				writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
					"type": "CUSTOM",
					"name": "mcp_app",
					"value": map[string]any{
						"toolCallId":  ev.ToolCallID,
						"toolName":    ev.ToolName,
						"serverName":  server,
						"resourceUri": uri,
						"toolInput":   toolInput,
					},
				})
			}
		case agent.EventToolCallResult:
			writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
				"type":       "TOOL_CALL_RESULT",
				"messageId":  uuid.NewString(),
				"toolCallId": ev.ToolCallID,
				"content":    ev.Content,
				"role":       "tool",
			})
		case agent.EventImage:
			if ev.ImageData == "" {
				continue
			}
			mediaType := ev.ImageMediaType
			if mediaType == "" {
				mediaType = "image/png"
			}
			writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
				"type": "CUSTOM",
				"name": "image",
				"value": map[string]any{
					"mediaType": mediaType,
					"data":      ev.ImageData,
				},
			})
		case agent.EventHITLRequest:
			if req, err := coordinator().Get(runCtx, ev.Content); err == nil {
				writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
					"type": "CUSTOM",
					"name": "hitl_request",
					"value": map[string]any{
						"requestId": req.RequestID,
						"sessionId": req.SessionID,
						"runId":     req.RunID,
						"stepName":  req.StepName,
						"kind":      req.Kind,
						"payload":   req.Payload,
						"status":    req.Status,
						"expiresAt": req.ExpiresAt,
					},
				})
			}
		case agent.EventDone:
			newAgentSessionID = ev.SessionID
		case agent.EventError:
			runErrored = true
			writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
				"type":    "RUN_ERROR",
				"message": ev.Error,
			})
		}
	}

	if runCtx.Err() != nil {
		runCancelled = true
		runErrored = true
		writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
			"type":    "RUN_ERROR",
			"message": "cancelled",
		})
	} else if !runErrored {
		writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
			"type":     "RUN_FINISHED",
			"threadId": threadID,
			"runId":    runID,
		})
	}

	switch {
	case runCancelled:
		assistantMsg := acc.toMessage(messageID)
		if assistantMsg.CreatedAt == "" {
			assistantMsg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		if assistantMsg.Content != "" || len(assistantMsg.Parts) > 0 {
			persistRichAssistantTurn(sessionID, assistantMsg, newAgentSessionID, multiStepWorkflow)
		}
		finishRunRecord(runID, RunStatusCancelled, assistantContent.String(), "cancelled")
	case runErrored:
		assistantMsg := acc.toMessage(messageID)
		if assistantMsg.CreatedAt == "" {
			assistantMsg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		if assistantMsg.Content != "" || len(assistantMsg.Parts) > 0 {
			persistRichAssistantTurn(sessionID, assistantMsg, newAgentSessionID, multiStepWorkflow)
		}
		finishRunRecord(runID, RunStatusFailed, assistantContent.String(), "error")
	default:
		assistantMsg := acc.toMessage(messageID)
		if assistantMsg.CreatedAt == "" {
			assistantMsg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		if assistantMsg.Content != "" || len(assistantMsg.Parts) > 0 {
			persistRichAssistantTurn(sessionID, assistantMsg, newAgentSessionID, multiStepWorkflow)
		}
		finishRunRecord(runID, RunStatusCompleted, assistantContent.String(), "")
	}
}

func writeAGUIEventIfConnected(reqCtx context.Context, w http.ResponseWriter, flusher http.Flusher, event any) {
	if reqCtx.Err() != nil {
		return
	}
	writeAGUIEvent(w, flusher, event)
}

func writeAGUIEvent(w http.ResponseWriter, flusher http.Flusher, event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func emitVisualizationEvent(reqCtx context.Context, w http.ResponseWriter, flusher http.Flusher, acc *assistantPartAccumulator, toolCallID, toolName, toolArgsJSON string) {
	if toolCallID == "" {
		return
	}
	var toolInput map[string]any
	if toolArgsJSON != "" {
		_ = json.Unmarshal([]byte(toolArgsJSON), &toolInput)
	}
	if toolInput == nil {
		if part := acc.toolPartForCall(toolCallID); part != nil && len(part.ToolArgs) > 0 {
			toolInput = part.ToolArgs
		}
	}
	if toolName == "" {
		if part := acc.toolPartForCall(toolCallID); part != nil {
			toolName = part.ToolName
		}
	}
	html, title, ok := viz.ParseFromTool(toolName, toolInput)
	if !ok {
		return
	}
	writeAGUIEventIfConnected(reqCtx, w, flusher, map[string]any{
		"type": "CUSTOM",
		"name": "visualization",
		"value": map[string]any{
			"toolCallId": toolCallID,
			"html":       html,
			"title":      title,
		},
	})
}
