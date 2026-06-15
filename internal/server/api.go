// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"loop/internal/agent"
	"loop/internal/model"
	"loop/internal/store"
)

type AgentType struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

var agentTypes = []AgentType{
	{ID: "claude-code", Label: "Claude Code"},
	{ID: "pi", Label: "Pi"},
	{ID: "docker", Label: "Docker"},
	{ID: "remote", Label: "Remote"},
	{ID: "docker-claude", Label: "Docker Claude Code"},
	{ID: "docker-pi", Label: "Docker Pi"},
}

type AppConfig struct {
	CopilotKitPublicApiKey string `json:"copilotKitPublicApiKey"`
	CopilotKitRuntimeURL   string `json:"copilotKitRuntimeUrl"`
}

type SandboxCapabilities struct {
	Bwrap agent.BwrapStatus `json:"bwrap"`
}

type Capabilities struct {
	Sandbox SandboxCapabilities `json:"sandbox"`
}


var (
	mu              sync.RWMutex
	sessions        []model.Session
	sessionMessages = map[string][]model.ChatMessage{}
	agentSessions   = map[string]string{} // session ID → agent session ID
)

func initStore() error {
	data, err := store.LoadData()
	if err != nil {
		return err
	}
	mu.Lock()
	sessions = data.Sessions
	agentSessions = data.AgentSessions
	mu.Unlock()
	return nil
}

// snapshotData must be called with mu held.
func snapshotData() store.Data {
	ss := make([]model.Session, len(sessions))
	copy(ss, sessions)
	as := make(map[string]string, len(agentSessions))
	for k, v := range agentSessions {
		as[k] = v
	}
	return store.Data{Sessions: ss, AgentSessions: as}
}

func registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/sessions", handleSessions)
	mux.HandleFunc("/api/sessions/", handleSession)
	mux.HandleFunc("/api/agent-types", handleAgentTypes)
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/settings", handleSettings)
	mux.HandleFunc("/api/capabilities", handleCapabilities)
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		list := make([]model.Session, len(sessions))
		copy(list, sessions)
		mu.RUnlock()
		writeJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var req struct {
			Name        string         `json:"name"`
			WorkingDir  string         `json:"workingDir"`
			AgentType   string         `json:"agentType"`
			AgentConfig map[string]any `json:"agentConfig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.AgentType == "" {
			http.Error(w, "name and agentType are required", http.StatusBadRequest)
			return
		}
		s := model.Session{
			ID:          uuid.NewString(),
			Name:        req.Name,
			WorkingDir:  req.WorkingDir,
			AgentType:   req.AgentType,
			AgentConfig: req.AgentConfig,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		}
		mu.Lock()
		sessions = append(sessions, s)
		snapshot := snapshotData()
		mu.Unlock()
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save data after create: %v\n", err)
		}
		writeJSON(w, http.StatusCreated, s)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	all := make([]AgentType, len(agentTypes))
	copy(all, agentTypes)

	defs, err := store.LoadADLDefinitions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: load ADL definitions: %v\n", err)
	}
	for _, def := range defs {
		all = append(all, AgentType{ID: "adl:" + def.Name, Label: def.Name})
	}

	writeJSON(w, http.StatusOK, all)
}

func findSession(id string) (model.Session, bool) {
	for _, s := range sessions {
		if s.ID == id {
			return s, true
		}
	}
	return model.Session{}, false
}

func deleteSession(id string) bool {
	for i, s := range sessions {
		if s.ID == id {
			sessions = append(sessions[:i], sessions[i+1:]...)
			return true
		}
	}
	return false
}

func renameSession(id, name string) bool {
	for i, s := range sessions {
		if s.ID == id {
			sessions[i].Name = name
			return true
		}
	}
	return false
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	// Route /api/sessions/:id/<sub>
	if idx := strings.Index(path, "/"); idx != -1 {
		id := path[:idx]
		rest := path[idx+1:]
		switch rest {
		case "messages":
			handleSessionMessages(w, r, id)
		case "chat":
			handleSessionChat(w, r, id)
		case "history":
			handleSessionHistory(w, r, id)
		default:
			http.NotFound(w, r)
		}
		return
	}

	id := path
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		s, ok := findSession(id)
		mu.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, s)

	case http.MethodPatch:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		mu.Lock()
		found := renameSession(id, name)
		var updated model.Session
		var snapshot store.Data
		if found {
			updated, _ = findSession(id)
			snapshot = snapshotData()
		}
		mu.Unlock()
		if !found {
			http.NotFound(w, r)
			return
		}
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save data after rename: %v\n", err)
		}
		writeJSON(w, http.StatusOK, updated)

	case http.MethodDelete:
		mu.Lock()
		var agentSessionID, workingDir, agentType string
		if s, ok := findSession(id); ok {
			agentSessionID = agentSessions[id]
			workingDir = s.WorkingDir
			agentType = s.AgentType
		}
		removed := deleteSession(id)
		if removed {
			delete(agentSessions, id)
			delete(sessionMessages, id)
		}
		var snapshot store.Data
		if removed {
			snapshot = snapshotData()
		}
		mu.Unlock()
		if !removed {
			http.NotFound(w, r)
			return
		}
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save data after delete: %v\n", err)
		}
		extensionManager.Stop(id)
		if agentSessionID != "" {
			var delErr error
			if agentType == "pi" {
				delErr = store.DeletePiSession(workingDir, agentSessionID)
			} else {
				delErr = store.DeleteClaudeSession(workingDir, agentSessionID)
			}
			if delErr != nil {
				fmt.Fprintf(os.Stderr, "warn: delete session file: %v\n", delErr)
			}
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSessionMessages(w http.ResponseWriter, r *http.Request, sessionID string) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		msgs := sessionMessages[sessionID]
		if msgs == nil {
			msgs = []model.ChatMessage{}
		}
		mu.RUnlock()
		writeJSON(w, http.StatusOK, msgs)

	case http.MethodPut:
		var msgs []model.ChatMessage
		if err := json.NewDecoder(r.Body).Decode(&msgs); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		mu.Lock()
		sessionMessages[sessionID] = msgs
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := AppConfig{
		CopilotKitPublicApiKey: os.Getenv("COPILOTKIT_PUBLIC_API_KEY"),
		CopilotKitRuntimeURL:   os.Getenv("COPILOTKIT_RUNTIME_URL"),
	}
	writeJSON(w, http.StatusOK, cfg)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s, err := store.LoadSettings()
		if err != nil {
			http.Error(w, "failed to load settings", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, s)

	case http.MethodPut:
		current, err := store.LoadSettings()
		if err != nil {
			http.Error(w, "failed to load settings", http.StatusInternalServerError)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&current); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if current.Theme != "light" && current.Theme != "dark" {
			http.Error(w, "theme must be 'light' or 'dark'", http.StatusBadRequest)
			return
		}
		if err := store.SaveSettings(current); err != nil {
			http.Error(w, "failed to save settings", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, current)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSessionChat(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
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

	userMsg := model.ChatMessage{
		ID:        uuid.NewString(),
		Role:      "user",
		Content:   req.Message,
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

	var ag agent.Agent
	if strings.HasPrefix(session.AgentType, "adl:") {
		defName := strings.TrimPrefix(session.AgentType, "adl:")
		defs, loadErr := store.LoadADLDefinitions()
		if loadErr != nil {
			http.Error(w, fmt.Sprintf("failed to load ADL definitions: %v", loadErr), http.StatusInternalServerError)
			return
		}
		var found bool
		for _, def := range defs {
			if def.Name == defName {
				ag = agent.NewADLAgent(def, session.ID, extensionManager)
				found = true
				break
			}
		}
		if !found {
			http.Error(w, fmt.Sprintf("ADL definition %q not found", defName), http.StatusNotFound)
			return
		}
	} else {
		var err error
		ag, err = extensionManager.GetAgent(session.ID, session.AgentType, session.WorkingDir, session.AgentConfig)
		if err != nil {
			http.Error(w, fmt.Sprintf("agent unavailable: %v", err), http.StatusServiceUnavailable)
			return
		}
	}

	isADL := strings.HasPrefix(session.AgentType, "adl:")
	events := make(chan agent.Event, 64)

	go func() {
		defer close(events)
		runReq := agent.RunRequest{
			WorkingDir: session.WorkingDir,
			Message:    req.Message,
		}
		if !isADL {
			runReq.SessionID = agentSessionID
		}
		err := ag.Run(r.Context(), runReq, events)
		if err != nil && r.Context().Err() == nil {
			events <- agent.Event{Type: agent.EventError, Error: err.Error()}
		}
	}()

	var assistantContent strings.Builder
	var newAgentSessionID string

	for ev := range events {
		data, _ := json.Marshal(ev)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		switch ev.Type {
		case agent.EventText:
			assistantContent.WriteString(ev.Content)
		case agent.EventDone:
			newAgentSessionID = ev.SessionID
		}
	}

	if assistantContent.Len() > 0 {
		assistantMsg := model.ChatMessage{
			ID:        uuid.NewString(),
			Role:      "assistant",
			Content:   assistantContent.String(),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		mu.Lock()
		sessionMessages[sessionID] = append(sessionMessages[sessionID], assistantMsg)
		if newAgentSessionID != "" && !isADL {
			agentSessions[sessionID] = newAgentSessionID
		}
		snapshot := snapshotData()
		mu.Unlock()
		if newAgentSessionID != "" && !isADL {
			if err := store.SaveData(snapshot); err != nil {
				fmt.Fprintf(os.Stderr, "warn: save session: %v\n", err)
			}
		}
	}
}

func handleSessionHistory(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
	if strings.HasPrefix(session.AgentType, "adl:") {
		writeJSON(w, http.StatusOK, []model.ChatMessage{})
		return
	}
	var msgs []model.ChatMessage
	var err error
	if session.AgentType == "pi" {
		msgs, err = store.LoadPiHistory(session.WorkingDir, agentSessionID)
	} else {
		msgs, err = store.LoadClaudeHistory(session.WorkingDir, agentSessionID)
	}
	if err != nil {
		http.Error(w, "failed to load history", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	caps := Capabilities{
		Sandbox: SandboxCapabilities{
			Bwrap: agent.GetBwrapStatus(),
		},
	}
	writeJSON(w, http.StatusOK, caps)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
