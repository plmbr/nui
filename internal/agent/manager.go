// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
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
	"time"

	"nui/internal/devcontainer"
	"nui/internal/extensions"
)

// builtinAgentTypes are harness types implemented in-process inside the nui binary.
var builtinAgentTypes = map[string]bool{
	"claude-code": true,
	"pi":          true,
	"codex":       true,
	"opencode":    true,
}

const containerIdleTimeout = 30 * time.Minute

// Manager launches in-process harness agents, Docker containers, and remote agents.
type Manager struct {
	registry       *extensions.Registry
	testHarnessRun HarnessRunHook
	builtinMu      sync.Mutex
	builtinAgents  map[string]Agent // projectID + harnessType → agent
	extMu         sync.Mutex
	extAgents     map[string]Agent // projectID + harnessID → extension harness agent
	containerMu   sync.Mutex       // protects containers, dockerURLs, and lastActivity
	agentMu       sync.Map         // map[projectID]*sync.Mutex — serialises per-project launch
	containers    map[string]string // projectID → containerID
	devcontainerDirs map[string]string // projectID → nui-managed devcontainer up folder
	dockerURLs    map[string]string // projectID → http base URL
	lastActivity  map[string]time.Time
}

// SetExtensionRegistry attaches the loaded extension registry.
func (m *Manager) SetExtensionRegistry(reg *extensions.Registry) {
	m.registry = reg
}

func NewManager() *Manager {
	m := &Manager{
		builtinAgents: make(map[string]Agent),
		extAgents:     make(map[string]Agent),
		containers:       make(map[string]string),
		devcontainerDirs: make(map[string]string),
		dockerURLs:       make(map[string]string),
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
		var toStop []struct {
			projectID   string
			containerID string
			baseURL     string
		}
		for projectID, t := range m.lastActivity {
			if time.Since(t) > containerIdleTimeout {
				if cid, ok := m.containers[projectID]; ok {
					toStop = append(toStop, struct {
						projectID   string
						containerID string
						baseURL     string
					}{projectID, cid, m.dockerURLs[projectID]})
					delete(m.containers, projectID)
					delete(m.devcontainerDirs, projectID)
					delete(m.dockerURLs, projectID)
					delete(m.lastActivity, projectID)
				}
			}
		}
		m.containerMu.Unlock()
		for _, entry := range toStop {
			fmt.Fprintf(os.Stderr, "info: stopping idle container for project %s\n", entry.projectID)
			if entry.baseURL != "" {
				ShutdownHTTPAgent(entry.baseURL)
			}
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

// GetClaudeCodeDocker launches (or reuses) a Docker container running the claude-code HTTP agent.
// When sessionConfigDir is set, ADL-provisioned ai assets are bind-mounted into the container.
func (m *Manager) GetClaudeCodeDocker(projectID, image, workingDir, sessionConfigDir string, userScope bool) (Agent, error) {
	if image == "" {
		image = "nui-claude-code:latest"
	}
	baseURL, err := m.ensureBuiltinDockerRunning(projectID, image, workingDir, "claude-code", sessionConfigDir, ".claude", !userScope, userScope)
	if err != nil {
		return nil, err
	}
	m.touchActivity(projectID)
	return NewHTTPExtensionAgent("claude-code-docker", baseURL), nil
}

// GetPiDocker launches (or reuses) a Docker container running the pi HTTP agent.
// The working dir and ~/.pi/agent/sessions are mounted.
func (m *Manager) GetPiDocker(projectID, image, workingDir, sessionConfigDir string) (Agent, error) {
	if image == "" {
		image = "nui-pi:latest"
	}
	home, _ := os.UserHomeDir()
	piSessions := filepath.Join(home, ".pi", "agent", "sessions")
	os.MkdirAll(piSessions, 0755) //nolint:errcheck
	baseURL, err := m.ensureBuiltinDockerRunning(projectID, image, workingDir, "pi", sessionConfigDir, "", false, false,
		piSessions+":/home/nui/.pi/agent/sessions")
	if err != nil {
		return nil, err
	}
	m.touchActivity(projectID)
	return NewHTTPExtensionAgent("pi-docker", baseURL), nil
}

// GetOpenCodeDocker launches (or reuses) a Docker container running the opencode HTTP agent.
func (m *Manager) GetOpenCodeDocker(projectID, image, workingDir, sessionConfigDir string) (Agent, error) {
	if image == "" {
		image = "nui-opencode:latest"
	}
	home, _ := os.UserHomeDir()
	ocSessions := filepath.Join(home, ".nui", "opencode-sessions")
	os.MkdirAll(ocSessions, 0755) //nolint:errcheck
	baseURL, err := m.ensureBuiltinDockerRunning(projectID, image, workingDir, "opencode", sessionConfigDir, "", false, false,
		ocSessions+":/home/nui/.local/share/opencode")
	if err != nil {
		return nil, err
	}
	m.touchActivity(projectID)
	return NewHTTPExtensionAgent("opencode-docker", baseURL), nil
}

// GetCodexDocker launches (or reuses) a Docker container running the codex HTTP agent.
// Auth is forwarded via ANTHROPIC_API_KEY / ANTHROPIC_BASE_URL from the host environment.
func (m *Manager) GetCodexDocker(projectID, image, workingDir, sessionConfigDir string, userScope bool) (Agent, error) {
	if image == "" {
		image = "nui-codex:latest"
	}
	baseURL, err := m.ensureBuiltinDockerRunning(projectID, image, workingDir, "codex", sessionConfigDir, ".codex", false, userScope)
	if err != nil {
		return nil, err
	}
	m.touchActivity(projectID)
	return NewHTTPExtensionAgent("codex-docker", baseURL), nil
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
	case "remote":
		baseURL, err := m.connectRemote(config)
		if err != nil {
			return nil, err
		}
		return NewHTTPExtensionAgent(agentType, baseURL), nil
	default:
		if m.registry != nil {
			if ref, ok := m.registry.ResolveHarness(agentType); ok {
				return m.getExtensionHarnessAgent(projectID, ref)
			}
		}
		if !builtinAgentTypes[agentType] {
			return nil, fmt.Errorf("unknown agent type: %q", agentType)
		}
		return m.getBuiltinAgent(projectID, agentType, config)
	}
}

func agentCacheKey(projectID, agentType string) string {
	return projectID + "\x00" + agentType
}

func projectIDFromCacheKey(key string) string {
	if i := strings.IndexByte(key, '\x00'); i >= 0 {
		return key[:i]
	}
	return key
}

func (m *Manager) getExtensionHarnessAgent(projectID string, ref extensions.HarnessRef) (Agent, error) {
	cacheKey := agentCacheKey(projectID, ref.AgentID)

	m.extMu.Lock()
	if ag, ok := m.extAgents[cacheKey]; ok {
		m.extMu.Unlock()
		return ag, nil
	}
	m.extMu.Unlock()

	v, _ := m.agentMu.LoadOrStore(projectID, &sync.Mutex{})
	lock := v.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	m.extMu.Lock()
	defer m.extMu.Unlock()
	if ag, ok := m.extAgents[cacheKey]; ok {
		return ag, nil
	}

	transport := strings.TrimSpace(ref.Runtime.Transport)
	if transport == "" {
		transport = "stdio"
	}

	var ag Agent
	var err error
	if ref.Extension.IsProgrammatic() {
		ag = newProgrammaticHarnessAgent(ref, projectID)
	} else {
		switch transport {
		case "stdio":
			ag, err = newStdioHarnessAgent(ref.AgentID, ref.Entry.ID, projectID, ref.Extension.Dir, ref.Runtime)
		case "tcp":
			conn, connErr := m.startTCPHarness(ref)
			if connErr != nil {
				return nil, connErr
			}
			ag = NewExtensionAgent(ref.AgentID, conn)
		case "http":
			baseURL, urlErr := m.startHTTPHarness(ref)
			if urlErr != nil {
				return nil, urlErr
			}
			ag = NewHTTPExtensionAgent(ref.AgentID, baseURL)
		default:
			return nil, fmt.Errorf("harness %s: unsupported transport %q", ref.AgentID, transport)
		}
	}
	if err != nil {
		return nil, err
	}
	m.extAgents[cacheKey] = ag
	return ag, nil
}

func (m *Manager) startTCPHarness(ref extensions.HarnessRef) (ConnectionInfo, error) {
	command := extensionsExpandCommand(ref.Runtime.Command, ref.Extension.Dir)
	if len(command) == 0 {
		return ConnectionInfo{}, fmt.Errorf("harness %s: empty runtime command", ref.AgentID)
	}
	cwd := ref.Extension.Dir
	if c := ref.Runtime.Cwd; c != "" && c != "." {
		cwd = filepath.Join(ref.Extension.Dir, c)
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"NUI_EXTENSION_DIR="+ref.Extension.Dir,
		"NUI_HARNESS_ID="+ref.Entry.ID,
		"NUI_CONNECTION_ID="+SanitizeConnectionID(ref.AgentID),
	)
	if err := cmd.Start(); err != nil {
		return ConnectionInfo{}, err
	}
	connID := SanitizeConnectionID(ref.AgentID)
	conn, err := WaitForConnectionInfo(connID, 10*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		return ConnectionInfo{}, fmt.Errorf("harness %s: %w", ref.AgentID, err)
	}
	return conn, nil
}

func (m *Manager) startHTTPHarness(ref extensions.HarnessRef) (string, error) {
	host := ref.Runtime.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := ref.Runtime.Port
	if port <= 0 {
		command := extensionsExpandCommand(ref.Runtime.Command, ref.Extension.Dir)
		if len(command) == 0 {
			return "", fmt.Errorf("harness %s: http transport requires port or command", ref.AgentID)
		}
		cwd := ref.Extension.Dir
		if c := ref.Runtime.Cwd; c != "" && c != "." {
			cwd = filepath.Join(ref.Extension.Dir, c)
		}
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Dir = cwd
		connID := SanitizeConnectionID(ref.AgentID)
		cmd.Env = append(os.Environ(),
			"NUI_EXTENSION_DIR="+ref.Extension.Dir,
			"NUI_CONNECTION_ID="+connID,
		)
		if err := cmd.Start(); err != nil {
			return "", err
		}
		var err error
		host, port, err = ConnectionHostPort(connID, 10*time.Second)
		if err != nil {
			_ = cmd.Process.Kill()
			return "", fmt.Errorf("harness %s: %w", ref.AgentID, err)
		}
	}
	return fmt.Sprintf("http://%s:%d", host, port), nil
}

func (m *Manager) getBuiltinAgent(projectID, agentType string, config map[string]any) (Agent, error) {
	sandbox := sandboxFromConfig(config)
	cacheKey := agentCacheKey(projectID, agentType)

	m.builtinMu.Lock()
	if ag, ok := m.builtinAgents[cacheKey]; ok {
		if builtinAgentSandbox(ag) != sandbox {
			m.builtinMu.Unlock()
			m.stopBuiltinAgentKey(cacheKey)
		} else if ag.Name() != agentType {
			m.builtinMu.Unlock()
			m.stopBuiltinAgentKey(cacheKey)
		} else {
			applyBuiltinSandbox(ag, sandbox)
			applyDevcontainerRuntime(ag, devcontainerWorkspaceFromConfig(config), devcontainerContainerIDFromConfig(config))
			m.builtinMu.Unlock()
			return ag, nil
		}
	} else {
		m.builtinMu.Unlock()
	}

	v, _ := m.agentMu.LoadOrStore(projectID, &sync.Mutex{})
	agLock := v.(*sync.Mutex)
	agLock.Lock()
	defer agLock.Unlock()

	m.builtinMu.Lock()
	defer m.builtinMu.Unlock()
	if ag, ok := m.builtinAgents[cacheKey]; ok {
		applyBuiltinSandbox(ag, sandbox)
		applyDevcontainerRuntime(ag, devcontainerWorkspaceFromConfig(config), devcontainerContainerIDFromConfig(config))
		return ag, nil
	}

	var ag Agent
	switch agentType {
	case "claude-code":
		ag = &ClaudeCodeAgent{}
	case "pi":
		ag = &PiAgent{}
	case "codex":
		ag = &CodexAgent{}
	case "opencode":
		ag = &OpenCodeAgent{}
	default:
		return nil, fmt.Errorf("unknown agent type: %q", agentType)
	}
	m.builtinAgents[cacheKey] = ag
	applyBuiltinSandbox(ag, sandbox)
	applyDevcontainerRuntime(ag, devcontainerWorkspaceFromConfig(config), devcontainerContainerIDFromConfig(config))
	return ag, nil
}

// GetDevcontainerAgent provisions a nui-managed devcontainer and returns the inner CLI agent.
func (m *Manager) GetDevcontainerAgent(projectID, innerHarness, workingDir, sessionConfigDir, image string) (Agent, error) {
	managedDir, containerID, err := m.ensureDevcontainerUp(projectID, innerHarness, workingDir, sessionConfigDir, image)
	if err != nil {
		return nil, err
	}
	cfg := map[string]any{
		"sandbox":                 sandboxDevcontainer,
		"devcontainerWorkspace":   managedDir,
		"devcontainerContainerID": containerID,
	}
	ag, err := m.getBuiltinAgent(projectID, innerHarness, cfg)
	if err != nil {
		return nil, err
	}
	applyDevcontainerRuntime(ag, managedDir, containerID)
	m.touchActivity(projectID)
	return ag, nil
}

func (m *Manager) stopBuiltinAgentKey(cacheKey string) {
	m.builtinMu.Lock()
	ag, ok := m.builtinAgents[cacheKey]
	if ok {
		delete(m.builtinAgents, cacheKey)
	}
	m.builtinMu.Unlock()
	if !ok {
		return
	}
	if s, ok := ag.(interface{ Stop() }); ok {
		s.Stop()
	}
}

func (m *Manager) stopBuiltinAgent(projectID string) {
	m.builtinMu.Lock()
	var keys []string
	for key := range m.builtinAgents {
		if projectIDFromCacheKey(key) == projectID {
			keys = append(keys, key)
		}
	}
	m.builtinMu.Unlock()
	for _, key := range keys {
		m.stopBuiltinAgentKey(key)
	}
}

// Stop terminates the in-process harness agent or Docker container for a specific project.
func (m *Manager) Stop(projectID string) {
	m.stopBuiltinAgent(projectID)
	m.stopExtensionAgent(projectID)

	m.containerMu.Lock()
	baseURL := m.dockerURLs[projectID]
	containerID, hasContainer := m.containers[projectID]
	if hasContainer {
		delete(m.containers, projectID)
		delete(m.devcontainerDirs, projectID)
		delete(m.dockerURLs, projectID)
		delete(m.lastActivity, projectID)
	}
	m.containerMu.Unlock()
	if baseURL != "" {
		ShutdownHTTPAgent(baseURL)
	}
	if hasContainer {
		exec.Command("docker", "stop", containerID).Run()
	}
}

func (m *Manager) stopExtensionAgentKey(cacheKey string) {
	m.extMu.Lock()
	ag, ok := m.extAgents[cacheKey]
	if ok {
		delete(m.extAgents, cacheKey)
	}
	m.extMu.Unlock()
	if !ok {
		return
	}
	if s, ok := ag.(interface{ Stop() }); ok {
		s.Stop()
	}
}

func (m *Manager) stopExtensionAgent(projectID string) {
	m.extMu.Lock()
	var keys []string
	for key := range m.extAgents {
		if projectIDFromCacheKey(key) == projectID {
			keys = append(keys, key)
		}
	}
	m.extMu.Unlock()
	for _, key := range keys {
		m.stopExtensionAgentKey(key)
	}
}

// StopAll terminates all managed harness agents and Docker containers.
func (m *Manager) StopAll() {
	type stopEntry struct {
		projectID    string
		baseURL      string
		containerID  string
		hasContainer bool
	}

	m.builtinMu.Lock()
	builtinKeys := make([]string, 0, len(m.builtinAgents))
	for key := range m.builtinAgents {
		builtinKeys = append(builtinKeys, key)
	}
	m.builtinMu.Unlock()

	m.extMu.Lock()
	extKeys := make([]string, 0, len(m.extAgents))
	for key := range m.extAgents {
		extKeys = append(extKeys, key)
	}
	m.extMu.Unlock()
	for _, key := range extKeys {
		m.stopExtensionAgentKey(key)
	}

	m.containerMu.Lock()
	entries := make([]stopEntry, 0, len(m.containers))
	for id, containerID := range m.containers {
		entries = append(entries, stopEntry{
			projectID:    id,
			baseURL:      m.dockerURLs[id],
			containerID:  containerID,
			hasContainer: true,
		})
		delete(m.containers, id)
		delete(m.devcontainerDirs, id)
		delete(m.dockerURLs, id)
		delete(m.lastActivity, id)
	}
	m.containerMu.Unlock()

	for _, key := range builtinKeys {
		m.stopBuiltinAgentKey(key)
	}
	for _, entry := range entries {
		if entry.baseURL != "" {
			ShutdownHTTPAgent(entry.baseURL)
		}
	}

	var wg sync.WaitGroup
	for _, entry := range entries {
		wg.Add(1)
		go func(cid string) {
			defer wg.Done()
			exec.Command("docker", "stop", cid).Run()
		}(entry.containerID)
	}
	wg.Wait()
}

// ── Docker agent ─────────────────────────────────────────────────────────────

// ── Builtin Docker agents (sandbox: docker) ───────────────────────────────────

const builtinContainerPort = 8090

func (m *Manager) ensureBuiltinDockerRunning(projectID, image, workingDir, harnessType, sessionConfigDir, agentConfigDir string, shadowSettings, userScope bool, extraVolumes ...string) (string, error) {
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
	return m.launchBuiltinDocker(projectID, image, workingDir, harnessType, sessionConfigDir, agentConfigDir, shadowSettings, userScope, extraVolumes...)
}

// snapshotJSONFile copies src to <dir>/<name>, validates it is valid JSON, and returns
// the snapshot path. Returns "" if the copy fails or the JSON is malformed (caller falls
// back to the live bind mount). This prevents mounting a partially-written file.
func snapshotJSONFile(src, dir, name string) string {
	data, err := os.ReadFile(src)
	if err != nil {
		return ""
	}
	if !json.Valid(data) {
		return ""
	}
	dst := filepath.Join(dir, name)
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return ""
	}
	return dst
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

func (m *Manager) launchBuiltinDocker(projectID, image, workingDir, harnessType, sessionConfigDir, agentConfigDir string, shadowSettings, userScope bool, extraVolumes ...string) (string, error) {
	m.containerMu.Lock()
	if oldID, ok := m.containers[projectID]; ok {
		go exec.Command("docker", "stop", oldID).Run()
		delete(m.containers, projectID)
	}
	delete(m.dockerURLs, projectID)
	m.containerMu.Unlock()

	home, _ := os.UserHomeDir()
	containerHome := "/home/nui"

	args := []string{"run", "-d", "--rm",
		"-p", fmt.Sprintf("127.0.0.1::%d", builtinContainerPort),
	}
	for _, envKey := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_OAUTH_TOKEN", "ANTHROPIC_BASE_URL", "OPENAI_API_KEY", "OPENAI_BASE_URL"} {
		if val := os.Getenv(envKey); val != "" {
			args = append(args, "-e", envKey+"="+val)
		}
	}
	// If any base URL points to a loopback hostname, route it to the host machine.
	for _, urlEnv := range []string{"ANTHROPIC_BASE_URL", "OPENAI_BASE_URL"} {
		if extraHosts := loopbackAddHostArgs(os.Getenv(urlEnv)); len(extraHosts) > 0 {
			args = append(args, extraHosts...)
		}
	}
	if workingDir != "" {
		args = append(args, "-v", workingDir+":"+workingDir)
	}
	mountArgs := dockerSessionConfigArgs(harnessType, sessionConfigDir, userScope)
	if len(mountArgs) > 0 {
		args = append(args, mountArgs...)
	}
	mountUserConfig := agentConfigDir != "" && (sessionConfigDir == "" || userScope)
	if mountUserConfig {
		hostConfigDir := filepath.Join(home, agentConfigDir)
		os.MkdirAll(hostConfigDir, 0700) //nolint:errcheck
		args = append(args, "-v", hostConfigDir+":"+containerHome+"/"+agentConfigDir)
		// Mount the top-level config JSON file (e.g. ~/.claude.json) if it exists.
		// Use a snapshot copy to avoid reading a partially-written file (the host process
		// may be actively updating it via atomic rename or in-place writes).
		hostConfigJSON := filepath.Join(home, agentConfigDir+".json")
		if _, statErr := os.Stat(hostConfigJSON); statErr == nil {
			snapshotPath := snapshotJSONFile(hostConfigJSON, filepath.Join(home, ".nui"), agentConfigDir+"-snapshot.json")
			if snapshotPath != "" {
				args = append(args, "-v", snapshotPath+":"+containerHome+"/"+agentConfigDir+".json:ro")
			} else {
				args = append(args, "-v", hostConfigJSON+":"+containerHome+"/"+agentConfigDir+".json")
			}
		}
		if shadowSettings {
			// Shadow the host's settings.json with an empty one so that host-specific
			// hooks, env overrides, and apiKeyHelper (which may depend on host-side
			// mTLS certificates not available in the container) do not interfere.
			// Docker containers authenticate via ANTHROPIC_API_KEY forwarded from the host env.
			overridePath := filepath.Join(home, ".nui", agentConfigDir+"-settings-override.json")
			if writeErr := os.WriteFile(overridePath, []byte("{}"), 0644); writeErr == nil {
				args = append(args, "-v", overridePath+":"+containerHome+"/"+agentConfigDir+"/settings.json:ro")
			}
		}
	}
	for _, vol := range extraVolumes {
		args = append(args, "-v", vol)
	}
	args = append(args, image)

	cmd := exec.Command("docker", args...)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		if stderr != "" {
			return "", fmt.Errorf("docker run %s: %w\n%s", image, err, stderr)
		}
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
	containerPort := configInt(config["containerPort"])
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

// ── Devcontainer sandbox ─────────────────────────────────────────────────────

func (m *Manager) ensureDevcontainerUp(projectID, innerHarness, workingDir, sessionConfigDir, image string) (managedDir, containerID string, err error) {
	m.containerMu.Lock()
	if cid, ok := m.containers[projectID]; ok && devcontainer.ContainerRunning(cid) {
		if dir, ok := m.devcontainerDirs[projectID]; ok {
			m.containerMu.Unlock()
			return dir, cid, nil
		}
	}
	m.containerMu.Unlock()

	v, _ := m.agentMu.LoadOrStore(projectID, &sync.Mutex{})
	agLock := v.(*sync.Mutex)
	agLock.Lock()
	defer agLock.Unlock()

	m.containerMu.Lock()
	if cid, ok := m.containers[projectID]; ok && devcontainer.ContainerRunning(cid) {
		if dir, ok := m.devcontainerDirs[projectID]; ok {
			m.containerMu.Unlock()
			return dir, cid, nil
		}
	}
	m.containerMu.Unlock()

	return m.launchManagedDevcontainer(projectID, innerHarness, workingDir, sessionConfigDir, image)
}

func (m *Manager) launchManagedDevcontainer(projectID, innerHarness, workingDir, sessionConfigDir, image string) (string, string, error) {
	m.containerMu.Lock()
	if oldID, ok := m.containers[projectID]; ok {
		go devcontainer.Stop(oldID)
		delete(m.containers, projectID)
		delete(m.devcontainerDirs, projectID)
	}
	m.containerMu.Unlock()

	managedDir, err := devcontainer.ProvisionSession(devcontainer.ProvisionOpts{
		SessionID:        projectID,
		InnerHarness:     innerHarness,
		Image:            image,
		WorkingDir:       workingDir,
		SessionConfigDir: sessionConfigDir,
	})
	if err != nil {
		return "", "", err
	}

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 30*time.Minute)
	if err := devcontainer.EnsureImage(buildCtx, innerHarness, image); err != nil {
		buildCancel()
		return "", "", err
	}
	buildCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	result, err := devcontainer.Up(ctx, devcontainer.UpOpts{WorkspaceFolder: managedDir})
	if err != nil {
		return "", "", err
	}

	m.containerMu.Lock()
	m.containers[projectID] = result.ContainerID
	m.devcontainerDirs[projectID] = managedDir
	m.containerMu.Unlock()

	return managedDir, result.ContainerID, nil
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
	port := configInt(config["port"])
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

// configInt reads an integer port/number from agent config values that may be
// float64 (JSON), int (Go structs), or other numeric types.
func configInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0
		}
		return int(i)
	default:
		return 0
	}
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
