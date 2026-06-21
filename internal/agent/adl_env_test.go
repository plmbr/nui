// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os/exec"
	"testing"

	"loop/internal/model"
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
	t.Setenv("LOOP_TEST_ENV_BASE", "from-host")
	got := envWithOverrides(map[string]string{
		"LOOP_TEST_ENV_BASE": "from-adl",
		"LOOP_TEST_ENV_NEW":  "added",
	})
	m := map[string]string{}
	for _, kv := range got {
		k, v, ok := cutEnv(kv)
		if !ok {
			t.Fatalf("bad kv %q", kv)
		}
		m[k] = v
	}
	if m["LOOP_TEST_ENV_BASE"] != "from-adl" {
		t.Fatalf("override = %q", m["LOOP_TEST_ENV_BASE"])
	}
	if m["LOOP_TEST_ENV_NEW"] != "added" {
		t.Fatalf("new = %q", m["LOOP_TEST_ENV_NEW"])
	}
}

func TestApplyCmdEnv(t *testing.T) {
	t.Setenv("LOOP_CMD_ENV_TEST", "host")
	cmd := exec.Command("true")
	applyCmdEnv(cmd, "claude-code", "/tmp/session-config", map[string]string{
		"LOOP_CMD_ENV_TEST": "adl",
	})
	m := envMap(cmd.Env)
	if m["LOOP_CMD_ENV_TEST"] != "adl" {
		t.Fatalf("override = %q", m["LOOP_CMD_ENV_TEST"])
	}
	if m["CLAUDE_CONFIG_DIR"] != "/tmp/session-config" {
		t.Fatalf("CLAUDE_CONFIG_DIR = %q", m["CLAUDE_CONFIG_DIR"])
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
