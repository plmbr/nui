// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nui/internal/extensions"
	"nui/internal/hitl"
	"nui/internal/model"
)

func TestProvisionClaudeHarnessConfig(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	skillSrc := filepath.Join(tmp, "review-skill")
	if err := os.MkdirAll(skillSrc, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("# Review skill\n"), 0644); err != nil {
		t.Fatal(err)
	}

	sessionID := "test-session-1"
	deps := HarnessDeps{
		SystemPrompt: "You are a test agent.",
		MCPServers: []model.ADLMCPServer{
			{Name: "docs", URL: "http://localhost:3040", Type: "http"},
			{Name: "local", Command: "npx", Args: []string{"-y", "some-mcp"}},
		},
		Skills: []model.ADLSkill{{Name: "review-skill", Path: skillSrc}},
	}

	configDir, err := ProvisionHarnessConfig(sessionID, "claude-code", deps)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(configDir, filepath.Join("sessions", sessionID)) {
		t.Fatalf("configDir = %q, want path ending in sessions/%s", configDir, sessionID)
	}

	claudeMD, err := os.ReadFile(filepath.Join(configDir, claudeSystemPromptFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(claudeMD), "You are a test agent.") {
		t.Fatalf("CLAUDE.md = %q", string(claudeMD))
	}

	cfgData, err := os.ReadFile(filepath.Join(configDir, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.MCPServers["docs"]["url"] != "http://localhost:3040" {
		t.Fatalf("docs mcp: %v", cfg.MCPServers["docs"])
	}
	if cfg.MCPServers["local"]["command"] != "npx" {
		t.Fatalf("local mcp: %v", cfg.MCPServers["local"])
	}

	skillPath := filepath.Join(configDir, "skills", "review-skill", "SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skillData), "Review skill") {
		t.Fatalf("skill file = %q", string(skillData))
	}
}

func TestProvisionClaudeHarnessConfigLinksCredentials(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	claudeHome := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeHome, ".credentials.json"), []byte("test-credential\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	sessionID := "cred-session"
	configDir, err := ProvisionHarnessConfig(sessionID, "claude-code", HarnessDeps{})
	if err != nil {
		t.Fatal(err)
	}

	credDst := filepath.Join(configDir, ".credentials.json")
	credData, err := os.ReadFile(credDst)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(credData), "test-credential") {
		t.Fatalf(".credentials.json = %q", string(credData))
	}
}

func TestProvisionClaudeHarnessConfigInstallsBuiltinSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	sessionID := "builtin-skills-session"
	configDir, err := ProvisionHarnessConfig(sessionID, "claude-code", HarnessDeps{})
	if err != nil {
		t.Fatal(err)
	}

	skillPath := filepath.Join(configDir, "skills", "create-agent", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "name: create-agent") {
		t.Fatalf("create-agent SKILL.md = %q", string(data))
	}
}

func TestProvisionClaudeHarnessConfigSkipsCredentialsWhenUserScope(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	claudeHome := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeHome, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeHome, ".credentials.json"), []byte("test-credential\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	configDir, err := ProvisionHarnessConfig("user-scope-session", "claude-code", HarnessDeps{UserScope: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(configDir, ".credentials.json")); err == nil {
		t.Fatal("expected no credential symlink when user scope is enabled")
	}
}

func TestProvisionCodexHarnessConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	sessionID := "codex-session"
	deps := HarnessDeps{
		SystemPrompt: "Codex agent instructions.",
		MCPServers: []model.ADLMCPServer{
			{Name: "docs", URL: "http://localhost:3040", Type: "http"},
		},
	}

	configDir, err := ProvisionHarnessConfig(sessionID, "codex", deps)
	if err != nil {
		t.Fatal(err)
	}

	agentsMD, err := os.ReadFile(filepath.Join(configDir, codexSystemPromptFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agentsMD), "Codex agent instructions.") {
		t.Fatalf("AGENTS.md = %q", string(agentsMD))
	}

	cfg, err := os.ReadFile(filepath.Join(configDir, codexConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	cfgStr := string(cfg)
	if !strings.Contains(cfgStr, "developer_instructions") {
		t.Fatalf("config.toml missing developer_instructions: %s", cfgStr)
	}
	if !strings.Contains(cfgStr, "[mcp_servers.docs]") {
		t.Fatalf("config.toml missing mcp server: %s", cfgStr)
	}
}

func TestProvisionPiHarnessConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	skillSrc := filepath.Join(tmp, "pi-skill")
	if err := os.MkdirAll(skillSrc, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("# Pi skill\n"), 0644); err != nil {
		t.Fatal(err)
	}

	sessionID := "pi-session"
	deps := HarnessDeps{
		SystemPrompt: "Pi system prompt.",
		Skills:       []model.ADLSkill{{Name: "pi-skill", Path: skillSrc}},
		MCPServers: []model.ADLMCPServer{
			{Name: "local", Command: "echo", Args: []string{"mcp"}},
		},
	}

	configDir, err := ProvisionHarnessConfig(sessionID, "pi", deps)
	if err != nil {
		t.Fatal(err)
	}

	agentDir := piAgentConfigDir(configDir)
	if _, err := os.Stat(filepath.Join(agentDir, piSystemPromptFile)); err != nil {
		t.Fatalf("missing pi system prompt file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentDir, "mcp.json")); err != nil {
		t.Fatalf("missing pi mcp.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentDir, "skills", "pi-skill", "SKILL.md")); err != nil {
		t.Fatalf("missing pi skill: %v", err)
	}
}

func TestProvisionOpenCodeHarnessConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	sessionID := "opencode-session"
	deps := HarnessDeps{
		SystemPrompt: "OpenCode instructions.",
		MCPServers: []model.ADLMCPServer{
			{Name: "remote", URL: "http://localhost:9090/mcp", Type: "http"},
		},
	}

	configDir, err := ProvisionHarnessConfig(sessionID, "opencode", deps)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(configDir, opencodeInstructionsFile)); err != nil {
		t.Fatalf("missing instructions file: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(configDir, opencodeConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	mcp, ok := cfg["mcp"].(map[string]any)
	if !ok || mcp["remote"] == nil {
		t.Fatalf("opencode mcp config: %v", cfg)
	}
}

func TestAdlMCPServersFromAIAssets(t *testing.T) {
	def := model.ADLDefinition{
		AIAssets: model.ADLAIAssets{
			MCPServers: []model.ADLMCPServer{{Name: "a", URL: "http://a", Type: "http"}},
		},
	}
	servers := adlMCPServersFromDef(def)
	if len(servers) != 1 || servers[0].Name != "a" {
		t.Fatalf("aiAssets: %v", servers)
	}

	step := model.ADLStep{
		AIAssets: model.ADLAIAssets{
			MCPServers: []model.ADLMCPServer{{Name: "step", URL: "http://step", Type: "http"}},
		},
	}
	deps := harnessDepsFromADL(def, &step, "")
	if len(deps.MCPServers) != 2 {
		t.Fatalf("step aiAssets: %v", deps.MCPServers)
	}
	names := map[string]string{}
	for _, s := range deps.MCPServers {
		names[s.Name] = s.URL
	}
	if names["a"] != "http://a" || names["step"] != "http://step" {
		t.Fatalf("merged mcp: %v", names)
	}
}

func TestHarnessDepsFromADLSkills(t *testing.T) {
	def := model.ADLDefinition{
		AIAssets: model.ADLAIAssets{
			Skills: []model.ADLSkill{{Name: "agent-skill", Ref: "agent-skill"}},
		},
	}
	step := model.ADLStep{
		AIAssets: model.ADLAIAssets{
			Skills: []model.ADLSkill{{Name: "step-skill", Path: "/tmp/step"}},
		},
	}
	deps := harnessDepsFromADL(def, &step, "/workspace")
	if len(deps.Skills) != 2 {
		t.Fatalf("step skills: %v", deps.Skills)
	}
	names := map[string]bool{}
	for _, s := range deps.Skills {
		names[s.Name] = true
	}
	if !names["agent-skill"] || !names["step-skill"] {
		t.Fatalf("merged skills: %v", names)
	}
}

func TestHarnessConfigBindDir(t *testing.T) {
	wantPi := filepath.Join("/tmp/session", piAgentSubdir)
	if harnessConfigBindDir("pi", "/tmp/session") != wantPi {
		t.Fatalf("pi bind dir: got %q want %q", harnessConfigBindDir("pi", "/tmp/session"), wantPi)
	}
	if harnessConfigBindDir("codex", "/tmp/session") != "/tmp/session" {
		t.Fatal("codex bind dir")
	}
}

func TestHarnessConfigEnvVar(t *testing.T) {
	if harnessConfigEnvVar("claude-code") != envClaudeConfigDir {
		t.Fatal("claude env")
	}
	if harnessConfigEnvVar("codex") != envCodexHome {
		t.Fatal("codex env")
	}
	if harnessConfigEnvVar("pi") != envPiCodingAgentDir {
		t.Fatal("pi env")
	}
	if harnessConfigEnvVar("opencode") != "" {
		t.Fatal("opencode uses OPENCODE_CONFIG file path via applyCmdEnv, not a single dir env")
	}
	if harnessConfigEnvVar("docker") != "" {
		t.Fatal("docker should have no env")
	}
}

func TestDockerSessionConfigArgs(t *testing.T) {
	args := dockerSessionConfigArgs("codex", "/tmp/session", false)
	if len(args) != 4 {
		t.Fatalf("args = %v", args)
	}
	if args[0] != "-v" || args[1] != "/tmp/session:"+dockerSessionConfigMount {
		t.Fatalf("volume mount: %v", args[:2])
	}
	if args[2] != "-e" || args[3] != envCodexHome+"="+dockerSessionConfigMount {
		t.Fatalf("codex env: %v", args[2:])
	}

	piArgs := dockerSessionConfigArgs("pi", "/tmp/session", false)
	wantPiEnv := envPiCodingAgentDir + "=" + dockerSessionConfigMount + "/" + piAgentSubdir
	if piArgs[len(piArgs)-1] != wantPiEnv {
		t.Fatalf("pi env: got %q want %q", piArgs[len(piArgs)-1], wantPiEnv)
	}
	if dockerSessionConfigArgs("claude-code", "", false) != nil {
		t.Fatal("empty session dir should produce no args")
	}

	ocArgs := dockerSessionConfigArgs("opencode", "/tmp/session", false)
	if len(ocArgs) != 6 {
		t.Fatalf("opencode args = %v", ocArgs)
	}
	if ocArgs[2] != "-e" || ocArgs[3] != envOpenCodeConfig+"="+dockerSessionConfigMount+"/"+opencodeConfigFile {
		t.Fatalf("opencode OPENCODE_CONFIG: %v", ocArgs)
	}
	if ocArgs[4] != "-e" || ocArgs[5] != envOpenCodeConfigDir+"="+dockerSessionConfigMount {
		t.Fatalf("opencode OPENCODE_CONFIG_DIR: %v", ocArgs)
	}

	userScopeArgs := dockerSessionConfigArgs("claude-code", "/tmp/session", true)
	if len(userScopeArgs) != 2 {
		t.Fatalf("user scope args = %v", userScopeArgs)
	}
	if userScopeArgs[0] != "-v" || userScopeArgs[1] != "/tmp/session:"+dockerSessionConfigMount {
		t.Fatalf("user scope volume = %v", userScopeArgs)
	}
}

func TestProvisionCodexHarnessConfigSkills(t *testing.T) {
	tmp := t.TempDir()
	skillSrc := filepath.Join(tmp, "codex-skill")
	if err := os.MkdirAll(skillSrc, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("# Codex skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	configDir, err := ProvisionHarnessConfig("codex-skills", "codex", HarnessDeps{
		Skills: []model.ADLSkill{{Name: "codex-skill", Path: skillSrc}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "skills", "codex-skill", "SKILL.md")); err != nil {
		t.Fatalf("missing codex skill: %v", err)
	}
}

func TestProvisionOpenCodeHarnessConfigSkills(t *testing.T) {
	tmp := t.TempDir()
	skillSrc := filepath.Join(tmp, "oc-skill")
	if err := os.MkdirAll(skillSrc, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("# OpenCode skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	configDir, err := ProvisionHarnessConfig("opencode-skills", "opencode", HarnessDeps{
		Skills: []model.ADLSkill{{Name: "oc-skill", Path: skillSrc}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "skills", "oc-skill", "SKILL.md")); err != nil {
		t.Fatalf("missing opencode skill: %v", err)
	}
}

func TestProvisionPiHarnessConfigLinksUserConfig(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)

	userAgentDir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(filepath.Join(userAgentDir, "npm"), 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"settings.json":     `{"defaultProvider":"example","defaultModel":"example-model"}`,
		"models.json":       `{"providers":{"example":{"baseUrl":"http://localhost:9999"}}}`,
		"auth.json":         `{"example":{"apiKey":"secret"}}`,
		"models-extra.json": `{"providers":{}}`,
		"mcp.json":          `{"mcpServers":{"user-only":{}}}`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(userAgentDir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	configDir, err := ProvisionHarnessConfig("pi-user-config", "pi", HarnessDeps{
		MCPServers: []model.ADLMCPServer{{Name: "local", Command: "echo", Args: []string{"mcp"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	agentDir := piAgentConfigDir(configDir)

	for _, name := range []string{"settings.json", "models.json", "auth.json", "models-extra.json", "npm", "sessions"} {
		if _, err := os.Stat(filepath.Join(agentDir, name)); err != nil {
			t.Fatalf("missing linked %s: %v", name, err)
		}
	}

	settings, err := os.ReadFile(filepath.Join(agentDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(settings), "example-model") {
		t.Fatalf("settings.json = %q", string(settings))
	}

	// nui generates mcp.json from the ADL, so the user file must not be linked over it.
	mcp, err := os.ReadFile(filepath.Join(agentDir, "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mcp), "user-only") {
		t.Fatalf("generated mcp.json overwritten by user config: %q", string(mcp))
	}
}

func TestProvisionOpenCodeHarnessConfigMergesUserConfig(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)
	userConfigDir := filepath.Join(home, ".config", "opencode")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	if err := os.MkdirAll(filepath.Join(userConfigDir, "node_modules"), 0755); err != nil {
		t.Fatal(err)
	}
	userCfg := `{"model":"example/example-model","instructions":["./AGENTS.md"]}`
	if err := os.WriteFile(filepath.Join(userConfigDir, opencodeConfigFile), []byte(userCfg), 0644); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "auth.json"), []byte(`{"anthropic":{"type":"api","key":"sk-test"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	configDir, err := ProvisionHarnessConfig("opencode-user-config", "opencode", HarnessDeps{
		SystemPrompt: "OpenCode instructions.",
		MCPServers:   []model.ADLMCPServer{{Name: "remote", URL: "http://localhost:9090/mcp", Type: "http"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(filepath.Join(configDir, "auth.json")); err != nil {
		t.Fatalf("expected auth.json symlink from data dir: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(configDir, opencodeConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["model"] != "example/example-model" {
		t.Fatalf("user model setting dropped: %v", cfg)
	}
	if mcp, ok := cfg["mcp"].(map[string]any); !ok || mcp["remote"] == nil {
		t.Fatalf("generated mcp config missing: %v", cfg)
	}
	instructions, ok := cfg["instructions"].([]any)
	if !ok || len(instructions) != 1 || instructions[0] != "./"+opencodeInstructionsFile {
		t.Fatalf("instructions = %v, want only the generated file", cfg["instructions"])
	}
	if _, err := os.Stat(filepath.Join(configDir, "node_modules")); err != nil {
		t.Fatalf("user node_modules not linked: %v", err)
	}
}

func TestParseOpenCodeJSONConfigJSONC(t *testing.T) {
	raw := []byte(`{
  // preferred model
  "model": "anthropic/claude-sonnet-4-6",
  /* providers */
  "provider": {"anthropic": {"options": {"baseURL": "http://example"}}}
}`)
	cfg, err := parseOpenCodeJSONConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["model"] != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("model = %v", cfg["model"])
	}
	provider, ok := cfg["provider"].(map[string]any)
	if !ok || provider["anthropic"] == nil {
		t.Fatalf("provider = %v", cfg["provider"])
	}
}

func TestUserOpenCodeConfigReadsJSONC(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	dir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	raw := `{
  // comment
  "model": "openai/gpt-4o-mini"
}`
	if err := os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := userOpenCodeConfig()
	if cfg["model"] != "openai/gpt-4o-mini" {
		t.Fatalf("cfg = %v", cfg)
	}
}

func TestProvisionHarnessConfigMCPEnvAndHeaders(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	deps := HarnessDeps{
		MCPServers: []model.ADLMCPServer{
			{
				Name:    "remote",
				URL:     "http://localhost:3040/mcp",
				Type:    "http",
				Headers: map[string]string{"Authorization": "Bearer token"},
			},
			{
				Name:    "local",
				Command: "npx",
				Args:    []string{"-y", "some-mcp"},
				Type:    "stdio",
				Env:     map[string]string{"API_KEY": "secret"},
			},
		},
	}

	t.Run("claude-code", func(t *testing.T) {
		configDir, err := ProvisionHarnessConfig("claude-mcp-env", "claude-code", deps)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(configDir, ".claude.json"))
		if err != nil {
			t.Fatal(err)
		}
		var cfg struct {
			MCPServers map[string]map[string]any `json:"mcpServers"`
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		headers, ok := cfg.MCPServers["remote"]["headers"].(map[string]any)
		if !ok || headers["Authorization"] != "Bearer token" {
			t.Fatalf("remote headers: %v", cfg.MCPServers["remote"])
		}
		env, ok := cfg.MCPServers["local"]["env"].(map[string]any)
		if !ok || env["API_KEY"] != "secret" {
			t.Fatalf("local env: %v", cfg.MCPServers["local"])
		}
	})

	t.Run("codex", func(t *testing.T) {
		configDir, err := ProvisionHarnessConfig("codex-mcp-env", "codex", deps)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := os.ReadFile(filepath.Join(configDir, codexConfigFile))
		if err != nil {
			t.Fatal(err)
		}
		cfgStr := string(cfg)
		if !strings.Contains(cfgStr, `http_headers = {`) || !strings.Contains(cfgStr, `Authorization = "Bearer token"`) {
			t.Fatalf("codex http_headers: %s", cfgStr)
		}
		if !strings.Contains(cfgStr, `env = {`) || !strings.Contains(cfgStr, `API_KEY = "secret"`) {
			t.Fatalf("codex env: %s", cfgStr)
		}
	})

	t.Run("pi", func(t *testing.T) {
		configDir, err := ProvisionHarnessConfig("pi-mcp-env", "pi", deps)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(piAgentConfigDir(configDir), "mcp.json"))
		if err != nil {
			t.Fatal(err)
		}
		var cfg struct {
			MCPServers map[string]map[string]any `json:"mcpServers"`
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.MCPServers["remote"]["headers"] == nil {
			t.Fatalf("pi remote headers: %v", cfg.MCPServers["remote"])
		}
		if cfg.MCPServers["local"]["env"] == nil {
			t.Fatalf("pi local env: %v", cfg.MCPServers["local"])
		}
	})

	t.Run("opencode", func(t *testing.T) {
		configDir, err := ProvisionHarnessConfig("opencode-mcp-env", "opencode", deps)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(configDir, opencodeConfigFile))
		if err != nil {
			t.Fatal(err)
		}
		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		mcp := cfg["mcp"].(map[string]any)
		remote := mcp["remote"].(map[string]any)
		if remote["headers"] == nil {
			t.Fatalf("opencode remote headers: %v", remote)
		}
		local := mcp["local"].(map[string]any)
		if local["environment"] == nil {
			t.Fatalf("opencode local environment: %v", local)
		}
	})
}

func TestExpandHarnessDepsCustomMCP(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	proxyPath := filepath.Join(repoRoot, "harness-sdk", "nui_mcp_tools.py")
	if _, err := os.Stat(proxyPath); err != nil {
		t.Skip("harness-sdk not found")
	}
	t.Setenv("NUI_MCP_TOOLS_PATH", proxyPath)

	home := t.TempDir()
	t.Setenv("HOME", home)
	sessionID := "custom-mcp-session"

	deps := HarnessDeps{
		PendingCustomMCPServers: []extensions.PendingCustomMCPServer{{
			ExtensionName: "corp-pack",
			ExtensionDir:  home,
			Server: extensions.ExtensionCustomMCPServer{
				Name: "corp-tools",
				Tools: []extensions.ExtensionMCPTool{
					{Name: "echo", Command: []string{"python3", "-c", "print('ok')"}},
				},
			},
		}},
	}
	expanded, err := ExpandHarnessDeps(deps, nil, sessionID, model.ADLDefinition{
		HITL: model.ADLHITL{Mode: hitl.ModeInteractive},
	}, map[string]any{"hitlMode": "interactive"})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded.MCPServers) != 4 {
		t.Fatalf("mcp servers: %+v", expanded.MCPServers)
	}
	if expanded.MCPServers[0].Name != "ext-corp-pack-corp-tools" || expanded.MCPServers[0].Type != "stdio" {
		t.Fatalf("server: %+v", expanded.MCPServers[0])
	}
	if expanded.MCPServers[1].Name != "nui-viz" {
		t.Fatalf("nui-viz server: %+v", expanded.MCPServers[1])
	}
	if expanded.MCPServers[2].Name != nuiAgentMCPName {
		t.Fatalf("nui-agent server: %+v", expanded.MCPServers[2])
	}
	if expanded.MCPServers[3].Name != nuiHitlMCPName {
		t.Fatalf("nui-hitl server: %+v", expanded.MCPServers[3])
	}

	configDir, err := ProvisionHarnessConfig(sessionID, "claude-code", expanded)
	if err != nil {
		t.Fatal(err)
	}
	cfgData, err := os.ReadFile(filepath.Join(configDir, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatal(err)
	}
	entry := cfg.MCPServers["ext-corp-pack-corp-tools"]
	if entry == nil || entry["command"] == nil {
		t.Fatalf("ext-corp-pack-corp-tools entry: %v", entry)
	}
}

func TestProvisionHarnessRules(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	deps := HarnessDeps{
		SystemPrompt: "Base system prompt.",
		ResolvedRules: []ResolvedRule{
			{Name: "corp-guidelines", Content: "Never commit secrets."},
			{Name: "style", Content: "Use gofmt."},
		},
	}

	t.Run("claude-code", func(t *testing.T) {
		configDir, err := ProvisionHarnessConfig("rules-claude", "claude-code", deps)
		if err != nil {
			t.Fatal(err)
		}
		claudeMD, err := os.ReadFile(filepath.Join(configDir, claudeSystemPromptFile))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(claudeMD), "Never commit secrets.") {
			t.Fatalf("rules should not be merged into CLAUDE.md: %q", string(claudeMD))
		}
		ruleData, err := os.ReadFile(filepath.Join(configDir, "rules", "corp-guidelines.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(ruleData), "Never commit secrets.") {
			t.Fatalf("rule file: %q", string(ruleData))
		}
	})

	t.Run("opencode", func(t *testing.T) {
		configDir, err := ProvisionHarnessConfig("rules-opencode", "opencode", deps)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(configDir, opencodeConfigFile))
		if err != nil {
			t.Fatal(err)
		}
		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		instructions, ok := cfg["instructions"].([]any)
		if !ok || len(instructions) != 3 {
			t.Fatalf("instructions: %v", cfg["instructions"])
		}
		if _, err := os.Stat(filepath.Join(configDir, "rules", "style.md")); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("pi", func(t *testing.T) {
		configDir, err := ProvisionHarnessConfig("rules-pi", "pi", deps)
		if err != nil {
			t.Fatal(err)
		}
		rulePath := filepath.Join(piAgentConfigDir(configDir), "rules", "corp-guidelines.md")
		if _, err := os.Stat(rulePath); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("codex", func(t *testing.T) {
		configDir, err := ProvisionHarnessConfig("rules-codex", "codex", deps)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := os.ReadFile(filepath.Join(configDir, codexConfigFile))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(cfg), "instructions = [") {
			t.Fatalf("config.toml missing instructions: %s", string(cfg))
		}
		if _, err := os.Stat(filepath.Join(configDir, "rules", "corp-guidelines.md")); err != nil {
			t.Fatal(err)
		}
	})
}

func TestHarnessDepsFromADLMergesStepAIAssets(t *testing.T) {
	def := model.ADLDefinition{
		AIAssets: model.ADLAIAssets{
			MCPServers: []model.ADLMCPServer{
				{Name: "global-mcp", URL: "http://localhost:1/mcp", Type: "http"},
			},
			Skills: []model.ADLSkill{
				{Name: "global-skill", Ref: "global-skill"},
			},
		},
	}
	step := &model.ADLStep{
		AIAssets: model.ADLAIAssets{
			MCPServers: []model.ADLMCPServer{
				{Name: "step-mcp", URL: "http://localhost:2/mcp", Type: "http"},
			},
			Skills: []model.ADLSkill{
				{Name: "global-skill", Ref: "overridden"},
			},
		},
	}
	deps := harnessDepsFromADL(def, step, "")
	if len(deps.MCPServers) != 2 {
		t.Fatalf("mcp servers = %d, want 2", len(deps.MCPServers))
	}
	if len(deps.Skills) != 1 || deps.Skills[0].Ref != "overridden" {
		t.Fatalf("skills = %+v", deps.Skills)
	}
	names := map[string]bool{}
	for _, s := range deps.MCPServers {
		names[s.Name] = true
	}
	if !names["global-mcp"] || !names["step-mcp"] {
		t.Fatalf("mcp names = %v", names)
	}
}

func TestWriteClaudeMCPConfigPreservesExistingKeys(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".claude.json")
	seed := map[string]any{
		"theme": "dark",
		"mcpServers": map[string]any{
			"stale": map[string]any{"url": "http://old.example"},
		},
		"userSettings": map[string]any{"foo": "bar"},
	}
	raw, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, raw, 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeClaudeMCPConfig(tmp, []model.ADLMCPServer{
		{Name: "docs", URL: "http://localhost:3040", Type: "http"},
	}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["theme"] != "dark" {
		t.Fatalf("theme wiped: %v", cfg["theme"])
	}
	userSettings, _ := cfg["userSettings"].(map[string]any)
	if userSettings["foo"] != "bar" {
		t.Fatalf("userSettings wiped: %v", cfg["userSettings"])
	}
	mcp, _ := cfg["mcpServers"].(map[string]any)
	if _, ok := mcp["stale"]; ok {
		t.Fatal("stale mcp server should be replaced")
	}
	docs, _ := mcp["docs"].(map[string]any)
	if docs["url"] != "http://localhost:3040" {
		t.Fatalf("docs mcp = %v", docs)
	}
}

func TestWriteClaudeMCPConfigEmptyDoesNotDeleteFile(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".claude.json")
	seed := map[string]any{
		"numStartups": float64(3),
		"mcpServers": map[string]any{
			"docs": map[string]any{"url": "http://localhost:3040"},
		},
	}
	raw, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, raw, 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeClaudeMCPConfig(tmp, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("expected .claude.json to remain: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg["mcpServers"]; ok {
		t.Fatalf("mcpServers should be cleared: %v", cfg["mcpServers"])
	}
	if cfg["numStartups"] != float64(3) {
		t.Fatalf("numStartups wiped: %v", cfg["numStartups"])
	}
}

func TestWriteClaudeMCPConfigEmptyNoFileIsNoop(t *testing.T) {
	tmp := t.TempDir()
	if err := writeClaudeMCPConfig(tmp, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".claude.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no .claude.json, err=%v", err)
	}
}
