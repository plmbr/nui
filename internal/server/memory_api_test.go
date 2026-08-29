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

	"nui/internal/memory"
	"nui/internal/store"
)

func TestHandleMemoryUser(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	putReq := httptest.NewRequest(http.MethodPut, "/api/memory/user", strings.NewReader(`{"content":"timezone PST"}`))
	putRec := httptest.NewRecorder()
	handleUserMemory(putRec, putReq)
	if putRec.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d body %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/memory/user", nil)
	getRec := httptest.NewRecorder()
	handleUserMemory(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", getRec.Code)
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(getRec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Content != "timezone PST" {
		t.Fatalf("content = %q", body.Content)
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/memory/user", nil)
	delRec := httptest.NewRecorder()
	handleUserMemory(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d", delRec.Code)
	}
	got, err := memory.ReadUser()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("ReadUser after delete = %q", got)
	}
}

func TestHandleMemoryAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	putReq := httptest.NewRequest(http.MethodPut, "/api/memory/agents/reviewer", strings.NewReader(`{"content":"check tests"}`))
	putRec := httptest.NewRecorder()
	handleAgentMemory(putRec, putReq, "reviewer")
	if putRec.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d", putRec.Code)
	}

	got, err := memory.ReadAgent("reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if got != "check tests" {
		t.Fatalf("ReadAgent() = %q", got)
	}
}

func TestHandleMemoryList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := memory.WriteUser("user note"); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteAgent("demo", "agent note"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSettings(store.Settings{
		MemoryUserMode:   memory.ModeDisabled,
		MemoryAgentsMode: map[string]string{"demo": memory.ModeManual},
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/memory", nil)
	rec := httptest.NewRecorder()
	handleMemory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var summary memory.Summary
	if err := json.NewDecoder(rec.Body).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.User.Mode != memory.ModeDisabled {
		t.Fatalf("expected user memory disabled in summary, got %q", summary.User.Mode)
	}
	if summary.Agents[0].Mode != memory.ModeManual {
		t.Fatalf("expected demo agent manual, got %q", summary.Agents[0].Mode)
	}
	if len(summary.Agents) != 1 || summary.Agents[0].AgentID != "demo" {
		t.Fatalf("agents = %+v", summary.Agents)
	}
	path, _ := memory.UserPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("user path missing: %v", err)
	}
	agentPath, _ := memory.AgentPath("demo")
	if !strings.HasSuffix(agentPath, filepath.Join("agents", "demo.md")) {
		t.Fatalf("agent path = %s", agentPath)
	}
}

func TestHandleSettingsMemoryModes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	putReq := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"memoryUserMode":"auto","memoryAgentsMode":{"demo":"disabled"}}`))
	putRec := httptest.NewRecorder()
	handleSettings(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", putRec.Code, putRec.Body.String())
	}
	settings, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.MemoryUserMode != memory.ModeAuto {
		t.Fatalf("MemoryUserMode = %q", settings.MemoryUserMode)
	}
	if settings.MemoryAgentsMode["demo"] != memory.ModeDisabled {
		t.Fatalf("demo mode = %q", settings.MemoryAgentsMode["demo"])
	}
}
