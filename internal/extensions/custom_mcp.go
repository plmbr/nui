// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	harnesssdk "nui/harness-sdk"
	"nui/internal/model"
)

// ExtensionCustomMCPServer is a command-tool MCP server declared under contributions.aiAssets.mcpServers.
type ExtensionCustomMCPServer struct {
	Name  string             `yaml:"name"`
	Tools []ExtensionMCPTool `yaml:"tools"`
}

// ExtensionMCPTool is one tool backed by a CLI command (JSON args on stdin).
type ExtensionMCPTool struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Command     []string       `yaml:"command"`
	InputSchema map[string]any `yaml:"inputSchema,omitempty"`
}

// customMCPToolsConfig is written to the session config dir for nui_mcp_tools.py.
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

// ExtensionCustomRule is a rule block declared under contributions.aiAssets.rules.
type ExtensionCustomRule struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path,omitempty"`
	Content string `yaml:"content,omitempty"`
}

func validateCustomRules(rules []ExtensionCustomRule, extName string) error {
	seen := map[string]bool{}
	for i, rule := range rules {
		name := strings.TrimSpace(rule.Name)
		if name == "" {
			return fmt.Errorf("extension %s: aiAssets.rules[%d]: name is required", extName, i)
		}
		if seen[name] {
			return fmt.Errorf("extension %s: duplicate aiAssets rule name %q", extName, name)
		}
		seen[name] = true
		hasPath := strings.TrimSpace(rule.Path) != ""
		hasContent := strings.TrimSpace(rule.Content) != ""
		switch {
		case hasContent && !hasPath:
		case hasPath && !hasContent:
		default:
			return fmt.Errorf("extension %s: aiAssets.rules[%q]: requires exactly one source (path or content)", extName, name)
		}
	}
	return nil
}

func expandCustomRules(extDir string, rules []ExtensionCustomRule) []ExtensionCustomRule {
	if len(rules) == 0 {
		return nil
	}
	out := make([]ExtensionCustomRule, len(rules))
	copy(out, rules)
	for i := range out {
		if p := strings.TrimSpace(out[i].Path); p != "" && !filepath.IsAbs(p) {
			out[i].Path = filepath.Join(extDir, p)
		}
	}
	return out
}

// ResolveCustomRule returns the rule body from inline content or a file path.
func ResolveCustomRule(rule ExtensionCustomRule) (string, error) {
	if content := strings.TrimSpace(rule.Content); content != "" {
		return content, nil
	}
	path := strings.TrimSpace(rule.Path)
	if path == "" {
		return "", fmt.Errorf("rule %q: empty source", rule.Name)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func customMCPServerName(extName, serverName string) string {
	return fmt.Sprintf("ext-%s-%s", extName, serverName)
}

// MaterializeCustomMCPServer writes tools JSON and returns a stdio ADLMCPServer using nui_mcp_tools.py.
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

// MCPToolsProxyPath locates harness-sdk/nui_mcp_tools.py, installing a copy under ~/.nui when needed.
func MCPToolsProxyPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("NUI_MCP_TOOLS_PATH")); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("NUI_MCP_TOOLS_PATH %q: %w", p, err)
		}
		return p, nil
	}
	return harnesssdk.FilePath("nui_mcp_tools.py")
}

func python3Path() string {
	if p := strings.TrimSpace(os.Getenv("NUI_PYTHON3_PATH")); p != "" {
		return p
	}
	if p, err := exec.LookPath("python3"); err == nil {
		return p
	}
	return "python3"
}
