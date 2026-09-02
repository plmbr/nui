// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvSetGetUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"env", "set", "MY_VAR=hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"env", "get", "MY_VAR"})
	out := captureStdout(t, func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("get = %q", out)
	}

	rootCmd.SetArgs([]string{"env", "unset", "MY_VAR"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"env", "get", "MY_VAR"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not set") {
		t.Fatalf("expected not set error, got %v", err)
	}
}

func TestEnvListMasksSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	rootCmd.SetArgs([]string{"env", "set", "ANTHROPIC_API_KEY=sk-secret"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"env", "list"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	if strings.Contains(out, "sk-secret") {
		t.Fatalf("secret leaked in list: %q", out)
	}
	if !strings.Contains(out, "********") {
		t.Fatalf("expected masked value: %q", out)
	}

	out = captureStdout(t, func() {
		rootCmd.SetArgs([]string{"env", "list", "--reveal"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "sk-secret") {
		t.Fatalf("expected revealed secret: %q", out)
	}
}

func TestEnvReservedKeyRejected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	rootCmd.SetArgs([]string{"env", "set", "NUI_API_URL=http://example"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("expected reserved key error, got %v", err)
	}
}

func TestExtensionEnvSetList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	extDir := filepath.Join(home, ".nui", "extensions", "cli-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.plmbr.dev/extension/v1
name: cli-pack
version: 1.0.0
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"extension", "env", "set", "cli-pack", "TOKEN=abc"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"extension", "env", "list", "cli-pack"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "TOKEN") || !strings.Contains(out, "abc") {
		t.Fatalf("list output = %q", out)
	}
}

func TestAgentAddOverwriteRequiresYes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	agentPath := filepath.Join(home, "my-agent.yaml")
	content := `adl: "1.0"
id: cli-test-agent
name: CLI Test Agent
harness:
  type: claude-code
`
	if err := os.WriteFile(agentPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs([]string{"agent", "add", agentPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"agent", "add", agentPath})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected already exists error, got %v", err)
	}

	rootCmd.SetArgs([]string{"agent", "add", agentPath, "-y"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}
