// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"loop/internal/agent"
	"loop/internal/mentions"
	"loop/internal/model"
	"loop/internal/store"
)

type resolvedRunAgent struct {
	Agent agent.Agent
	IsADL bool
}

func resolveRunAgent(session model.Session, workingDir string) (resolvedRunAgent, error) {
	if def, found := findADLDef(session.AgentType); found {
		isADL := len(def.Steps) > 0 || def.Kind == "workflow"
		return resolvedRunAgent{
			Agent: agent.NewADLAgent(def, session.ID, extensionManager),
			IsADL: isADL,
		}, nil
	}
	ag, err := extensionManager.GetAgent(session.ID, session.AgentType, workingDir, session.AgentConfig)
	if err != nil {
		return resolvedRunAgent{}, fmt.Errorf("agent unavailable: %w", err)
	}
	return resolvedRunAgent{Agent: ag}, nil
}

type executeRunOptions struct {
	SessionID       string
	Session         model.Session
	Message         string
	ResolveMentions bool
	RunID           string
	PersistLog      bool
	AgentSessionID  string
	OnEvent         func(agent.Event)
}

type executeRunResult struct {
	AssistantContent  string
	NewAgentSessionID string
	Errored           bool
	Cancelled         bool
	ErrorMessage      string
}

func prepareRunMessage(ctx context.Context, session model.Session, workingDir, message string, resolveMentions bool) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		if def, ok := findADLDef(session.AgentType); ok && model.IsADLAutoPrompt(def) {
			message = model.ResolveADLLaunchPrompt(def, "")
		}
	}
	if message == "" {
		return "", fmt.Errorf("message is required")
	}
	if !resolveMentions {
		return message, nil
	}
	return mentions.DefaultRegistry.ResolveMessage(ctx, workingDir, message, mentionAllowedRoots(session))
}

func appendUserMessage(sessionID, content string) {
	userMsg := model.ChatMessage{
		ID:        uuid.NewString(),
		Role:      "user",
		Content:   content,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	mu.Lock()
	sessionMessages[sessionID] = append(sessionMessages[sessionID], userMsg)
	mu.Unlock()
}

func persistAssistantTurn(sessionID, content, newAgentSessionID string, isADL bool) {
	if content == "" {
		return
	}
	assistantMsg := model.ChatMessage{
		ID:        uuid.NewString(),
		Role:      "assistant",
		Content:   content,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	mu.Lock()
	sessionMessages[sessionID] = append(sessionMessages[sessionID], assistantMsg)
	if newAgentSessionID != "" && !isADL {
		agentSessions[sessionID] = newAgentSessionID
	}
	snapshot := snapshotData()
	mu.Unlock()
	if err := store.SaveData(snapshot); err != nil {
		fmt.Fprintf(os.Stderr, "warn: save session: %v\n", err)
	}
}

func executeRun(ctx context.Context, opts executeRunOptions) executeRunResult {
	workingDir, err := effectiveWorkingDir(opts.Session.WorkingDir)
	if err != nil {
		return executeRunResult{Errored: true, ErrorMessage: err.Error()}
	}

	resolved, err := prepareRunMessage(ctx, opts.Session, workingDir, opts.Message, opts.ResolveMentions)
	if err != nil {
		return executeRunResult{Errored: true, ErrorMessage: err.Error()}
	}

	ra, err := resolveRunAgent(opts.Session, workingDir)
	if err != nil {
		return executeRunResult{Errored: true, ErrorMessage: err.Error()}
	}

	events := make(chan agent.Event, 64)
	go func() {
		defer close(events)
		runReq := agent.RunRequest{
			WorkingDir:       workingDir,
			Message:          resolved,
			UserScopeHarness: agent.UserScopeHarnessConfig(opts.Session.AgentConfig),
		}
		if !ra.IsADL {
			runReq.SessionID = opts.AgentSessionID
		}
		runErr := ra.Agent.Run(ctx, runReq, events)
		if runErr != nil && ctx.Err() == nil {
			events <- agent.Event{Type: agent.EventError, Error: runErr.Error()}
		}
	}()

	var assistantContent strings.Builder
	var newAgentSessionID string
	var result executeRunResult
	seq := 0

	for ev := range events {
		seq++
		if opts.PersistLog && opts.RunID != "" {
			_ = appendRunEvent(opts.RunID, seq, ev)
		}
		if opts.OnEvent != nil {
			opts.OnEvent(ev)
		}
		switch ev.Type {
		case agent.EventText:
			assistantContent.WriteString(ev.Content)
		case agent.EventDone:
			newAgentSessionID = ev.SessionID
		case agent.EventError:
			result.Errored = true
			result.ErrorMessage = ev.Error
		}
	}

	if ctx.Err() != nil {
		result.Cancelled = true
	} else if result.Errored {
		// keep error from event
	} else {
		result.Errored = false
	}

	result.AssistantContent = assistantContent.String()
	result.NewAgentSessionID = newAgentSessionID
	return result
}
