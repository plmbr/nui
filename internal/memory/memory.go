// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"loop/internal/store"
)

const (
	userFileName = "user.md"
	agentsSubdir = "agents"

	ModeAuto     = "auto"
	ModeManual   = "manual"
	ModeDisabled = "disabled"

	EnvLoopMemoryAgentID   = "LOOP_MEMORY_AGENT_ID"
	EnvLoopMemoryUserMode  = "LOOP_MEMORY_USER_MODE"
	EnvLoopMemoryAgentMode = "LOOP_MEMORY_AGENT_MODE"
)

var agentIDSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Dir returns ~/.loop/memory, creating it if needed.
func Dir() (string, error) {
	base, err := store.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "memory")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// UserPath returns ~/.loop/memory/user.md.
func UserPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, userFileName), nil
}

// AgentPath returns ~/.loop/memory/agents/<agent-id>.md.
func AgentPath(agentID string) (string, error) {
	safe, err := sanitizeAgentID(agentID)
	if err != nil {
		return "", err
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	agentsDir := filepath.Join(dir, agentsSubdir)
	if err := os.MkdirAll(agentsDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(agentsDir, safe+".md"), nil
}

func sanitizeAgentID(agentID string) (string, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return "", fmt.Errorf("agent id is required")
	}
	safe := strings.Trim(agentIDSanitizer.ReplaceAllString(agentID, "-"), "-")
	if safe == "" {
		return "", fmt.Errorf("invalid agent id %q", agentID)
	}
	return safe, nil
}

func normalizeMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case ModeAuto:
		return ModeAuto
	case ModeDisabled:
		return ModeDisabled
	default:
		return ModeManual
	}
}

// UserMode returns the effective user memory mode (default manual).
func UserMode(s store.Settings) string {
	if mode := strings.TrimSpace(s.MemoryUserMode); mode != "" {
		return normalizeMode(mode)
	}
	if s.MemoryUserEnabled != nil && !*s.MemoryUserEnabled {
		return ModeDisabled
	}
	return ModeManual
}

// AgentMode returns the effective agent memory mode for agentID (default manual).
func AgentMode(s store.Settings, agentID string) string {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return ModeDisabled
	}
	if s.MemoryAgentsMode != nil {
		if mode, ok := s.MemoryAgentsMode[agentID]; ok && strings.TrimSpace(mode) != "" {
			return normalizeMode(mode)
		}
	}
	if s.MemoryAgentsEnabled != nil {
		if enabled, ok := s.MemoryAgentsEnabled[agentID]; ok && !enabled {
			return ModeDisabled
		}
	}
	return ModeManual
}

// InjectionEnabled reports whether memory content should be injected for a mode.
func InjectionEnabled(mode string) bool {
	return normalizeMode(mode) != ModeDisabled
}

// SavingEnabled reports whether memory writes are allowed for a mode.
func SavingEnabled(mode string) bool {
	return normalizeMode(mode) != ModeDisabled
}

// RememberSkillNeeded reports whether the remember skill should be attached.
func RememberSkillNeeded(s store.Settings, agentID string) bool {
	return InjectionEnabled(UserMode(s)) || InjectionEnabled(AgentMode(s, agentID))
}

// AutoSavePromptNeeded reports whether proactive auto-save instructions should be injected.
func AutoSavePromptNeeded(s store.Settings, agentID string) bool {
	return UserMode(s) == ModeAuto || AgentMode(s, agentID) == ModeAuto
}

// CanWriteScope reports whether update_memory is allowed for the given scope and settings.
func CanWriteScope(scope string, s store.Settings, agentID string) bool {
	switch strings.TrimSpace(strings.ToLower(scope)) {
	case "user":
		return SavingEnabled(UserMode(s))
	case "agent":
		return SavingEnabled(AgentMode(s, agentID))
	default:
		return false
	}
}

// ReadUser returns user memory content, or empty string when the file does not exist.
func ReadUser() (string, error) {
	return currentStore.ReadUser()
}

// ReadAgent returns agent memory content, or empty string when the file does not exist.
func ReadAgent(agentID string) (string, error) {
	return currentStore.ReadAgent(agentID)
}

func readFile(pathFn func() (string, error)) (string, error) {
	path, err := pathFn()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteUser writes user memory atomically.
func WriteUser(content string) error {
	return currentStore.WriteUser(content)
}

// WriteAgent writes agent memory atomically.
func WriteAgent(agentID, content string) error {
	return currentStore.WriteAgent(agentID, content)
}

func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	body := strings.TrimSpace(content)
	if body != "" {
		body += "\n"
	}
	tmp, err := os.CreateTemp(dir, ".memory-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// Update applies replace or append mode to user or agent memory.
func Update(scope, agentID, content, writeMode string) (string, error) {
	return currentStore.Update(scope, agentID, content, writeMode)
}

// PromptAppendix returns memory file sections to append to the system prompt.
func PromptAppendix(settings store.Settings, agentID string) string {
	var blocks []string
	if InjectionEnabled(UserMode(settings)) {
		if body, err := ReadUser(); err == nil && body != "" {
			blocks = append(blocks, "## User memory\n\n"+body)
		}
	}
	if InjectionEnabled(AgentMode(settings, agentID)) {
		if body, err := ReadAgent(agentID); err == nil && body != "" {
			blocks = append(blocks, "## Agent memory\n\n"+body)
		}
	}
	return strings.Join(blocks, "\n\n")
}

// AutoSaveAppendix returns proactive save instructions when any layer is in auto mode.
func AutoSaveAppendix(settings store.Settings, agentID string) string {
	if !AutoSavePromptNeeded(settings, agentID) {
		return ""
	}
	var scopes []string
	if UserMode(settings) == ModeAuto {
		scopes = append(scopes, "user scope for cross-agent preferences")
	}
	if AgentMode(settings, agentID) == ModeAuto {
		scopes = append(scopes, "agent scope for this agent's workflow")
	}
	scopeHint := "agent scope by default (including choices and decisions); user scope only for cross-agent personal preferences"
	if len(scopes) == 1 {
		scopeHint = scopes[0]
	}
	return strings.TrimSpace(`## Memory (auto-save)

When you make a durable decision or learn a fact that should persist, call **update_memory** on the **loop-agent** MCP server (` + scopeHint + `). Use ` + "`mode: append`" + `. Do not wait for the user to ask.`)
}

// AgentEntry describes one agent memory file.
type AgentEntry struct {
	AgentID   string `json:"agentId"`
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Mode      string `json:"mode"`
}

// UserInfo describes user memory metadata.
type UserInfo struct {
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Mode      string `json:"mode"`
}

// Summary is returned by GET /api/memory.
type Summary struct {
	User   UserInfo     `json:"user"`
	Agents []AgentEntry `json:"agents"`
}

// ListSummary returns memory metadata for the UI.
func ListSummary(settings store.Settings) (Summary, error) {
	out := Summary{
		User:   UserInfo{Mode: UserMode(settings)},
		Agents: []AgentEntry{},
	}
	userPath, err := UserPath()
	if err != nil {
		return out, err
	}
	if info, err := fileInfo(userPath); err == nil && info != nil {
		out.User.Size = info.size
		out.User.UpdatedAt = info.updatedAt
	}
	dir, err := Dir()
	if err != nil {
		return out, err
	}
	agentsDir := filepath.Join(dir, agentsSubdir)
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		agentID := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(agentsDir, entry.Name())
		item := AgentEntry{
			AgentID: agentID,
			Mode:    AgentMode(settings, agentID),
		}
		if info, err := fileInfo(path); err == nil && info != nil {
			item.Size = info.size
			item.UpdatedAt = info.updatedAt
		}
		out.Agents = append(out.Agents, item)
	}
	return out, nil
}

type fileMeta struct {
	size      int64
	updatedAt string
}

func fileInfo(path string) (*fileMeta, error) {
	st, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &fileMeta{
		size:      st.Size(),
		updatedAt: st.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

// DeleteAgent removes an agent memory file.
func DeleteAgent(agentID string) error {
	return currentStore.DeleteAgent(agentID)
}

// DeleteUser removes user memory file.
func DeleteUser() error {
	return currentStore.DeleteUser()
}

// AgentIDFromEnv returns LOOP_MEMORY_AGENT_ID when set.
func AgentIDFromEnv() string {
	return strings.TrimSpace(os.Getenv(EnvLoopMemoryAgentID))
}

// UserModeFromEnv returns LOOP_MEMORY_USER_MODE when set.
func UserModeFromEnv() string {
	return normalizeMode(os.Getenv(EnvLoopMemoryUserMode))
}

// AgentModeFromEnv returns LOOP_MEMORY_AGENT_MODE when set.
func AgentModeFromEnv() string {
	return normalizeMode(os.Getenv(EnvLoopMemoryAgentMode))
}
