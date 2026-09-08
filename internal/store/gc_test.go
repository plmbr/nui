// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"nui/internal/model"
)

func TestRegisterAndRemoveSessionRunLogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NUI_DATA_DIR", "")
	SetRunsDirOverride("")
	t.Cleanup(func() { SetRunsDirOverride("") })

	if err := RegisterSessionRun("sess-1", "run-a"); err != nil {
		t.Fatal(err)
	}
	if err := RegisterSessionRun("sess-1", "run-b"); err != nil {
		t.Fatal(err)
	}
	ids, err := SessionRunIDs("sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids = %v", ids)
	}

	for _, id := range ids {
		path, err := RunLogPath(id)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := RemoveSessionRunLogs("sess-1"); err != nil {
		t.Fatal(err)
	}
	ids, err = SessionRunIDs("sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty index, got %v", ids)
	}
	for _, id := range []string{"run-a", "run-b"} {
		path, _ := RunLogPath(id)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still present", id)
		}
	}
}

func TestGarbageCollect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NUI_DATA_DIR", "")
	SetRunsDirOverride("")
	t.Cleanup(func() { SetRunsDirOverride("") })

	dir, err := UserDir()
	if err != nil {
		t.Fatal(err)
	}

	liveID := "live-session"
	if err := SaveData(Data{
		Sessions:        []model.Session{{ID: liveID, Name: "Live", AgentType: "cli"}},
		AgentSessions:   map[string]string{liveID: "agent-1", "dead-session": "agent-x"},
		SessionMessages: map[string][]model.ChatMessage{"dead-session": {{ID: "m1", Role: "user", Content: "x"}}},
	}); err != nil {
		t.Fatal(err)
	}

	orphanSess := filepath.Join(dir, "sessions", "orphan-sess")
	orphanWS := filepath.Join(dir, "workspaces", "orphan-ws")
	liveSess := filepath.Join(dir, "sessions", liveID)
	for _, p := range []string{orphanSess, orphanWS, liveSess} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(orphanSess, "manifest.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tmpPath := filepath.Join(dir, "nui-save-abc.tmp")
	if err := os.WriteFile(tmpPath, []byte("leftover"), 0o600); err != nil {
		t.Fatal(err)
	}
	legacyClaudeSnap := filepath.Join(dir, ".claude-snapshot.json")
	legacyPiOverride := filepath.Join(dir, ".pi-settings-override.json")
	for _, p := range []string{legacyClaudeSnap, legacyPiOverride} {
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	runsDir, err := RunsDir()
	if err != nil {
		t.Fatal(err)
	}
	legacyRun := filepath.Join(runsDir, "legacy-run.jsonl")
	if err := os.WriteFile(legacyRun, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RegisterSessionRun("dead-session", "indexed-dead"); err != nil {
		t.Fatal(err)
	}
	deadPath, err := RunLogPath("indexed-dead")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(deadPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RegisterSessionRun(liveID, "indexed-live"); err != nil {
		t.Fatal(err)
	}
	livePath, err := RunLogPath("indexed-live")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(livePath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	uploadOrphan := filepath.Join(os.TempDir(), "nui-uploads", "orphan-upload-"+t.Name())
	if err := os.MkdirAll(uploadOrphan, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(os.TempDir(), "nui-uploads", "orphan-upload-"+t.Name())) })

	updateDir, err := os.MkdirTemp("", "nui-update-*")
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(updateDir, old, old); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(updateDir) })

	bridge := filepath.Join(dir, "vscode-bridge-instances")
	if err := os.MkdirAll(bridge, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := GarbageCollect(GCOptions{UpdateMaxAge: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if result.TempFiles < 1 {
		t.Fatalf("tempFiles = %d", result.TempFiles)
	}
	if result.SessionDirs != 1 {
		t.Fatalf("sessionDirs = %d", result.SessionDirs)
	}
	if result.WorkspaceDirs != 1 {
		t.Fatalf("workspaceDirs = %d", result.WorkspaceDirs)
	}
	if result.RunLogs < 2 {
		t.Fatalf("runLogs = %d (want legacy+indexed-dead)", result.RunLogs)
	}
	if result.UploadDirs < 1 {
		t.Fatalf("uploadDirs = %d", result.UploadDirs)
	}
	if result.UpdateDirs < 1 {
		t.Fatalf("updateDirs = %d", result.UpdateDirs)
	}
	if result.EmptyBridgeDirs != 1 {
		t.Fatalf("emptyBridgeDirs = %d", result.EmptyBridgeDirs)
	}
	if result.LegacyDockerFiles < 2 {
		t.Fatalf("legacyDockerFiles = %d", result.LegacyDockerFiles)
	}
	if result.DataKeysPruned < 2 {
		t.Fatalf("dataKeysPruned = %d", result.DataKeysPruned)
	}

	if _, err := os.Stat(orphanSess); !os.IsNotExist(err) {
		t.Fatal("orphan session dir still present")
	}
	if _, err := os.Stat(liveSess); err != nil {
		t.Fatal("live session dir removed")
	}
	if _, err := os.Stat(livePath); err != nil {
		t.Fatal("live run log removed")
	}
	if _, err := os.Stat(legacyRun); !os.IsNotExist(err) {
		t.Fatal("legacy run still present")
	}
	if _, err := os.Stat(deadPath); !os.IsNotExist(err) {
		t.Fatal("dead indexed run still present")
	}
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatal("tmp still present")
	}
	if _, err := os.Stat(bridge); !os.IsNotExist(err) {
		t.Fatal("empty bridge dir still present")
	}
	if _, err := os.Stat(legacyClaudeSnap); !os.IsNotExist(err) {
		t.Fatal("legacy claude snapshot still present")
	}
	if _, err := os.Stat(legacyPiOverride); !os.IsNotExist(err) {
		t.Fatal("legacy pi override still present")
	}

	data, err := LoadData()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := data.AgentSessions["dead-session"]; ok {
		t.Fatal("stale agentSessions key kept")
	}
	if _, ok := data.SessionMessages["dead-session"]; ok {
		t.Fatal("stale sessionMessages key kept")
	}
	if _, ok := data.AgentSessions[liveID]; !ok {
		t.Fatal("live agentSessions key removed")
	}

	// index should still list live run
	ids, err := SessionRunIDs(liveID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "indexed-live" {
		t.Fatalf("live index = %v", ids)
	}
	deadIDs, err := SessionRunIDs("dead-session")
	if err != nil {
		t.Fatal(err)
	}
	if len(deadIDs) != 0 {
		t.Fatalf("dead index = %v", deadIDs)
	}
}

func TestGarbageCollectDryRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NUI_DATA_DIR", "")
	SetRunsDirOverride("")
	t.Cleanup(func() { SetRunsDirOverride("") })

	dir, err := UserDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveData(Data{Sessions: []model.Session{}}); err != nil {
		t.Fatal(err)
	}
	tmpPath := filepath.Join(dir, "123.tmp")
	if err := os.WriteFile(tmpPath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := GarbageCollect(GCOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.TempFiles != 1 {
		t.Fatalf("tempFiles = %d", result.TempFiles)
	}
	if _, err := os.Stat(tmpPath); err != nil {
		t.Fatal("dry-run deleted temp file")
	}
}
