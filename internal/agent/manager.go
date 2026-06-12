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
	mu            sync.Mutex
	processes     map[string]*os.Process
}

func NewManager(extensionsDir string) *Manager {
	return &Manager{
		extensionsDir: extensionsDir,
		processes:     make(map[string]*os.Process),
	}
}

// GetAgent returns an ExtensionAgent for the given agent type, starting the extension if needed.
func (m *Manager) GetAgent(agentType string) (Agent, error) {
	script, ok := builtinExtensions[agentType]
	if !ok {
		return nil, fmt.Errorf("unknown agent type: %q", agentType)
	}
	conn, err := m.ensureRunning(agentType, script)
	if err != nil {
		return nil, err
	}
	return NewExtensionAgent(agentType, conn), nil
}

// Prewarm starts all built-in extensions in the background.
func (m *Manager) Prewarm() {
	for agentType, script := range builtinExtensions {
		go func(t, s string) {
			if _, err := m.ensureRunning(t, s); err != nil {
				fmt.Fprintf(os.Stderr, "warn: prewarm extension %s: %v\n", t, err)
			}
		}(agentType, script)
	}
}

// StopAll sends SIGTERM to all managed extension processes.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, proc := range m.processes {
		proc.Signal(syscall.SIGTERM)
	}
}

func (m *Manager) ensureRunning(agentType, script string) (ConnectionInfo, error) {
	if conn, err := m.readConnectionFile(agentType); err == nil && isAlive(conn.PID) {
		return conn, nil
	}
	return m.launch(agentType, script)
}

func (m *Manager) launch(agentType, script string) (ConnectionInfo, error) {
	scriptPath := filepath.Join(m.extensionsDir, script)
	cmd := exec.Command("python3", scriptPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return ConnectionInfo{}, fmt.Errorf("launch extension %s: %w", agentType, err)
	}

	m.mu.Lock()
	m.processes[agentType] = cmd.Process
	m.mu.Unlock()

	conn, err := m.waitForConnection(agentType, 15*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return ConnectionInfo{}, fmt.Errorf("extension %s failed to start: %w", agentType, err)
	}

	go func() {
		cmd.Wait()
		m.mu.Lock()
		delete(m.processes, agentType)
		m.mu.Unlock()
	}()

	return conn, nil
}

func (m *Manager) waitForConnection(agentType string, timeout time.Duration) (ConnectionInfo, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn, err := m.readConnectionFile(agentType); err == nil {
			return conn, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return ConnectionInfo{}, fmt.Errorf("timed out waiting for extension %s to start", agentType)
}

func (m *Manager) readConnectionFile(agentType string) (ConnectionInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return ConnectionInfo{}, fmt.Errorf("get home dir: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".loop", "extensions", agentType+".json"))
	if err != nil {
		return ConnectionInfo{}, err
	}
	var conn ConnectionInfo
	return conn, json.Unmarshal(data, &conn)
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
