// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"loop/internal/model"
)

// ExtensionCustomMCPServer is a command-tool MCP server declared under contributions.aiAssets.mcpServers.
type ExtensionCustomMCPServer struct {
	Name    string             `yaml:"name"`
	Install bool               `yaml:"install,omitempty"`
	Tools   []ExtensionMCPTool `yaml:"tools"`
}

// ExtensionMCPTool is one tool backed by a CLI command (JSON args on stdin).
type ExtensionMCPTool struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Command     []string       `yaml:"command"`
	InputSchema map[string]any `yaml:"inputSchema,omitempty"`
}

// customMCPToolsConfig is written to the session config dir for loop_mcp_tools.py.
type customMCPToolsConfig struct {
	ServerName   string             `json:"serverName"`
	ExtensionDir string             `json:"extensionDir"`
	Tools        []customMCPToolDef `json:"tools"`
}

type customMCPToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Command     []string       `json:"command"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

func validateCustomMCPServers(servers []ExtensionCustomMCPServer, extName string) error {
	seen := map[string]bool{}
	for i, s := range servers {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			return fmt.Errorf("extension %s: mcpServers[%d]: name is required", extName, i)
		}
		if seen[name] {
			return fmt.Errorf("extension %s: duplicate mcpServer name %q", extName, name)
		}
		seen[name] = true
		if len(s.Tools) == 0 {
			return fmt.Errorf("extension %s: mcpServers[%q]: at least one tool is required", extName, name)
		}
		toolNames := map[string]bool{}
		for j, tool := range s.Tools {
			toolName := strings.TrimSpace(tool.Name)
			if toolName == "" {
				return fmt.Errorf("extension %s: mcpServers[%q].tools[%d]: name is required", extName, name, j)
			}
			if toolNames[toolName] {
				return fmt.Errorf("extension %s: mcpServers[%q]: duplicate tool name %q", extName, name, toolName)
			}
			toolNames[toolName] = true
			if len(tool.Command) == 0 {
				return fmt.Errorf("extension %s: mcpServers[%q].tools[%q]: command is required", extName, name, toolName)
			}
		}
	}
	return nil
}

func expandCustomMCPServers(extDir string, servers []ExtensionCustomMCPServer) []ExtensionCustomMCPServer {
	if len(servers) == 0 {
		return nil
	}
	out := make([]ExtensionCustomMCPServer, len(servers))
	copy(out, servers)
	for i := range out {
		for j := range out[i].Tools {
			out[i].Tools[j].Command = expandCommand(out[i].Tools[j].Command, extDir)
		}
	}
	return out
}

// ExtensionCustomSkill is a skill declared under contributions.aiAssets.skills.
type ExtensionCustomSkill struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path,omitempty"`
	Content string `yaml:"content,omitempty"`
	Install bool   `yaml:"install,omitempty"`
}

func (s ExtensionCustomSkill) adlSkill() model.ADLSkill {
	return model.ADLSkill{
		Name:    s.Name,
		Path:    s.Path,
		Content: s.Content,
	}
}

func validateCustomSkills(skills []ExtensionCustomSkill, extName string) error {
	seen := map[string]bool{}
	for i, s := range skills {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			return fmt.Errorf("extension %s: aiAssets.skills[%d]: name is required", extName, i)
		}
		if seen[name] {
			return fmt.Errorf("extension %s: duplicate aiAssets skill name %q", extName, name)
		}
		seen[name] = true
		adl := s.adlSkill()
		if _, err := model.SkillSourceKind(adl); err != nil {
			return fmt.Errorf("extension %s: aiAssets.skills[%q]: %w", extName, name, err)
		}
	}
	return nil
}

func expandCustomSkills(extDir string, skills []ExtensionCustomSkill) []ExtensionCustomSkill {
	if len(skills) == 0 {
		return nil
	}
	out := make([]ExtensionCustomSkill, len(skills))
	copy(out, skills)
	for i := range out {
		if p := strings.TrimSpace(out[i].Path); p != "" && !filepath.IsAbs(p) {
			out[i].Path = filepath.Join(extDir, p)
		}
	}
	return out
}

// ResolveCustomSkill resolves an aiAssets skill to an ADL skill with absolute path.
func ResolveCustomSkill(extDir string, skill ExtensionCustomSkill) (model.ADLSkill, error) {
	dir, err := skillDir(extDir, skill.adlSkill())
	if err != nil {
		return model.ADLSkill{}, err
	}
	out := skill.adlSkill()
	out.Path = dir
	out.Content = ""
	return out, nil
}

// InstallableCustomMCPServers returns custom MCP servers with install: true.
func (ext *Extension) InstallableCustomMCPServers() []ExtensionCustomMCPServer {
	var out []ExtensionCustomMCPServer
	for _, s := range ext.CustomMCPServers {
		if s.Install {
			out = append(out, s)
		}
	}
	return out
}

// InstallableCustomSkills returns aiAssets skills with install: true.
func (ext *Extension) InstallableCustomSkills() []ExtensionCustomSkill {
	var out []ExtensionCustomSkill
	for _, s := range ext.CustomSkills {
		if s.Install {
			out = append(out, s)
		}
	}
	return out
}

// ExtensionCustomInstruction is a system-prompt block declared under contributions.aiAssets.instructions.
type ExtensionCustomInstruction struct {
	Name    string `yaml:"name"`
	Install bool   `yaml:"install,omitempty"`
	Path    string `yaml:"path,omitempty"`
	Content string `yaml:"content,omitempty"`
}

func validateCustomInstructions(instructions []ExtensionCustomInstruction, extName string) error {
	seen := map[string]bool{}
	for i, inst := range instructions {
		name := strings.TrimSpace(inst.Name)
		if name == "" {
			return fmt.Errorf("extension %s: aiAssets.instructions[%d]: name is required", extName, i)
		}
		if seen[name] {
			return fmt.Errorf("extension %s: duplicate aiAssets instruction name %q", extName, name)
		}
		seen[name] = true
		hasPath := strings.TrimSpace(inst.Path) != ""
		hasContent := strings.TrimSpace(inst.Content) != ""
		switch {
		case hasContent && !hasPath:
		case hasPath && !hasContent:
		default:
			return fmt.Errorf("extension %s: aiAssets.instructions[%q]: requires exactly one source (path or content)", extName, name)
		}
	}
	return nil
}

func expandCustomInstructions(extDir string, instructions []ExtensionCustomInstruction) []ExtensionCustomInstruction {
	if len(instructions) == 0 {
		return nil
	}
	out := make([]ExtensionCustomInstruction, len(instructions))
	copy(out, instructions)
	for i := range out {
		if p := strings.TrimSpace(out[i].Path); p != "" && !filepath.IsAbs(p) {
			out[i].Path = filepath.Join(extDir, p)
		}
	}
	return out
}

// ResolveCustomInstruction returns the instruction body from inline content or a file path.
func ResolveCustomInstruction(inst ExtensionCustomInstruction) (string, error) {
	if content := strings.TrimSpace(inst.Content); content != "" {
		return content, nil
	}
	path := strings.TrimSpace(inst.Path)
	if path == "" {
		return "", fmt.Errorf("instruction %q: empty source", inst.Name)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// InstallableCustomInstructions returns aiAssets instructions with install: true.
func (ext *Extension) InstallableCustomInstructions() []ExtensionCustomInstruction {
	var out []ExtensionCustomInstruction
	for _, inst := range ext.CustomInstructions {
		if inst.Install {
			out = append(out, inst)
		}
	}
	return out
}

func appendSystemPromptInstructions(base string, blocks []string) string {
	var parts []string
	if trimmed := strings.TrimSpace(base); trimmed != "" {
		parts = append(parts, trimmed)
	}
	for _, block := range blocks {
		if trimmed := strings.TrimSpace(block); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, "\n\n")
}

// AgentHarnessDepsInput is passed from the agent package for aiAssets merge.
type AgentHarnessDepsInput struct {
	MCPServers   []model.ADLMCPServer
	Skills       []model.ADLSkill
	SystemPrompt string
}

// AgentHarnessDepsOutput carries merged aiAssets state back to the agent package.
type AgentHarnessDepsOutput struct {
	MCPServers              []model.ADLMCPServer
	Skills                  []model.ADLSkill
	SystemPrompt            string
	PendingCustomMCPServers []PendingCustomMCPServer
}

// MergeInstallableAIAssetsForAgent merges installable aiAssets MCP servers, skills, and instructions from all
// enabled extensions (install: true) into harness deps.
func (r *Registry) MergeInstallableAIAssetsForAgent(in AgentHarnessDepsInput, agentID string) (AgentHarnessDepsOutput, error) {
	_ = agentID
	out := AgentHarnessDepsOutput{
		MCPServers:   in.MCPServers,
		Skills:       in.Skills,
		SystemPrompt: in.SystemPrompt,
	}
	if r == nil {
		return out, nil
	}
	existingMCP := map[string]bool{}
	for _, s := range out.MCPServers {
		existingMCP[strings.TrimSpace(s.Name)] = true
	}
	pendingMCP := map[string]bool{}
	existingSkills := map[string]bool{}
	for _, s := range out.Skills {
		existingSkills[strings.TrimSpace(s.Name)] = true
	}
	existingInstructions := map[string]bool{}
	var instructionBlocks []string

	r.mu.RLock()
	extList := make([]*Extension, 0, len(r.extensions))
	for name, ext := range r.extensions {
		if r.isDisabled(name) {
			continue
		}
		extList = append(extList, ext)
	}
	r.mu.RUnlock()

	for _, ext := range extList {
		extName := ext.Manifest.Name
		for _, custom := range ext.InstallableCustomMCPServers() {
			mcpName := customMCPServerName(extName, custom.Name)
			if existingMCP[mcpName] || pendingMCP[mcpName] {
				continue
			}
			pendingMCP[mcpName] = true
			out.PendingCustomMCPServers = append(out.PendingCustomMCPServers, PendingCustomMCPServer{
				ExtensionName: extName,
				ExtensionDir:  ext.Dir,
				Server:        custom,
			})
		}
		for _, skill := range ext.InstallableCustomSkills() {
			if existingSkills[skill.Name] {
				continue
			}
			resolved, err := ResolveCustomSkill(ext.Dir, skill)
			if err != nil {
				return out, fmt.Errorf("extension %s skill %q: %w", extName, skill.Name, err)
			}
			out.Skills = append(out.Skills, resolved)
			existingSkills[skill.Name] = true
		}
		for _, inst := range ext.InstallableCustomInstructions() {
			if existingInstructions[inst.Name] {
				continue
			}
			body, err := ResolveCustomInstruction(inst)
			if err != nil {
				return out, fmt.Errorf("extension %s instruction %q: %w", extName, inst.Name, err)
			}
			instructionBlocks = append(instructionBlocks, body)
			existingInstructions[inst.Name] = true
		}
	}
	if len(instructionBlocks) > 0 {
		out.SystemPrompt = appendSystemPromptInstructions(out.SystemPrompt, instructionBlocks)
	}
	return out, nil
}

// MergeInstallableMCPServersForAgent merges installable aiAssets (MCP servers and skills).
func (r *Registry) MergeInstallableMCPServersForAgent(in AgentHarnessDepsInput, agentID string) AgentHarnessDepsOutput {
	out, err := r.MergeInstallableAIAssetsForAgent(in, agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[extensions] merge aiAssets: %v\n", err)
	}
	return out
}

func customMCPServerName(extName, serverName string) string {
	return fmt.Sprintf("ext-%s-%s", extName, serverName)
}

// MaterializeCustomMCPServer writes tools JSON and returns a stdio ADLMCPServer using loop_mcp_tools.py.
func MaterializeCustomMCPServer(configDir string, pending PendingCustomMCPServer) (model.ADLMCPServer, error) {
	proxyPath, err := MCPToolsProxyPath()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	toolsDir := filepath.Join(configDir, "mcp-tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return model.ADLMCPServer{}, err
	}
	cfgPath := filepath.Join(toolsDir, pending.Server.Name+".json")
	cfg := customMCPToolsConfig{
		ServerName:   pending.Server.Name,
		ExtensionDir: pending.ExtensionDir,
	}
	for _, tool := range pending.Server.Tools {
		cfg.Tools = append(cfg.Tools, customMCPToolDef{
			Name:        tool.Name,
			Description: tool.Description,
			Command:     tool.Command,
			InputSchema: tool.InputSchema,
		})
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return model.ADLMCPServer{}, err
	}
	python := python3Path()
	return model.ADLMCPServer{
		Name:    customMCPServerName(pending.ExtensionName, pending.Server.Name),
		Command: python,
		Args:    []string{proxyPath, cfgPath},
		Type:    "stdio",
	}, nil
}

// PendingCustomMCPServer is exported for agent-side materialization.
type PendingCustomMCPServer struct {
	ExtensionName string
	ExtensionDir  string
	Server        ExtensionCustomMCPServer
}

type pendingCustomMCPServer = PendingCustomMCPServer

// MCPToolsProxyPath locates harness-sdk/loop_mcp_tools.py, installing a copy under ~/.loop when needed.
func MCPToolsProxyPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("LOOP_MCP_TOOLS_PATH")); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("LOOP_MCP_TOOLS_PATH %q: %w", p, err)
		}
		return p, nil
	}
	if source, err := findMCPToolsProxySource(); err == nil {
		return installMCPToolsProxy(source)
	}
	return installedMCPToolsProxyPath()
}

func installedMCPToolsProxyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".loop", "harness-sdk", "loop_mcp_tools.py")
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func findMCPToolsProxySource() (string, error) {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 6; i++ {
			candidate := filepath.Join(dir, "harness-sdk", "loop_mcp_tools.py")
			if _, err := os.Stat(candidate); err == nil {
				return filepath.Abs(candidate)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "harness-sdk", "loop_mcp_tools.py")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("loop_mcp_tools.py not found (set LOOP_MCP_TOOLS_PATH)")
}

func installMCPToolsProxy(source string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	destDir := filepath.Join(home, ".loop", "harness-sdk")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
	dest := filepath.Join(destDir, "loop_mcp_tools.py")
	data, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return "", err
	}
	return dest, nil
}

func python3Path() string {
	if p := strings.TrimSpace(os.Getenv("LOOP_PYTHON3_PATH")); p != "" {
		return p
	}
	if p, err := exec.LookPath("python3"); err == nil {
		return p
	}
	return "python3"
}
