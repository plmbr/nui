// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package storageext_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"loop/internal/extensions"
	"loop/internal/memory"
	"loop/internal/model"
	"loop/internal/storageext"
	"loop/internal/store"
)

func testHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	return home
}

func installStorageDemo(t *testing.T, home string) *extensions.Registry {
	t.Helper()
	src := filepath.Join("..", "..", "dev", "extension-examples", "storage-demo")
	extDir := filepath.Join(home, ".loop", "extensions", "storage-demo")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"extension.yaml", "storage-handlers.yaml", "agents.yaml", "storage_host.py"} {
		data, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		mode := os.FileMode(0o644)
		if name == "storage_host.py" {
			mode = 0o755
		}
		if err := os.WriteFile(filepath.Join(extDir, name), data, mode); err != nil {
			t.Fatal(err)
		}
	}
	sdk, err := os.ReadFile(filepath.Join("..", "..", "harness-sdk", "loop_storage.py"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "loop_storage.py"), sdk, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestCoordinatorUserMemorySkipsBuiltin(t *testing.T) {
	home := testHome(t)
	if err := memory.WriteBuiltinUser("builtin-only"); err != nil {
		t.Fatal(err)
	}
	reg := installStorageDemo(t, home)
	coord := storageext.NewCoordinator(reg)
	memory.SetStore(coord)
	t.Cleanup(memory.ResetStore)

	if err := coord.WriteUser("extension-owned"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	got, err := memory.ReadUser()
	if err != nil {
		t.Fatal(err)
	}
	if got != "extension-owned" {
		t.Fatalf("ReadUser() = %q, want extension-owned", got)
	}
	builtin, err := memory.ReadBuiltinUser()
	if err != nil {
		t.Fatal(err)
	}
	if builtin != "builtin-only" {
		t.Fatalf("builtin file should be unchanged, got %q", builtin)
	}
}

func TestCoordinatorSessionReadWrite(t *testing.T) {
	home := testHome(t)
	reg := installStorageDemo(t, home)
	coord := storageext.NewCoordinator(reg)

	agentType := "ext:storage-demo/demo-agent"
	msgs := []model.ChatMessage{{ID: "m1", Role: "user", Content: "hello"}}
	coord.WriteSession("sess-1", agentType, t.TempDir(), "harness-1", msgs)
	time.Sleep(200 * time.Millisecond)

	readMsgs, agentSessionID, err := coord.ReadSession("sess-1", agentType, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if agentSessionID != "harness-1" {
		t.Fatalf("agentSessionID = %q", agentSessionID)
	}
	if len(readMsgs) != 1 || readMsgs[0].Content != "hello" {
		t.Fatalf("messages = %+v", readMsgs)
	}
}

func TestCoordinatorAgentMemoryMergeSingleHandler(t *testing.T) {
	home := testHome(t)
	reg := installStorageDemo(t, home)
	coord := storageext.NewCoordinator(reg)
	memory.SetStore(coord)
	t.Cleanup(memory.ResetStore)

	agentID := "ext:storage-demo/demo-agent"
	if err := memory.WriteAgent(agentID, "agent notes"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	got, err := memory.ReadAgent(agentID)
	if err != nil {
		t.Fatal(err)
	}
	if got != "agent notes" {
		t.Fatalf("ReadAgent() = %q", got)
	}
}

func TestCoordinatorListSummaryUsesExtensionUser(t *testing.T) {
	home := testHome(t)
	reg := installStorageDemo(t, home)
	coord := storageext.NewCoordinator(reg)
	memory.SetStore(coord)
	t.Cleanup(memory.ResetStore)

	if err := coord.WriteUser("cloud user memory"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	summary, err := coord.ListSummary(store.Settings{MemoryUserMode: memory.ModeManual})
	if err != nil {
		t.Fatal(err)
	}
	if summary.User.Size != int64(len("cloud user memory")) {
		t.Fatalf("summary user size = %d", summary.User.Size)
	}
}

func TestStorageRPCClientRoundTrip(t *testing.T) {
	home := testHome(t)
	reg := installStorageDemo(t, home)
	client, err := reg.StorageClient("storage-demo")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	msgs := []model.ChatMessage{{ID: "1", Role: "assistant", Content: "hi"}}
	if err := client.WriteSession(ctx, "demo-sessions", "rpc-sess", "ext:storage-demo/demo-agent", "hs-1", t.TempDir(), msgs); err != nil {
		t.Fatal(err)
	}
	readMsgs, agentSessionID, err := client.ReadSession(ctx, "demo-sessions", "rpc-sess", "ext:storage-demo/demo-agent", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if agentSessionID != "hs-1" || len(readMsgs) != 1 || readMsgs[0].Content != "hi" {
		t.Fatalf("read session = %+v %q", readMsgs, agentSessionID)
	}
	if err := client.WriteUserMemory(ctx, "demo-user-memory", "rpc user", "replace"); err != nil {
		t.Fatal(err)
	}
	content, err := client.ReadUserMemory(ctx, "demo-user-memory")
	if err != nil {
		t.Fatal(err)
	}
	if content != "rpc user" {
		t.Fatalf("user memory = %q", content)
	}
}
