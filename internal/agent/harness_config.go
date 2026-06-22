// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"loop/internal/model"
	"loop/internal/skills"
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
	Skills       []model.ADLSkill
	WorkingDir   string
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

// dockerSessionConfigArgs returns docker run -v/-e flags for a provisioned session config dir.
func dockerSessionConfigArgs(harnessType, sessionConfigDir string) []string {
	if sessionConfigDir == "" {
		return nil
	}
	args := []string{"-v", sessionConfigDir + ":" + dockerSessionConfigMount}
	if envKey := harnessConfigEnvVar(harnessType); envKey != "" {
		args = append(args, "-e", envKey+"="+dockerHarnessConfigEnvValue(harnessType))
	}
	return args
}

func dockerHarnessConfigEnvValue(harnessType string) string {
	if normalizeHarnessType(harnessType) == "pi" {
		return dockerSessionConfigMount + "/" + piAgentSubdir
	}
	return dockerSessionConfigMount
}

// harnessDepsFromADL merges top-level ADL fields with optional step overrides.
func harnessDepsFromADL(def model.ADLDefinition, step *model.ADLStep, workingDir string) HarnessDeps {
	defCopy := def
	model.NormalizeADLSkills(&defCopy)

	deps := HarnessDeps{
		SystemPrompt: defCopy.SystemPrompt,
		MCPServers:   adlMCPServersFromDef(defCopy),
		Skills:       adlSkillsFromDef(defCopy),
		WorkingDir:   workingDir,
	}
	if step != nil {
		if step.SystemPrompt != "" {
			deps.SystemPrompt = step.SystemPrompt
		}
		if servers := adlMCPServersFromStep(*step); len(servers) > 0 {
			deps.MCPServers = servers
		}
		if stepSkills := adlSkillsFromStep(*step); len(stepSkills) > 0 {
			deps.Skills = stepSkills
		}
	}
	return deps
}

func adlSkillsFromDef(def model.ADLDefinition) []model.ADLSkill {
	return def.AIAssets.Skills
}

func adlSkillsFromStep(step model.ADLStep) []model.ADLSkill {
	return step.AIAssets.Skills
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

// applyCmdEnv sets cmd.Env from the host environment, ADL overrides, and the harness config dir.
func applyCmdEnv(cmd *exec.Cmd, harnessType, sessionConfigDir string, adlEnv map[string]string) {
	overrides := make(map[string]string, len(adlEnv)+1)
	for k, v := range adlEnv {
		overrides[k] = v
	}
	bindDir := harnessConfigBindDir(harnessType, sessionConfigDir)
	if bindDir != "" {
		if envKey := harnessConfigEnvVar(harnessType); envKey != "" {
			overrides[envKey] = bindDir
		}
	}
	cmd.Env = envWithOverrides(overrides)
}

func envWithOverrides(overrides map[string]string) []string {
	if len(overrides) == 0 {
		return os.Environ()
	}
	m := make(map[string]string)
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if ok {
			m[k] = v
		}
	}
	for k, v := range overrides {
		m[k] = v
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

func writeHarnessManifest(configDir, harness string, deps HarnessDeps, extra map[string]any) error {
	manifest := map[string]any{
		"harness": harness,
		"deps": map[string]any{
			"systemPrompt": deps.SystemPrompt != "",
			"skills":       len(deps.Skills),
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
	return skills.ExpandPath(p)
}
