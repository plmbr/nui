// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func installStorageDemoForServer(t *testing.T, home string) {
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
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	storageext.NewCoordinator(reg)
	memory.SetStore(storageext.Default)
	t.Cleanup(memory.ResetStore)
}

func TestHandleSessionMessagesExtensionLazyLoad(t *testing.T) {
	home := withTempHome(t)
	installStorageDemoForServer(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	agentType := "ext:storage-demo/demo-agent"
	sessionID := "ext-sess-1"
	workingDir := t.TempDir()
	seedSession(sessionID, "Ext", agentType, workingDir)

	storageext.Default.WriteSession(sessionID, agentType, workingDir, "", []model.ChatMessage{
		{ID: "m1", Role: "user", Content: "stored externally"},
	})
	time.Sleep(200 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID+"/messages", nil)
	rec := httptest.NewRecorder()
	handleSessionMessages(rec, req, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var msgs []model.ChatMessage
	if err := json.NewDecoder(rec.Body).Decode(&msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Content != "stored externally" {
		t.Fatalf("messages = %+v", msgs)
	}
}

func TestSnapshotOmitsExtensionBackedSession(t *testing.T) {
	home := withTempHome(t)
	installStorageDemoForServer(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	agentType := "ext:storage-demo/demo-agent"
	sessionID := "ext-sess-2"
	seedSession(sessionID, "Ext", agentType, t.TempDir())

	mu.Lock()
	sessionMessages[sessionID] = []model.ChatMessage{{ID: "m1", Role: "user", Content: "local"}}
	agentSessions[sessionID] = "harness-xyz"
	snapshot := snapshotData()
	mu.Unlock()

	if _, ok := snapshot.SessionMessages[sessionID]; ok {
		t.Fatal("extension session messages should be omitted from data.json snapshot")
	}
	if _, ok := snapshot.AgentSessions[sessionID]; ok {
		t.Fatal("extension agent session should be omitted from data.json snapshot")
	}
	if err := store.SaveData(snapshot); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadData()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.SessionMessages) != 0 || len(loaded.AgentSessions) != 0 {
		t.Fatalf("persisted snapshot should omit extension session: %+v %+v", loaded.SessionMessages, loaded.AgentSessions)
	}
}

func TestHandleSessionMessagesPutPersistsToExtension(t *testing.T) {
	home := withTempHome(t)
	installStorageDemoForServer(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	agentType := "ext:storage-demo/demo-agent"
	sessionID := "ext-sess-3"
	workingDir := t.TempDir()
	seedSession(sessionID, "Ext", agentType, workingDir)

	body, _ := json.Marshal([]model.ChatMessage{{ID: "m2", Role: "assistant", Content: "saved via put"}})
	req := httptest.NewRequest(http.MethodPut, "/api/sessions/"+sessionID+"/messages", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleSessionMessages(rec, req, sessionID)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	time.Sleep(200 * time.Millisecond)

	msgs, _, err := storageext.Default.ReadSession(sessionID, agentType, workingDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Content != "saved via put" {
		t.Fatalf("extension messages = %+v", msgs)
	}
}
