// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentAddLocalFile(t *testing.T) {
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

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"agent", "add", agentPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	installed := filepath.Join(home, ".nui", "agents", "cli-test-agent.yaml")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("installed file missing: %v", err)
	}
}

func TestAgentListWithMockServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/agent-types", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "anthropic", "label": "Anthropic", "harness": "api", "available": true, "isBuiltin": true, "source": "builtin"},
			{"id": "claude-code", "label": "Claude Code", "harness": "claude-code", "available": true, "isBuiltin": true, "source": "builtin"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	agentListURL = srv.URL
	t.Cleanup(func() { agentListURL = "" })
	rootCmd.SetArgs([]string{"agent", "list"})
	runErr := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	if runErr != nil {
		t.Fatal(runErr)
	}
	if !strings.Contains(out, "anthropic") || !strings.Contains(out, "claude-code") {
		t.Fatalf("output = %q", out)
	}
}

func TestRunHelp(t *testing.T) {
	cmd := NewRunCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestScheduleHelp(t *testing.T) {
	cmd := NewScheduleCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}
