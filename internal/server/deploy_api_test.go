// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/extensions"
)

func TestHandleAgentDeployers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	extDir := filepath.Join(home, ".loop", "extensions", "deploy-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
name: deploy-pack
version: 1.0.0
contributions:
  aiAssets:
    agentDeployers:
      - name: docker
        description: Docker deployer
        command: ["python3", "deploy.py"]
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	extensions.Default = reg

	req := httptest.NewRequest(http.MethodGet, "/api/agent-deployers", nil)
	rec := httptest.NewRecorder()
	handleAgentDeployers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Deployers []struct {
			ID string `json:"id"`
		} `json:"deployers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Deployers) != 1 || body.Deployers[0].ID != "ext:deploy-pack/docker" {
		t.Fatalf("deployers = %+v", body.Deployers)
	}
}

func TestHandleAgentDeployRoute(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	agentsDir := filepath.Join(home, ".loop", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	agentYAML := `adl: "1.0"
id: my-agent
name: My Agent
harness:
  type: claude-code
`
	if err := os.WriteFile(filepath.Join(agentsDir, "my-agent.yaml"), []byte(agentYAML), 0644); err != nil {
		t.Fatal(err)
	}

	extDir := filepath.Join(home, ".loop", "extensions", "stub-deployer")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := `#!/usr/bin/env python3
import json, sys
json.dump({"ok": True, "deploymentId": "test-deploy", "status": "ready", "message": "stub ok"}, sys.stdout)
sys.stdout.write("\n")
`
	if err := os.WriteFile(filepath.Join(extDir, "stub.py"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
name: stub-deployer
version: 1.0.0
contributions:
  aiAssets:
    agentDeployers:
      - name: stub
        command: ["python3", "stub.py"]
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	extensions.Default = reg

	req := httptest.NewRequest(http.MethodPost, "/api/agents/my-agent/deploy", strings.NewReader(`{"deployerId":"ext:stub-deployer/stub"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleAgentFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var result struct {
		DeploymentID string `json:"deploymentId"`
		Message      string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.DeploymentID != "test-deploy" {
		t.Fatalf("deploymentId = %q", result.DeploymentID)
	}
}
