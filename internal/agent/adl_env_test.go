// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"nui/internal/model"
	"nui/internal/store"
)

func TestMergeADLEnv(t *testing.T) {
	def := model.ADLDefinition{
		Env: map[string]string{
			"ANTHROPIC_API_KEY":  "global-key",
			"ANTHROPIC_BASE_URL": "https://global.example",
		},
	}
	harness := model.ADLHarness{
		Env: map[string]string{
			"ANTHROPIC_API_KEY": "harness-key",
		},
	}
	got := mergeADLEnv(def, harness)
	if got["ANTHROPIC_API_KEY"] != "harness-key" {
		t.Fatalf("API key = %q, want harness override", got["ANTHROPIC_API_KEY"])
	}
	if got["ANTHROPIC_BASE_URL"] != "https://global.example" {
		t.Fatalf("base URL = %q", got["ANTHROPIC_BASE_URL"])
	}
}

func TestEnvWithOverrides(t *testing.T) {
	t.Setenv("NUI_TEST_ENV_BASE", "from-host")
	got := envWithOverrides(map[string]string{
		"NUI_TEST_ENV_BASE": "from-adl",
		"NUI_TEST_ENV_NEW":  "added",
	})
	m := map[string]string{}
	for _, kv := range got {
		k, v, ok := cutEnv(kv)
		if !ok {
			t.Fatalf("bad kv %q", kv)
		}
		m[k] = v
	}
	if m["NUI_TEST_ENV_BASE"] != "from-adl" {
		t.Fatalf("override = %q", m["NUI_TEST_ENV_BASE"])
	}
	if m["NUI_TEST_ENV_NEW"] != "added" {
		t.Fatalf("new = %q", m["NUI_TEST_ENV_NEW"])
	}
}

func TestApplyCmdEnv(t *testing.T) {
	t.Setenv("NUI_CMD_ENV_TEST", "host")
	cmd := exec.Command("true")
	applyCmdEnv(cmd, "claude-code", "/tmp/session-config", map[string]string{
		"NUI_CMD_ENV_TEST": "adl",
	}, false, "sess-1", "run-1")
	m := envMap(cmd.Env)
	if m["NUI_CMD_ENV_TEST"] != "adl" {
		t.Fatalf("override = %q", m["NUI_CMD_ENV_TEST"])
	}
	if m["CLAUDE_CONFIG_DIR"] != "/tmp/session-config" {
		t.Fatalf("CLAUDE_CONFIG_DIR = %q", m["CLAUDE_CONFIG_DIR"])
	}
}

func TestEnvWithOverrides_injectsSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "")
	if err := store.SaveSecrets(store.Secrets{Env: map[string]string{
		"ANTHROPIC_API_KEY": "sk-from-secrets",
	}}); err != nil {
		t.Fatal(err)
	}

	got := envWithOverrides(map[string]string{"NUI_TEST_ENV_NEW": "added"})
	m := envMap(got)
	if m["ANTHROPIC_API_KEY"] != "sk-from-secrets" {
		t.Fatalf("ANTHROPIC_API_KEY = %q, want secrets value", m["ANTHROPIC_API_KEY"])
	}
	if m["NUI_TEST_ENV_NEW"] != "added" {
		t.Fatalf("override missing: %q", m["NUI_TEST_ENV_NEW"])
	}
}

func TestEnvWithOverrides_processEnvWinsOverSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSecrets(store.Secrets{Env: map[string]string{
		"ANTHROPIC_API_KEY": "sk-from-secrets",
	}}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "sk-from-env")

	got := envWithOverrides(nil)
	m := envMap(got)
	if m["ANTHROPIC_API_KEY"] != "sk-from-env" {
		t.Fatalf("ANTHROPIC_API_KEY = %q, want process env", m["ANTHROPIC_API_KEY"])
	}
}

func TestApplyCmdEnv_forwardsSecretsToClaude(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "")
	if err := store.SaveSecrets(store.Secrets{Env: map[string]string{
		"ANTHROPIC_API_KEY": "sk-claude-desktop",
	}}); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("true")
	applyCmdEnv(cmd, "claude-code", "/tmp/session-config", nil, false, "sess-1", "run-1")
	m := envMap(cmd.Env)
	if m["ANTHROPIC_API_KEY"] != "sk-claude-desktop" {
		t.Fatalf("ANTHROPIC_API_KEY = %q, want secrets forwarded to claude-code", m["ANTHROPIC_API_KEY"])
	}
}

func cutEnv(kv string) (string, string, bool) {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return kv[:i], kv[i+1:], true
		}
	}
	return "", "", false
}

func envMap(env []string) map[string]string {
	m := make(map[string]string)
	for _, kv := range env {
		k, v, ok := cutEnv(kv)
		if ok {
			m[k] = v
		}
	}
	return m
}
