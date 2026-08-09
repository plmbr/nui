// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"nui/internal/agent"
	"nui/internal/model"
	"nui/internal/store"
)

const (
	sessionTitleSystemPrompt = "You generate short chat titles. Do not use tools. Reply with only the title text: a concise phrase of at most 6 words. No quotes, punctuation, markdown, or explanation."
	// PendingSessionTitle is shown until the first message triggers auto-title.
	PendingSessionTitle      = "New session"
	maxTitlePromptUserChars  = 500
	maxTitlePromptAssistChars = 400
	maxSessionTitleLen       = 80
	fallbackTitleLen         = 60
	titleGenerationTimeout   = 45 * time.Second
)

var titleGenerationInFlight sync.Map // sessionID -> struct{}

// maybeGenerateSessionTitle renames a session from its first exchange using an ephemeral harness call.
func maybeGenerateSessionTitle(sessionID string) {
	if _, loaded := titleGenerationInFlight.LoadOrStore(sessionID, struct{}{}); loaded {
		return
	}
	defer titleGenerationInFlight.Delete(sessionID)

	mu.RLock()
	session, ok := findSession(sessionID)
	msgs := append([]model.ChatMessage(nil), sessionMessages[sessionID]...)
	mu.RUnlock()
	if !ok {
		return
	}

	def, hasDef := resolveSessionADLDef(session)
	if !hasDef || !shouldAutoTitle(session, def, msgs) {
		return
	}

	defer extensionManager.Stop(agent.EphemeralProjectID(sessionID))

	ctx, cancel := context.WithTimeout(context.Background(), titleGenerationTimeout)
	defer cancel()

	title, err := generateSessionTitle(ctx, session, def, msgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: generate session title for %s: %v\n", sessionID, err)
	}
	if title == "" {
		title = fallbackSessionTitle(msgs)
	}
	title = sanitizeSessionTitle(title)
	if title == "" {
		return
	}

	mu.Lock()
	current, ok := findSession(sessionID)
	currentMsgs := append([]model.ChatMessage(nil), sessionMessages[sessionID]...)
	if !ok || !shouldAutoTitle(current, def, currentMsgs) {
		mu.Unlock()
		return
	}
	if !renameSession(sessionID, title) {
		mu.Unlock()
		return
	}
	snapshot := snapshotData()
	mu.Unlock()

	if err := store.SaveData(snapshot); err != nil {
		fmt.Fprintf(os.Stderr, "warn: save data after auto title: %v\n", err)
	}
	notifySessionsChanged()
}

func isPendingSessionName(name string, def model.ADLDefinition) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if name == PendingSessionTitle {
		return true
	}
	// Legacy sessions created before auto-title used the agent label as a placeholder.
	return name == model.ADLAgentLabel(def) ||
		name == model.ADLAgentID(def) ||
		name == strings.TrimSpace(def.Name) ||
		name == strings.TrimSpace(def.ID)
}

func shouldAutoTitle(session model.Session, def model.ADLDefinition, msgs []model.ChatMessage) bool {
	if session.ScheduleID != "" {
		return false
	}
	if !isPendingSessionName(session.Name, def) {
		return false
	}
	return countUserMessages(msgs) == 1
}

func countUserMessages(msgs []model.ChatMessage) int {
	n := 0
	for _, m := range msgs {
		if m.Role == "user" && strings.TrimSpace(m.Content) != "" {
			n++
		}
	}
	return n
}

func fallbackSessionTitle(msgs []model.ChatMessage) string {
	for _, m := range msgs {
		if m.Role != "user" {
			continue
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if i := strings.IndexAny(content, "\r\n"); i >= 0 {
			content = strings.TrimSpace(content[:i])
		}
		return truncateRunes(content, fallbackTitleLen)
	}
	return ""
}

func buildTitlePrompt(msgs []model.ChatMessage) string {
	var b strings.Builder
	b.WriteString("Generate a short title for this conversation.\n\n")
	wrote := false
	for _, m := range msgs {
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		switch m.Role {
		case "user":
			content = truncateRunes(content, maxTitlePromptUserChars)
			fmt.Fprintf(&b, "User: %s\n", content)
			wrote = true
		case "assistant":
			content = truncateRunes(content, maxTitlePromptAssistChars)
			fmt.Fprintf(&b, "Assistant: %s\n", content)
			wrote = true
		}
	}
	if !wrote {
		return ""
	}
	return b.String()
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func sanitizeSessionTitle(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	s = strings.Trim(s, `"'`)
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > maxSessionTitleLen {
		s = strings.TrimSpace(string(runes[:maxSessionTitleLen]))
	}
	return s
}

func generateSessionTitle(ctx context.Context, session model.Session, def model.ADLDefinition, msgs []model.ChatMessage) (string, error) {
	prompt := buildTitlePrompt(msgs)
	if prompt == "" {
		return "", nil
	}

	workingDir, err := effectiveWorkingDir(session.WorkingDir)
	if err != nil {
		return "", err
	}

	adlAg := agent.NewADLAgent(def, session.ID, extensionManager)
	runReq := agent.RunRequest{
		NuiSessionID:    session.ID,
		WorkingDir:       workingDir,
		Message:          prompt,
		SystemPrompt:     sessionTitleSystemPrompt,
		UserScopeHarness: agent.UserScopeHarnessConfig(session.AgentConfig),
		AgentConfig:      session.AgentConfig,
	}

	events := make(chan agent.Event, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(events)
		errCh <- adlAg.RunEphemeral(ctx, runReq, events)
	}()

	var content strings.Builder
	for ev := range events {
		if ev.Type == agent.EventText {
			content.WriteString(ev.Content)
		}
	}
	if err := <-errCh; err != nil {
		return "", err
	}
	return content.String(), nil
}
