// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"loop/internal/model"
	"loop/internal/store"

	"github.com/google/uuid"
)

// StartOptions configures optional CLI-driven session bootstrap on server start.
type StartOptions struct {
	AgentType  string
	Prompt     string
	WorkingDir string
	Open       bool // open the UI in the system default browser
}

type bootstrapState struct {
	SessionID     string `json:"sessionId,omitempty"`
	InitialPrompt string `json:"initialPrompt,omitempty"`
}

var (
	bootstrapMu sync.Mutex
	bootstrap   bootstrapState
)

func setBootstrap(sessionID, prompt string) {
	bootstrapMu.Lock()
	bootstrap.SessionID = sessionID
	bootstrap.InitialPrompt = prompt
	bootstrapMu.Unlock()
}

func takeBootstrap() bootstrapState {
	bootstrapMu.Lock()
	defer bootstrapMu.Unlock()
	out := bootstrap
	bootstrap = bootstrapState{}
	return out
}

func handleBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, takeBootstrap())
}

// bootstrapFromCLI creates a session when loop ui is started with --agent-type.
func bootstrapFromCLI(opts StartOptions) error {
	agentType := strings.TrimSpace(opts.AgentType)
	if agentType == "" {
		return nil
	}

	def, ok := findADLDef(agentType)
	if !ok {
		return fmt.Errorf("unknown agent type %q", agentType)
	}

	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		}
	}

	s, err := createSession(def.Name, workingDir, def.Name, nil)
	if err != nil {
		return err
	}

	prompt := strings.TrimSpace(opts.Prompt)
	if prompt != "" {
		setBootstrap(s.ID, prompt)
	}

	settings, err := store.LoadSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: load settings for bootstrap: %v\n", err)
		settings = store.Settings{Theme: "light"}
	}
	settings.LastAgentType = def.Name
	settings.LastSessionID = s.ID
	if err := store.SaveSettings(settings); err != nil {
		fmt.Fprintf(os.Stderr, "warn: save settings after bootstrap: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Created session %q (%s) with agent %s\n", s.Name, s.ID, def.Name)
	return nil
}

func createSession(name, workingDir, agentType string, agentConfig map[string]any) (model.Session, error) {
	if name == "" || agentType == "" {
		return model.Session{}, fmt.Errorf("name and agentType are required")
	}
	def, ok := findADLDef(agentType)
	if !ok {
		return model.Session{}, fmt.Errorf("unknown agent type: %s", agentType)
	}

	s := model.Session{
		ID:          uuid.NewString(),
		Name:        name,
		WorkingDir:  workingDir,
		AgentType:   def.Name,
		AgentConfig: agentConfig,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := validateSessionConnector(s); err != nil {
		extensionManager.Stop(s.ID)
		return model.Session{}, err
	}

	mu.Lock()
	sessions = append(sessions, s)
	snapshot := snapshotData()
	mu.Unlock()
	if err := store.SaveData(snapshot); err != nil {
		fmt.Fprintf(os.Stderr, "warn: save data after create: %v\n", err)
	}
	return s, nil
}
