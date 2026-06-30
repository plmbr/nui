// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/extensions"
	"loop/internal/mentions"
)

func TestMentionExtensionRoots(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".loop", "extensions", "mention-pack")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
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
	sdkPath := filepath.Join("..", "..", "harness-sdk", "loop_mention.py")
	sdkData, err := os.ReadFile(sdkPath)
	if err != nil {
		t.Skip("loop_mention.py not found")
	}
	if err := os.WriteFile(filepath.Join(extDir, "loop_mention.py"), sdkData, 0o644); err != nil {
		t.Fatal(err)
	}
	hostData, err := os.ReadFile(filepath.Join("..", "..", "dev", "extension-examples", "corp-pack", "mention_host.py"))
	if err != nil {
		t.Skip("corp-pack mention_host.py not found")
	}
	if err := os.WriteFile(filepath.Join(extDir, "mention_host.py"), hostData, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	src := reg.MentionSource()
	roots := src.ListExtensionRoots()
	if len(roots) != 1 || roots[0].Value != "ext:mention-pack:refs" {
		t.Fatalf("roots: %+v", roots)
	}

	mentionReg := mentions.NewRegistry(reg.MentionSource())
	if resp, err := mentionReg.List(context.Background(), mentions.ListRequest{
		WorkingDir: t.TempDir(),
		Parent:     "",
		Limit:      20,
	}); err != nil {
		t.Fatal(err)
	} else {
		for _, item := range resp.Items {
			if strings.HasPrefix(item.Value, "ext:") {
				t.Fatalf("extension roots should not appear without agent refs: %+v", resp.Items)
			}
		}
	}

	allowed := map[string]bool{"ext:mention-pack:refs": true}
	regResp, err := mentionReg.List(context.Background(), mentions.ListRequest{
		WorkingDir:            t.TempDir(),
		Parent:                "ext:mention-pack:refs",
		Limit:                 20,
		AllowedExtensionRoots: allowed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(regResp.Items) == 0 {
		t.Fatalf("expected runbooks category: %+v", regResp)
	}
}

func TestMentionExtensionResolve(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".loop", "extensions", "mention-pack")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
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
	sdkPath := filepath.Join("..", "..", "harness-sdk", "loop_mention.py")
	sdkData, err := os.ReadFile(sdkPath)
	if err != nil {
		t.Skip("loop_mention.py not found")
	}
	if err := os.WriteFile(filepath.Join(extDir, "loop_mention.py"), sdkData, 0o644); err != nil {
		t.Fatal(err)
	}
	hostData, err := os.ReadFile(filepath.Join("..", "..", "dev", "extension-examples", "corp-pack", "mention_host.py"))
	if err != nil {
		t.Skip("corp-pack mention_host.py not found")
	}
	if err := os.WriteFile(filepath.Join(extDir, "mention_host.py"), hostData, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	mentionReg := mentions.NewRegistry(reg.MentionSource())
	allowed := map[string]bool{"ext:mention-pack:refs": true}
	msg := "please @ext:mention-pack:refs:runbooks/deploy now"
	got, err := mentionReg.ResolveMessage(context.Background(), t.TempDir(), msg, allowed)
	if err != nil {
		t.Fatal(err)
	}
	if got == msg {
		t.Fatalf("message was not resolved: %q", got)
	}
	if !strings.Contains(got, "deploy checklist") {
		t.Fatalf("resolved = %q", got)
	}
}
