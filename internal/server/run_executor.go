// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"nui/internal/agent"
	"nui/internal/hitl"
	"nui/internal/mentions"
	"nui/internal/model"
	"nui/internal/skills"
	"nui/internal/store"
	"nui/internal/viz"
)

type resolvedRunAgent struct {
	Agent                       agent.Agent
	SkipsTopLevelHarnessSession bool
}

func resolveRunAgent(session model.Session, workingDir string) (resolvedRunAgent, error) {
	if def, found := findADLDef(session.AgentType); found {
		return resolvedRunAgent{
			Agent:                       agent.NewADLAgent(def, session.ID, extensionManager),
			SkipsTopLevelHarnessSession: model.SkipsHarnessSessionPersistence(def),
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
	if def, ok := findADLDef(session.AgentType); ok {
		expanded, err := skills.ExpandSlashCommand(
			skills.Context{WorkingDir: workingDir},
			skills.AgentSkills(def),
			message,
		)
		if err != nil {
			return "", err
		}
		message = expanded
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
	session, sessionOK := findSession(sessionID)
	mu.Unlock()
	if sessionOK {
		workingDir, _ := effectiveWorkingDir(session.WorkingDir)
		persistSessionState(sessionID, session, workingDir)
	}
}

func persistAssistantTurn(sessionID, content, newAgentSessionID string, skipTopLevelHarnessSession bool) {
	persistRichAssistantTurn(sessionID, model.ChatMessage{
		ID:        uuid.NewString(),
		Role:      "assistant",
		Content:   content,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, newAgentSessionID, skipTopLevelHarnessSession)
}

// persistRichAssistantTurn saves assistant content. When skipTopLevelHarnessSession is true,
// harness session IDs are not stored on the top-level session key.
func persistRichAssistantTurn(sessionID string, assistantMsg model.ChatMessage, newAgentSessionID string, skipTopLevelHarnessSession bool) {
	if assistantMsg.Content == "" && len(assistantMsg.Parts) == 0 {
		return
	}
	if len(assistantMsg.Parts) > 0 {
		assistantMsg.Parts = viz.NormalizeParts(assistantMsg.Parts)
	}
	if assistantMsg.ID == "" {
		assistantMsg.ID = uuid.NewString()
	}
	if assistantMsg.Role == "" {
		assistantMsg.Role = "assistant"
	}
	if assistantMsg.CreatedAt == "" {
		assistantMsg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	mu.Lock()
	sessionMessages[sessionID] = append(sessionMessages[sessionID], assistantMsg)
	if newAgentSessionID != "" && !skipTopLevelHarnessSession {
		agentSessions[sessionID] = newAgentSessionID
	}
	session, sessionOK := findSession(sessionID)
	snapshot := snapshotData()
	mu.Unlock()
	if sessionOK {
		workingDir, _ := effectiveWorkingDir(session.WorkingDir)
		persistSessionState(sessionID, session, workingDir)
		if sessionUsesExtensionStorage(session) {
			return
		}
	}
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
			NuiSessionID:    opts.Session.ID,
			RunID:            opts.RunID,
			WorkingDir:       workingDir,
			Message:          resolved,
			UserScopeHarness: agent.UserScopeHarnessConfig(opts.Session.AgentConfig),
			AgentConfig:      opts.Session.AgentConfig,
		}
		if def, ok := findADLDef(opts.Session.AgentType); ok {
			runReq.HarnessPermissions = hitl.EffectivePermissions(def, opts.Session.AgentConfig)
			runReq.ToolApprovalPolicy, runReq.ToolApprovalTools = hitl.EffectiveToolApprovals(def, opts.Session.AgentConfig)
			wireOrchestratorRunRequest(&runReq, opts.Session.ID, def)
			wireAPIHarnessRunRequest(&runReq, opts.Session.ID, def)
		}
		if !ra.SkipsTopLevelHarnessSession {
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
