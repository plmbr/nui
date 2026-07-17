// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"os"
	"path/filepath"
	"testing"

	"nui/internal/extensions"
)

func TestAllowedMentionRootsForAgent(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".nui", "extensions", "nbi")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.dev/extension/v1
name: nbi
version: 1.0.0
contributions:
  mentionProviders:
    source:
      file: mention-providers.yaml
    runtime:
      transport: stdio
      command: ["python3", "mention_host.py"]
  agents:
    source:
      file: agents.yaml
`
	agents := `agents:
  - id: notebook-agent
    name: Notebook Agent
    harness:
      type: claude-code
    aiAssets:
      mentionProviders:
        - ref: ext:nbi/nbi-kernels
`
	providers := `mentionProviders:
  - id: nbi-kernels
    displayName: Notebook / Kernel
`
	for name, content := range map[string]string{
		"extension.yaml":         manifest,
		"agents.yaml":            agents,
		"mention-providers.yaml": providers,
	} {
		if err := os.WriteFile(filepath.Join(extDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	extensions.Default = reg

	allowed := allowedMentionRootsForAgent("ext:nbi/notebook-agent")
	if len(allowed) != 1 || !allowed["ext:nbi:nbi-kernels"] {
		t.Fatalf("allowed: %+v", allowed)
	}
	if roots := allowedMentionRootsForAgent("claude-code"); len(roots) != 0 {
		t.Fatalf("builtin agent should have no extension mentions: %+v", roots)
	}
}
