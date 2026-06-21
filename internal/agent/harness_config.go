// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"loop/internal/model"
	"loop/internal/store"
)

const (
	envClaudeConfigDir    = "CLAUDE_CONFIG_DIR"
	envCodexHome          = "CODEX_HOME"
	envOpenCodeConfigDir  = "OPENCODE_CONFIG_DIR"
	envPiCodingAgentDir   = "PI_CODING_AGENT_DIR"
)

// HarnessDeps are ADL-derived files Loop materializes into a session config directory.
type HarnessDeps struct {
	SystemPrompt string
	MCPServers   []model.ADLMCPServer
	Skill        string
}

type harnessProvisioner interface {
	provision(configDir string, deps HarnessDeps) error
}

var harnessProvisioners = map[string]harnessProvisioner{
	"claude-code": claudeHarnessProvisioner{},
	"":            claudeHarnessProvisioner{},
	"codex":       codexHarnessProvisioner{},
	"pi":          piHarnessProvisioner{},
	"opencode":    opencodeHarnessProvisioner{},
}

// harnessConfigEnvVar returns the environment variable that redirects harness config
// to a session directory (empty when the harness type has no known config dir env).
func harnessConfigEnvVar(harnessType string) string {
	switch normalizeHarnessType(harnessType) {
	case "claude-code":
		return envClaudeConfigDir
	case "codex":
		return envCodexHome
	case "opencode":
		return envOpenCodeConfigDir
	case "pi":
		return envPiCodingAgentDir
	default:
		return ""
	}
}

func normalizeHarnessType(harnessType string) string {
	if harnessType == "" {
		return "claude-code"
	}
	return harnessType
}

// harnessDepsFromADL merges top-level ADL fields with optional step overrides.
func harnessDepsFromADL(def model.ADLDefinition, step *model.ADLStep) HarnessDeps {
	deps := HarnessDeps{
		SystemPrompt: def.SystemPrompt,
		MCPServers:   adlMCPServersFromDef(def),
		Skill:        def.Skill,
	}
	if step != nil {
		if step.SystemPrompt != "" {
			deps.SystemPrompt = step.SystemPrompt
		}
		if servers := adlMCPServersFromStep(*step); len(servers) > 0 {
			deps.MCPServers = servers
		}
	}
	return deps
}

func adlMCPServersFromDef(def model.ADLDefinition) []model.ADLMCPServer {
	return def.AIAssets.MCPServers
}

func adlMCPServersFromStep(step model.ADLStep) []model.ADLMCPServer {
	return step.AIAssets.MCPServers
}

// ProvisionHarnessConfig creates ~/.loop/sessions/<sessionID> and writes harness-specific
// config files derived from ADL dependencies.
func ProvisionHarnessConfig(sessionID, harnessType string, deps HarnessDeps) (string, error) {
	harnessType = normalizeHarnessType(harnessType)
	configDir, err := store.SessionConfigDir(sessionID)
	if err != nil {
		return "", err
	}

	prov, ok := harnessProvisioners[harnessType]
	if !ok {
		return configDir, nil
	}
	if err := prov.provision(configDir, deps); err != nil {
		return "", err
	}
	return configDir, nil
}

// applyHarnessConfigEnv sets cmd.Env with the harness config directory env var.
func applyHarnessConfigEnv(cmd *exec.Cmd, harnessType, sessionConfigDir string) {
	bindDir := harnessConfigBindDir(harnessType, sessionConfigDir)
	if bindDir == "" {
		return
	}
	envKey := harnessConfigEnvVar(harnessType)
	if envKey == "" {
		return
	}
	cmd.Env = append(os.Environ(), envKey+"="+bindDir)
}

func writeHarnessManifest(configDir, harness string, deps HarnessDeps, extra map[string]any) error {
	manifest := map[string]any{
		"harness": harness,
		"deps": map[string]any{
			"systemPrompt": deps.SystemPrompt != "",
			"skill":        deps.Skill,
			"mcpServers":   len(deps.MCPServers),
		},
	}
	for k, v := range extra {
		manifest[k] = v
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "manifest.json"), data, 0644)
}

func expandPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	if p[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if len(p) == 1 || p[1] == '/' {
			p = filepath.Join(home, p[1:])
		}
	}
	return filepath.Abs(p)
}
