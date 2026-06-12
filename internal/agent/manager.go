// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// builtinExtensions maps agent type IDs to their Python script filenames.
var builtinExtensions = map[string]string{
	"claude-code": "claude_code.py",
	"pi":          "pi.py",
}

// Manager launches and reconnects to extension processes.
type Manager struct {
	extensionsDir string
	mu            sync.Mutex        // protects processes
	agentMu       sync.Map          // map[agentType]*sync.Mutex — serialises per-agent launch
	processes     map[string]*os.Process
}

func NewManager(extensionsDir string) *Manager {
	return &Manager{
		extensionsDir: extensionsDir,
		processes:     make(map[string]*os.Process),
	}
}

// GetAgent returns an ExtensionAgent for the given project, starting the extension if needed.
func (m *Manager) GetAgent(projectID, agentType string) (Agent, error) {
	script, ok := builtinExtensions[agentType]
	if !ok {
		return nil, fmt.Errorf("unknown agent type: %q", agentType)
	}
	conn, err := m.ensureRunning(projectID, script)
	if err != nil {
		return nil, err
	}
	return NewExtensionAgent(agentType, conn), nil
}

// PrewarmProjects starts extension processes for each project in the background.
func (m *Manager) PrewarmProjects(projects []PrewarmEntry) {
	for _, p := range projects {
		go func(projectID, agentType string) {
			script, ok := builtinExtensions[agentType]
			if !ok {
				return
			}
			if _, err := m.ensureRunning(projectID, script); err != nil {
				fmt.Fprintf(os.Stderr, "warn: prewarm project %s: %v\n", projectID, err)
			}
		}(p.ProjectID, p.AgentType)
	}
}

// PrewarmEntry holds the project ID and agent type for prewarming.
type PrewarmEntry struct {
	ProjectID string
	AgentType string
}

// Stop sends SIGTERM to the extension process for a specific project.
func (m *Manager) Stop(projectID string) {
	m.mu.Lock()
	proc, ok := m.processes[projectID]
	m.mu.Unlock()
	if ok {
		proc.Signal(syscall.SIGTERM)
	}
	m.deleteConnectionFile(projectID)
}

// StopAll sends SIGTERM to all managed extension processes and removes their connection files.
func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.processes))
	for id, proc := range m.processes {
		proc.Signal(syscall.SIGTERM)
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.deleteConnectionFile(id)
	}
}

// ensureRunning returns a live connection, launching the extension if needed.
// A per-project mutex prevents concurrent launches for the same project.
func (m *Manager) ensureRunning(projectID, script string) (ConnectionInfo, error) {
	// Fast path without the per-project lock.
	if conn, err := m.readConnectionFile(projectID); err == nil && isAlive(conn.PID) {
		return conn, nil
	}

	// Acquire the per-project launch lock to serialise launch attempts.
	v, _ := m.agentMu.LoadOrStore(projectID, &sync.Mutex{})
	agLock := v.(*sync.Mutex)
	agLock.Lock()
	defer agLock.Unlock()

	// Re-check now that we hold the lock; another goroutine may have launched it.
	if conn, err := m.readConnectionFile(projectID); err == nil && isAlive(conn.PID) {
		return conn, nil
	}
	return m.launch(projectID, script)
}

func (m *Manager) launch(projectID, script string) (ConnectionInfo, error) {
	// Remove a stale connection file so waitForConnection cannot return it.
	m.deleteConnectionFile(projectID)

	scriptPath := filepath.Join(m.extensionsDir, script)
	cmd := exec.Command("python3", scriptPath, "--project-id", projectID)
	cmd.Stderr = os.Stderr
	// Run the extension in its own session so terminal signals (SIGINT, SIGHUP)
	// do not propagate from the Go server's process group to the extension.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return ConnectionInfo{}, fmt.Errorf("launch extension for project %s: %w", projectID, err)
	}

	m.mu.Lock()
	m.processes[projectID] = cmd.Process
	m.mu.Unlock()

	conn, err := m.waitForConnection(projectID, 15*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return ConnectionInfo{}, fmt.Errorf("extension for project %s failed to start: %w", projectID, err)
	}

	go func() {
		cmd.Wait()
		m.mu.Lock()
		delete(m.processes, projectID)
		m.mu.Unlock()
	}()

	return conn, nil
}

func (m *Manager) waitForConnection(projectID string, timeout time.Duration) (ConnectionInfo, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn, err := m.readConnectionFile(projectID); err == nil {
			return conn, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return ConnectionInfo{}, fmt.Errorf("timed out waiting for extension for project %s to start", projectID)
}

func (m *Manager) readConnectionFile(projectID string) (ConnectionInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return ConnectionInfo{}, fmt.Errorf("get home dir: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".loop", "extensions", projectID+".json"))
	if err != nil {
		return ConnectionInfo{}, err
	}
	var conn ConnectionInfo
	return conn, json.Unmarshal(data, &conn)
}

func (m *Manager) deleteConnectionFile(projectID string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	os.Remove(filepath.Join(home, ".loop", "extensions", projectID+".json"))
}

func isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
