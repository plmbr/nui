// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// builtinExtensions maps agent type IDs to their Python script filenames.
var builtinExtensions = map[string]string{
	"claude-code": "claude_code.py",
	"pi":          "pi.py",
}

// Manager launches and reconnects to extension processes, Docker containers, and remote agents.
type Manager struct {
	extensionsDir string
	mu            sync.Mutex  // protects processes
	containerMu   sync.Mutex  // protects containers and dockerURLs
	agentMu       sync.Map    // map[projectID]*sync.Mutex — serialises per-project launch
	processes     map[string]*os.Process
	containers    map[string]string // projectID → containerID
	dockerURLs    map[string]string // projectID → http base URL
}

func NewManager(extensionsDir string) *Manager {
	return &Manager{
		extensionsDir: extensionsDir,
		processes:     make(map[string]*os.Process),
		containers:    make(map[string]string),
		dockerURLs:    make(map[string]string),
	}
}

// GetAgent returns an Agent for the given project, starting or connecting as needed.
func (m *Manager) GetAgent(projectID, agentType string, config map[string]any) (Agent, error) {
	switch agentType {
	case "docker":
		baseURL, err := m.ensureDockerRunning(projectID, config)
		if err != nil {
			return nil, err
		}
		return NewHTTPExtensionAgent(agentType, baseURL), nil
	case "remote":
		baseURL, err := m.connectRemote(config)
		if err != nil {
			return nil, err
		}
		return NewHTTPExtensionAgent(agentType, baseURL), nil
	default:
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
}

// PrewarmProjects starts or connects to agents for each project in the background.
func (m *Manager) PrewarmProjects(projects []PrewarmEntry) {
	for _, p := range projects {
		go func(e PrewarmEntry) {
			// ADL projects are not managed by the extension manager — skip silently.
			if strings.HasPrefix(e.AgentType, "adl:") {
				return
			}
			if _, err := m.GetAgent(e.ProjectID, e.AgentType, e.AgentConfig); err != nil {
				fmt.Fprintf(os.Stderr, "warn: prewarm project %s: %v\n", e.ProjectID, err)
			}
		}(p)
	}
}

// PrewarmEntry holds the project ID, agent type, and config for prewarming.
type PrewarmEntry struct {
	ProjectID   string
	AgentType   string
	AgentConfig map[string]any
}

// Stop terminates the extension process or Docker container for a specific project.
func (m *Manager) Stop(projectID string) {
	m.mu.Lock()
	proc, ok := m.processes[projectID]
	m.mu.Unlock()
	if ok {
		proc.Signal(syscall.SIGTERM)
	}

	m.containerMu.Lock()
	containerID, hasContainer := m.containers[projectID]
	if hasContainer {
		delete(m.containers, projectID)
	}
	m.containerMu.Unlock()
	if hasContainer {
		exec.Command("docker", "stop", containerID).Run()
	}

	m.deleteConnectionFile(projectID)
}

// StopAll terminates all managed extension processes and Docker containers.
func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.processes))
	for id, proc := range m.processes {
		proc.Signal(syscall.SIGTERM)
		ids = append(ids, id)
	}
	m.mu.Unlock()

	m.containerMu.Lock()
	for id, containerID := range m.containers {
		go exec.Command("docker", "stop", containerID).Run()
		delete(m.containers, id)
	}
	m.containerMu.Unlock()

	for _, id := range ids {
		m.deleteConnectionFile(id)
	}
}

// ── Python extension ─────────────────────────────────────────────────────────

func (m *Manager) ensureRunning(projectID, script string) (ConnectionInfo, error) {
	if conn, err := m.readConnectionFile(projectID); err == nil && isAlive(conn.PID) {
		return conn, nil
	}

	v, _ := m.agentMu.LoadOrStore(projectID, &sync.Mutex{})
	agLock := v.(*sync.Mutex)
	agLock.Lock()
	defer agLock.Unlock()

	if conn, err := m.readConnectionFile(projectID); err == nil && isAlive(conn.PID) {
		return conn, nil
	}
	return m.launch(projectID, script)
}

func (m *Manager) launch(projectID, script string) (ConnectionInfo, error) {
	m.deleteConnectionFile(projectID)

	scriptPath := filepath.Join(m.extensionsDir, script)
	cmd := exec.Command("python3", scriptPath, "--project-id", projectID)
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	env := os.Environ()
	if s := GetBwrapStatus(); s.Available {
		env = append(env, "LOOP_BWRAP_PATH="+s.Path)
	}
	cmd.Env = env

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

// ── Docker agent ─────────────────────────────────────────────────────────────

func (m *Manager) ensureDockerRunning(projectID string, config map[string]any) (string, error) {
	m.containerMu.Lock()
	baseURL, exists := m.dockerURLs[projectID]
	m.containerMu.Unlock()
	if exists && isHTTPAlive(baseURL) {
		return baseURL, nil
	}

	v, _ := m.agentMu.LoadOrStore(projectID, &sync.Mutex{})
	agLock := v.(*sync.Mutex)
	agLock.Lock()
	defer agLock.Unlock()

	m.containerMu.Lock()
	baseURL, exists = m.dockerURLs[projectID]
	m.containerMu.Unlock()
	if exists && isHTTPAlive(baseURL) {
		return baseURL, nil
	}
	return m.launchDocker(projectID, config)
}

func (m *Manager) launchDocker(projectID string, config map[string]any) (string, error) {
	image, _ := config["image"].(string)
	if image == "" {
		return "", fmt.Errorf("docker agent config missing 'image'")
	}
	containerPortF, _ := config["containerPort"].(float64)
	containerPort := int(containerPortF)
	if containerPort == 0 {
		return "", fmt.Errorf("docker agent config missing 'containerPort'")
	}

	// Stop any running container for this project first.
	m.containerMu.Lock()
	if oldID, ok := m.containers[projectID]; ok {
		go exec.Command("docker", "stop", oldID).Run()
		delete(m.containers, projectID)
	}
	delete(m.dockerURLs, projectID)
	m.containerMu.Unlock()

	out, err := exec.Command("docker", "run", "-d",
		"-p", fmt.Sprintf("127.0.0.1::%d", containerPort),
		image,
	).Output()
	if err != nil {
		return "", fmt.Errorf("docker run: %w", err)
	}
	containerID := strings.TrimSpace(string(out))

	m.containerMu.Lock()
	m.containers[projectID] = containerID
	m.containerMu.Unlock()

	portOut, err := exec.Command("docker", "port", containerID, strconv.Itoa(containerPort)).Output()
	if err != nil {
		exec.Command("docker", "stop", containerID).Run()
		return "", fmt.Errorf("docker port: %w", err)
	}

	hostPort, err := parseDockerPort(strings.TrimSpace(string(portOut)))
	if err != nil {
		exec.Command("docker", "stop", containerID).Run()
		return "", err
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", hostPort)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if isHTTPAlive(baseURL) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !isHTTPAlive(baseURL) {
		exec.Command("docker", "stop", containerID).Run()
		return "", fmt.Errorf("docker container for project %s timed out waiting on %s", projectID, baseURL)
	}

	m.containerMu.Lock()
	m.dockerURLs[projectID] = baseURL
	m.containerMu.Unlock()
	return baseURL, nil
}

func parseDockerPort(s string) (int, error) {
	// docker port outputs lines like "0.0.0.0:32768" or "127.0.0.1:32768".
	// Take the last line (IPv4/IPv6 may produce two lines).
	lines := strings.Split(s, "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" && len(lines) > 1 {
		last = strings.TrimSpace(lines[len(lines)-2])
	}
	idx := strings.LastIndex(last, ":")
	if idx < 0 {
		return 0, fmt.Errorf("unexpected docker port output: %q", s)
	}
	port, err := strconv.Atoi(last[idx+1:])
	if err != nil {
		return 0, fmt.Errorf("parse docker port %q: %w", s, err)
	}
	return port, nil
}

// ── Remote agent ─────────────────────────────────────────────────────────────

func (m *Manager) connectRemote(config map[string]any) (string, error) {
	host, _ := config["host"].(string)
	portF, _ := config["port"].(float64)
	port := int(portF)
	if host == "" || port == 0 {
		return "", fmt.Errorf("remote agent config requires 'host' and 'port'")
	}
	baseURL := fmt.Sprintf("http://%s:%d", host, port)
	if !isHTTPAlive(baseURL) {
		return "", fmt.Errorf("remote agent at %s is not reachable", baseURL)
	}
	return baseURL, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

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

func (m *Manager) writeConnectionFile(projectID string, conn ConnectionInfo) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dir := filepath.Join(home, ".loop", "extensions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(conn)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, projectID+".json"), data, 0644)
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

func isHTTPAlive(baseURL string) bool {
	cl := &http.Client{Timeout: time.Second}
	resp, err := cl.Get(baseURL + "/info")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}
