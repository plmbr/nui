// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
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

const containerIdleTimeout = 30 * time.Minute

// Manager launches and reconnects to extension processes, Docker containers, and remote agents.
type Manager struct {
	extensionsDir string
	mu            sync.Mutex  // protects processes
	containerMu   sync.Mutex  // protects containers, dockerURLs, and lastActivity
	agentMu       sync.Map    // map[projectID]*sync.Mutex — serialises per-project launch
	processes     map[string]*os.Process
	containers    map[string]string // projectID → containerID
	dockerURLs    map[string]string // projectID → http base URL
	lastActivity  map[string]time.Time
}

func NewManager(extensionsDir string) *Manager {
	m := &Manager{
		extensionsDir: extensionsDir,
		processes:     make(map[string]*os.Process),
		containers:    make(map[string]string),
		dockerURLs:    make(map[string]string),
		lastActivity:  make(map[string]time.Time),
	}
	go m.idleReaper()
	return m
}

// idleReaper periodically stops Docker containers that have been idle too long.
func (m *Manager) idleReaper() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.containerMu.Lock()
		var toStop []struct{ projectID, containerID string }
		for projectID, t := range m.lastActivity {
			if time.Since(t) > containerIdleTimeout {
				if cid, ok := m.containers[projectID]; ok {
					toStop = append(toStop, struct{ projectID, containerID string }{projectID, cid})
					delete(m.containers, projectID)
					delete(m.dockerURLs, projectID)
					delete(m.lastActivity, projectID)
				}
			}
		}
		m.containerMu.Unlock()
		for _, entry := range toStop {
			fmt.Fprintf(os.Stderr, "info: stopping idle container for project %s\n", entry.projectID)
			exec.Command("docker", "stop", entry.containerID).Run()
		}
	}
}

// touchActivity records the current time as the last activity for a docker project.
func (m *Manager) touchActivity(projectID string) {
	m.containerMu.Lock()
	m.lastActivity[projectID] = time.Now()
	m.containerMu.Unlock()
}

// GetAgent returns an Agent for the given project, starting or connecting as needed.
func (m *Manager) GetAgent(projectID, agentType, workingDir string, config map[string]any) (Agent, error) {
	switch agentType {
	case "docker":
		baseURL, err := m.ensureDockerRunning(projectID, config)
		if err != nil {
			return nil, err
		}
		m.touchActivity(projectID)
		return NewHTTPExtensionAgent(agentType, baseURL), nil
	case "docker-claude":
		image, _ := config["image"].(string)
		if image == "" {
			image = "loop-claude-code:latest"
		}
		baseURL, err := m.ensureBuiltinDockerRunning(projectID, image, workingDir, ".claude")
		if err != nil {
			return nil, err
		}
		m.touchActivity(projectID)
		return NewHTTPExtensionAgent(agentType, baseURL), nil
	case "docker-pi":
		image, _ := config["image"].(string)
		if image == "" {
			image = "loop-pi:latest"
		}
		baseURL, err := m.ensureBuiltinDockerRunning(projectID, image, workingDir, "")
		if err != nil {
			return nil, err
		}
		m.touchActivity(projectID)
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

// isDockerOrRemote returns true for agent types that are launched lazily on first use.
func isDockerOrRemote(agentType string) bool {
	switch agentType {
	case "docker", "docker-claude", "docker-pi", "remote":
		return true
	}
	return false
}

// PrewarmProjects eagerly starts local extension processes. Docker and remote
// agents are skipped — they start lazily when a project is first used.
func (m *Manager) PrewarmProjects(projects []PrewarmEntry) {
	for _, p := range projects {
		go func(e PrewarmEntry) {
			if strings.HasPrefix(e.AgentType, "adl:") || isDockerOrRemote(e.AgentType) {
				return
			}
			m.GetAgent(e.ProjectID, e.AgentType, e.WorkingDir, e.AgentConfig) //nolint:errcheck
		}(p)
	}
}

// PrewarmEntry holds the project ID, agent type, working dir, and config for prewarming.
type PrewarmEntry struct {
	ProjectID   string
	AgentType   string
	WorkingDir  string
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
// It waits for all containers to stop before returning.
func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.processes))
	for id, proc := range m.processes {
		proc.Signal(syscall.SIGTERM)
		ids = append(ids, id)
	}
	m.mu.Unlock()

	m.containerMu.Lock()
	containerIDs := make([]string, 0, len(m.containers))
	for id, containerID := range m.containers {
		containerIDs = append(containerIDs, containerID)
		delete(m.containers, id)
	}
	m.containerMu.Unlock()

	var wg sync.WaitGroup
	for _, cid := range containerIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			exec.Command("docker", "stop", id).Run()
		}(cid)
	}
	wg.Wait()

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

// ── Builtin Docker agents (docker-claude, docker-pi) ─────────────────────────

const builtinContainerPort = 8090

func (m *Manager) ensureBuiltinDockerRunning(projectID, image, workingDir, agentConfigDir string) (string, error) {
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
	return m.launchBuiltinDocker(projectID, image, workingDir, agentConfigDir)
}

// loopbackAddHostArgs returns --add-host flags for any hostname in baseURL that
// resolves to a loopback address, so containers can reach host-local proxies.
func loopbackAddHostArgs(baseURL string) []string {
	if baseURL == "" {
		return nil
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" {
		return nil
	}
	hostname := u.Hostname()
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.IsLoopback() {
			return []string{"--add-host", hostname + ":host-gateway"}
		}
	}
	return nil
}

func (m *Manager) launchBuiltinDocker(projectID, image, workingDir, agentConfigDir string) (string, error) {
	m.containerMu.Lock()
	if oldID, ok := m.containers[projectID]; ok {
		go exec.Command("docker", "stop", oldID).Run()
		delete(m.containers, projectID)
	}
	delete(m.dockerURLs, projectID)
	m.containerMu.Unlock()

	home, _ := os.UserHomeDir()
	containerHome := "/home/loop"

	args := []string{"run", "-d", "--rm",
		"-p", fmt.Sprintf("127.0.0.1::%d", builtinContainerPort),
	}
	for _, envKey := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_OAUTH_TOKEN", "ANTHROPIC_BASE_URL"} {
		if val := os.Getenv(envKey); val != "" {
			args = append(args, "-e", envKey+"="+val)
		}
	}
	// If ANTHROPIC_BASE_URL points to a loopback hostname, route it to the host machine.
	if extraHosts := loopbackAddHostArgs(os.Getenv("ANTHROPIC_BASE_URL")); len(extraHosts) > 0 {
		args = append(args, extraHosts...)
	}
	if workingDir != "" {
		args = append(args, "-v", workingDir+":"+workingDir)
	}
	if agentConfigDir != "" {
		hostConfigDir := filepath.Join(home, agentConfigDir)
		os.MkdirAll(hostConfigDir, 0700) //nolint:errcheck
		args = append(args, "-v", hostConfigDir+":"+containerHome+"/"+agentConfigDir)
		// Mount the top-level config JSON file (e.g. ~/.claude.json) if it exists.
		hostConfigJSON := filepath.Join(home, agentConfigDir+".json")
		if _, statErr := os.Stat(hostConfigJSON); statErr == nil {
			args = append(args, "-v", hostConfigJSON+":"+containerHome+"/"+agentConfigDir+".json")
		}
		// Shadow the host's settings.json with an empty one so that host-specific
		// env overrides (e.g. ANTHROPIC_BASE_URL pointing to localhost) do not
		// break network connectivity inside the container.
		overridePath := filepath.Join(home, ".loop", agentConfigDir+"-settings-override.json")
		if writeErr := os.WriteFile(overridePath, []byte("{}"), 0644); writeErr == nil {
			args = append(args, "-v", overridePath+":"+containerHome+"/"+agentConfigDir+"/settings.json:ro")
		}
	}
	args = append(args, image)

	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		return "", fmt.Errorf("docker run %s: %w", image, err)
	}
	containerID := strings.TrimSpace(string(out))

	m.containerMu.Lock()
	m.containers[projectID] = containerID
	m.containerMu.Unlock()

	portOut, err := exec.Command("docker", "port", containerID, strconv.Itoa(builtinContainerPort)).Output()
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
		return "", fmt.Errorf("container %s for project %s did not become ready on %s", image, projectID, baseURL)
	}

	m.containerMu.Lock()
	m.dockerURLs[projectID] = baseURL
	m.containerMu.Unlock()
	return baseURL, nil
}

// ── User-configured Docker agent ─────────────────────────────────────────────

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

	out, err := exec.Command("docker", "run", "-d", "--rm",
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
