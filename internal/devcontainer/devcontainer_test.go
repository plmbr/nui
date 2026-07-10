// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package devcontainer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindConfig(t *testing.T) {
	dir := t.TempDir()
	devDir := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devDir, devcontainerJSONName), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FindConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := ".devcontainer/devcontainer.json"
	if got != want {
		t.Fatalf("FindConfig() = %q, want %q", got, want)
	}
}

func TestFindConfigAltPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".devcontainer.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := FindConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != altConfigRel {
		t.Fatalf("FindConfig() = %q, want %q", got, altConfigRel)
	}
}

func TestFindConfigMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := FindConfig(dir)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseUpOutput(t *testing.T) {
	stdout := `[1234 ms] Container started
{"outcome":"success","containerId":"abc123","remoteUser":"vscode","remoteWorkspaceFolder":"/workspaces/proj"}`
	result, err := parseUpOutput(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "success" || result.ContainerID != "abc123" {
		t.Fatalf("result = %+v", result)
	}
	if result.RemoteUser != "vscode" || result.RemoteWorkspaceFolder != "/workspaces/proj" {
		t.Fatalf("remote fields = %+v", result)
	}
}

func TestParseUpOutputNoJSON(t *testing.T) {
	_, err := parseUpOutput("Container started\n")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseUpOutputError(t *testing.T) {
	stdout := `[1234 ms] An error occurred setting up the container.
{"outcome":"error","message":"Command failed: docker ps -q -a --filter label=devcontainer.local_folder=/tmp/ws","description":"An error occurred setting up the container."}`
	result, err := parseUpOutput(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "error" {
		t.Fatalf("outcome = %q", result.Outcome)
	}
	upErr := upResultError(result)
	if upErr == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(upErr.Error(), "Docker is not running") {
		t.Fatalf("error = %v", upErr)
	}
}

func TestParseDockerPort(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"127.0.0.1:32768", 32768},
		{"0.0.0.0:32768\n[::]:32768", 32768},
	}
	for _, tc := range tests {
		got, err := parseDockerPort(tc.in)
		if err != nil {
			t.Fatalf("parseDockerPort(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseDockerPort(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestResolveConfigPathExplicit(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".devcontainer", devcontainerJSONName)
	if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveConfigPath(dir, defaultConfigRel)
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultConfigRel {
		t.Fatalf("ResolveConfigPath() = %q, want %q", got, defaultConfigRel)
	}
}
