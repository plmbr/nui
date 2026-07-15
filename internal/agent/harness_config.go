// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"loop/internal/extensions"
	"loop/internal/hitl"
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
	SystemPrompt            string
	MCPServers              []model.ADLMCPServer
	Skills                  []model.ADLSkill
	Rules                   []model.ADLRule
	ResolvedRules           []ResolvedRule
	WorkingDir              string
	UserScope               bool
	ToolApprovalPolicy      string
	ToolApprovalTools       []string
	PendingCustomMCPServers []extensions.PendingCustomMCPServer
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
	"api":         apiHarnessProvisioner{},
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
// When userScope is true the session dir is still mounted for ADL files such as MCP config,
// but harness config env vars are omitted so the container keeps user-scoped settings.
func dockerSessionConfigArgs(harnessType, sessionConfigDir string, userScope bool) []string {
	if sessionConfigDir == "" {
		return nil
	}
	args := []string{"-v", sessionConfigDir + ":" + dockerSessionConfigMount}
	if userScope {
		return args
	}
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
		Rules:        adlRulesFromDef(defCopy),
		WorkingDir:   workingDir,
	}
	if step != nil {
		if step.SystemPrompt != "" {
			deps.SystemPrompt = step.SystemPrompt
		}
		deps.MCPServers = mergeMCPServers(deps.MCPServers, adlMCPServersFromStep(*step))
		deps.Skills = mergeSkills(deps.Skills, adlSkillsFromStep(*step))
		deps.Rules = mergeRules(deps.Rules, adlRulesFromStep(*step))
	}
	return deps
}

func mergeMCPServers(base, step []model.ADLMCPServer) []model.ADLMCPServer {
	return mergeByName(base, step, func(s model.ADLMCPServer) string { return s.Name })
}

func mergeSkills(base, step []model.ADLSkill) []model.ADLSkill {
	return mergeByName(base, step, func(s model.ADLSkill) string { return s.Name })
}

func mergeRules(base, step []model.ADLRule) []model.ADLRule {
	return mergeByName(base, step, func(r model.ADLRule) string { return r.Name })
}

func mergeByName[T any](base, override []T, nameFn func(T) string) []T {
	if len(base) == 0 {
		return append([]T(nil), override...)
	}
	if len(override) == 0 {
		return append([]T(nil), base...)
	}
	index := map[string]int{}
	out := make([]T, 0, len(base)+len(override))
	for _, item := range base {
		name := nameFn(item)
		index[name] = len(out)
		out = append(out, item)
	}
	for _, item := range override {
		name := nameFn(item)
		if i, ok := index[name]; ok {
			out[i] = item
			continue
		}
		index[name] = len(out)
		out = append(out, item)
	}
	return out
}

func adlSkillsFromDef(def model.ADLDefinition) []model.ADLSkill {
	return def.AIAssets.Skills
}

func adlSkillsFromStep(step model.ADLStep) []model.ADLSkill {
	return step.AIAssets.Skills
}

func adlRulesFromDef(def model.ADLDefinition) []model.ADLRule {
	return def.AIAssets.Rules
}

func adlRulesFromStep(step model.ADLStep) []model.ADLRule {
	return step.AIAssets.Rules
}

func adlMCPServersFromDef(def model.ADLDefinition) []model.ADLMCPServer {
	return def.AIAssets.MCPServers
}

func adlMCPServersFromStep(step model.ADLStep) []model.ADLMCPServer {
	return step.AIAssets.MCPServers
}

// PrepareSessionHarnessConfig provisions harness config when a session is created so
// MCP/skills are available before the first message.
func PrepareSessionHarnessConfig(sessionID string, def model.ADLDefinition, reg *extensions.Registry, agentConfig map[string]any) error {
	deps, err := buildHarnessDeps(sessionID, def, nil, "", reg, agentConfig)
	if err != nil {
		return err
	}
	harnessType := def.Harness.Type
	if harnessType == "" {
		harnessType = "claude-code"
	}
	_, err = ProvisionHarnessConfig(sessionID, harnessType, deps)
	return err
}

func buildHarnessDeps(sessionID string, def model.ADLDefinition, step *model.ADLStep, workingDir string, reg *extensions.Registry, agentConfig map[string]any) (HarnessDeps, error) {
	deps := harnessDepsFromADL(def, step, workingDir)
	return ExpandHarnessDeps(deps, reg, sessionID, def, agentConfig)
}

func ExpandHarnessDeps(deps HarnessDeps, reg *extensions.Registry, sessionID string, def model.ADLDefinition, agentConfig map[string]any) (HarnessDeps, error) {
	if reg != nil {
		var pending []extensions.PendingCustomMCPServer
		var err error
		deps.MCPServers, pending, err = reg.ExpandMCPServers(deps.MCPServers)
		if err != nil {
			return deps, err
		}
		deps.PendingCustomMCPServers = append(deps.PendingCustomMCPServers, pending...)
	}
	if len(deps.PendingCustomMCPServers) > 0 {
		configDir, dirErr := store.SessionConfigDir(sessionID)
		if dirErr != nil {
			return deps, dirErr
		}
		for _, pending := range deps.PendingCustomMCPServers {
			srv, matErr := extensions.MaterializeCustomMCPServer(configDir, pending)
			if matErr != nil {
				return deps, matErr
			}
			deps.MCPServers = append(deps.MCPServers, srv)
		}
		deps.PendingCustomMCPServers = nil
	}
	var err error
	deps.MCPServers, err = appendLoopVizMCP(deps.MCPServers)
	if err != nil {
		return deps, err
	}
	if def.Harness.Type == "api" {
		deps.MCPServers, err = appendLoopAgentMCP(deps.MCPServers)
		if err != nil {
			return deps, err
		}
	}
	deps.SystemPrompt = appendVizSystemPrompt(deps.SystemPrompt)
	if def.Harness.Type == "api" && strings.TrimSpace(def.Harness.Provider) == "ollama" {
		deps.SystemPrompt = appendOllamaToolsSystemPrompt(deps.SystemPrompt)
	}
	if hitl.RuntimeAllowed(def, agentConfig) {
		deps.SystemPrompt = appendHitlSystemPrompt(deps.SystemPrompt)
		deps.Skills = appendHitlAskUserSkill(deps.Skills)
		deps.MCPServers, err = appendLoopHitlMCP(deps.MCPServers, sessionID, defaultLoopAPIURL())
		if err != nil {
			return deps, err
		}
	}
	if reg == nil {
		resolved, err := resolveHarnessRules(deps.Rules, nil)
		if err != nil {
			return deps, err
		}
		deps.ResolvedRules = resolved
		deps.Rules = nil
		deps.Skills = skills.WithBuiltins(deps.Skills)
		deps.ToolApprovalPolicy, deps.ToolApprovalTools = hitl.EffectiveToolApprovals(def, agentConfig)
		return deps, nil
	}
	expandedSkills := make([]model.ADLSkill, 0, len(deps.Skills))
	for _, skill := range deps.Skills {
		ref := strings.TrimSpace(skill.Ref)
		if ref != "" && extensions.IsExtRef(ref) {
			_, dir, resolveErr := reg.ResolveSkill(ref)
			if resolveErr != nil {
				return deps, resolveErr
			}
			skill.Ref = ""
			skill.Path = dir
		}
		expandedSkills = append(expandedSkills, skill)
	}
	deps.Skills = expandedSkills

	resolved, err := resolveHarnessRules(deps.Rules, reg)
	if err != nil {
		return deps, err
	}
	deps.ResolvedRules = resolved
	deps.Rules = nil
	deps.Skills = skills.WithBuiltins(deps.Skills)
	deps.ToolApprovalPolicy, deps.ToolApprovalTools = hitl.EffectiveToolApprovals(def, agentConfig)
	return deps, nil
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
// When userScope is true, harness config env vars such as CLAUDE_CONFIG_DIR are omitted.
func applyCmdEnv(cmd *exec.Cmd, harnessType, sessionConfigDir string, adlEnv map[string]string, userScope bool, sessionID, runID string) {
	overrides := make(map[string]string, len(adlEnv)+4)
	for k, v := range adlEnv {
		overrides[k] = v
	}
	for k, v := range loopHarnessEnv(sessionID, runID, defaultLoopAPIURL()) {
		overrides[k] = v
	}
	if !userScope {
		bindDir := harnessConfigBindDir(harnessType, sessionConfigDir)
		if bindDir != "" {
			if envKey := harnessConfigEnvVar(harnessType); envKey != "" {
				overrides[envKey] = bindDir
			}
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

func appendHitlAskUserSkill(skillList []model.ADLSkill) []model.ADLSkill {
	for _, skill := range skillList {
		if skill.Name == skills.HitlAskUserSkillName {
			return skillList
		}
	}
	return append(skillList, skills.HitlAskUserSkill())
}

func writeHarnessManifest(configDir, harness string, deps HarnessDeps, extra map[string]any) error {
	manifest := map[string]any{
		"harness": harness,
		"deps": map[string]any{
			"systemPrompt": deps.SystemPrompt != "",
			"skills":       len(deps.Skills),
			"mcpServers":   len(deps.MCPServers),
			"rules":        len(deps.ResolvedRules),
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
