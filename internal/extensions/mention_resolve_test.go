// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"os"
	"path/filepath"
	"testing"

	"nui/internal/extensions"
	"nui/internal/model"
)

func TestExpandMentionProviders(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".nui", "extensions", "mention-pack")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.dev/extension/v1
name: mention-pack
version: 1.0.0
contributions:
  mentionProviders:
    source:
      file: mention-providers.yaml
    runtime:
      transport: stdio
      command: ["python3", "mention_host.py"]
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "mention-providers.yaml"), []byte(`mentionProviders:
  - id: refs
    displayName: References
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	roots, err := reg.ExpandMentionProviders([]model.ADLMentionProvider{
		{Ref: "ext:mention-pack/refs"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || roots[0] != "ext:mention-pack:refs" {
		t.Fatalf("roots: %+v", roots)
	}
}
