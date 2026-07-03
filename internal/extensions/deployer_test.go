// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/extensions"
)

func TestAgentDeployersLoaded(t *testing.T) {
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
	items := reg.AllDeployers()
	if len(items) != 1 {
		t.Fatalf("deployers = %d, want 1", len(items))
	}
	if items[0].ID != "ext:deploy-pack/docker" {
		t.Fatalf("id = %q", items[0].ID)
	}
	ref, err := reg.ResolveDeployer("ext:deploy-pack/docker")
	if err != nil {
		t.Fatal(err)
	}
	if len(ref.Deployer.Command) == 0 {
		t.Fatal("expected expanded command")
	}
}

func TestInvokeDeployerStub(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	extDir := filepath.Join(home, ".loop", "extensions", "stub-deployer")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := `#!/usr/bin/env python3
import json, sys
req = json.loads(sys.stdin.readline())
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
	ref, err := reg.ResolveDeployer("ext:stub-deployer/stub")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := extensions.InvokeDeployer(ref, extensions.DeployRequest{
		Action:     "deploy",
		DeployerID: ref.ID,
		AgentID:    "my-agent",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.DeploymentID != "test-deploy" {
		t.Fatalf("deploymentId = %q", resp.DeploymentID)
	}
}
