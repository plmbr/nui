// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"loop/internal/model"
)

func TestHandleSessionMentionsEmptyWorkingDirUsesCWD(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	mu.Lock()
	sessions = []model.Session{{
		ID:         "sess-empty-wd",
		Name:       "Test",
		WorkingDir: "",
		AgentType:  "claude-code",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		sessions = nil
		mu.Unlock()
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/sess-empty-wd/mentions?parent=builtin:files", nil)
	rec := httptest.NewRecorder()
	handleSessionMentions(rec, req, "sess-empty-wd")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []struct {
			Value string `json:"value"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) == 0 {
		t.Fatalf("expected files under cwd, got %+v", resp.Items)
	}
}

func TestEffectiveWorkingDir(t *testing.T) {
	dir := t.TempDir()
	if got, err := effectiveWorkingDir(dir); err != nil || got != dir {
		t.Fatalf("explicit = %q, err = %v", got, err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	got, err := effectiveWorkingDir("")
	if err != nil {
		t.Fatal(err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		gotResolved = got
	}
	dirResolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		dirResolved = dir
	}
	if gotResolved != dirResolved {
		t.Fatalf("default cwd = %q, want %q", gotResolved, dirResolved)
	}
}

func TestHandleSessionMentions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	sessions = []model.Session{{
		ID:         "sess-1",
		Name:       "Test",
		WorkingDir: dir,
		AgentType:  "claude-code",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		sessions = nil
		mu.Unlock()
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/sess-1/mentions?parent=", nil)
	rec := httptest.NewRecorder()
	handleSessionMentions(rec, req, "sess-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []struct {
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) == 0 || resp.Items[0].Value != "builtin:files" {
		t.Fatalf("items: %+v", resp.Items)
	}
}
