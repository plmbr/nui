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
}

type AppConfig struct {
	CopilotKitPublicApiKey string `json:"copilotKitPublicApiKey"`
	CopilotKitRuntimeURL   string `json:"copilotKitRuntimeUrl"`
}


var (
	mu              sync.RWMutex
	projects        []model.Project
	projectMessages = map[string][]model.ChatMessage{}
	projectSessions = map[string]string{} // project ID → claude session ID
)

func initStore() error {
	data, err := store.LoadData()
	if err != nil {
		return err
	}
	mu.Lock()
	projects = data.Projects
	projectSessions = data.Sessions
	mu.Unlock()
	return nil
}

// snapshotData must be called with mu held.
func snapshotData() store.Data {
	ps := make([]model.Project, len(projects))
	copy(ps, projects)
	ss := make(map[string]string, len(projectSessions))
	for k, v := range projectSessions {
		ss[k] = v
	}
	return store.Data{Projects: ps, Sessions: ss}
}

func registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/projects", handleProjects)
	mux.HandleFunc("/api/projects/", handleProject)
	mux.HandleFunc("/api/agent-types", handleAgentTypes)
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/settings", handleSettings)
}

func handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		list := make([]model.Project, len(projects))
		copy(list, projects)
		mu.RUnlock()
		writeJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var req struct {
			Name       string `json:"name"`
			WorkingDir string `json:"workingDir"`
			AgentType  string `json:"agentType"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.AgentType == "" {
			http.Error(w, "name and agentType are required", http.StatusBadRequest)
			return
		}
		p := model.Project{
			ID:         uuid.NewString(),
			Name:       req.Name,
			WorkingDir: req.WorkingDir,
			AgentType:  req.AgentType,
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		}
		mu.Lock()
		projects = append(projects, p)
		snapshot := snapshotData()
		mu.Unlock()
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save data after create: %v\n", err)
		}
		writeJSON(w, http.StatusCreated, p)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, agentTypes)
}

func findProject(id string) (model.Project, bool) {
	for _, p := range projects {
		if p.ID == id {
			return p, true
		}
	}
	return model.Project{}, false
}

func deleteProject(id string) bool {
	for i, p := range projects {
		if p.ID == id {
			projects = append(projects[:i], projects[i+1:]...)
			return true
		}
	}
	return false
}

func renameProject(id, name string) bool {
	for i, p := range projects {
		if p.ID == id {
			projects[i].Name = name
			return true
		}
	}
	return false
}

func handleProject(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	// Route /api/projects/:id/<sub>
	if idx := strings.Index(path, "/"); idx != -1 {
		id := path[:idx]
		rest := path[idx+1:]
		switch rest {
		case "messages":
			handleProjectMessages(w, r, id)
		case "chat":
			handleProjectChat(w, r, id)
		case "history":
			handleProjectHistory(w, r, id)
		default:
			http.NotFound(w, r)
		}
		return
	}

	id := path
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		p, ok := findProject(id)
		mu.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, p)

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
		found := renameProject(id, name)
		var updated model.Project
		var snapshot store.Data
		if found {
			updated, _ = findProject(id)
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
		var sessionID, workingDir, agentType string
		if p, ok := findProject(id); ok {
			sessionID = projectSessions[id]
			workingDir = p.WorkingDir
			agentType = p.AgentType
		}
		removed := deleteProject(id)
		if removed {
			delete(projectSessions, id)
			delete(projectMessages, id)
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
		if sessionID != "" {
			var delErr error
			if agentType == "pi" {
				delErr = store.DeletePiSession(workingDir, sessionID)
			} else {
				delErr = store.DeleteClaudeSession(workingDir, sessionID)
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

func handleProjectMessages(w http.ResponseWriter, r *http.Request, projectID string) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		msgs := projectMessages[projectID]
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
		projectMessages[projectID] = msgs
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

func handleProjectChat(w http.ResponseWriter, r *http.Request, projectID string) {
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
	project, ok := findProject(projectID)
	sessionID := projectSessions[projectID]
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
	projectMessages[projectID] = append(projectMessages[projectID], userMsg)
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

	ag, err := extensionManager.GetAgent(project.AgentType)
	if err != nil {
		http.Error(w, fmt.Sprintf("agent unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}

	events := make(chan agent.Event, 64)

	go func() {
		defer close(events)
		err := ag.Run(r.Context(), agent.RunRequest{
			SessionID:  sessionID,
			WorkingDir: project.WorkingDir,
			Message:    req.Message,
		}, events)
		if err != nil && r.Context().Err() == nil {
			events <- agent.Event{Type: agent.EventError, Error: err.Error()}
		}
	}()

	var assistantContent strings.Builder
	var newSessionID string

	for ev := range events {
		data, _ := json.Marshal(ev)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		switch ev.Type {
		case agent.EventText:
			assistantContent.WriteString(ev.Content)
		case agent.EventDone:
			newSessionID = ev.SessionID
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
		projectMessages[projectID] = append(projectMessages[projectID], assistantMsg)
		if newSessionID != "" {
			projectSessions[projectID] = newSessionID
		}
		snapshot := snapshotData()
		mu.Unlock()
		if newSessionID != "" {
			if err := store.SaveData(snapshot); err != nil {
				fmt.Fprintf(os.Stderr, "warn: save session: %v\n", err)
			}
		}
	}
}

func handleProjectHistory(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mu.RLock()
	project, ok := findProject(projectID)
	sessionID := projectSessions[projectID]
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	var msgs []model.ChatMessage
	var err error
	if project.AgentType == "pi" {
		msgs, err = store.LoadPiHistory(project.WorkingDir, sessionID)
	} else {
		msgs, err = store.LoadClaudeHistory(project.WorkingDir, sessionID)
	}
	if err != nil {
		http.Error(w, "failed to load history", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
