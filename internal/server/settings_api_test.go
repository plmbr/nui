// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nui/internal/store"
)

func TestHandleSettings_getAndPut(t *testing.T) {
	setupTestServerEnv(t)

	getReq := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	getRec := httptest.NewRecorder()
	handleSettings(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d body=%s", getRec.Code, getRec.Body.String())
	}
	var initial store.Settings
	if err := json.Unmarshal(getRec.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}

	putBody := `{
		"defaultAgentType": "anthropic",
		"lastAgentType": "claude-code",
		"disabledExtensions": ["corp-pack"],
		"memoryAgentsEnabled": {"reviewer": true}
	}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(putBody))
	putRec := httptest.NewRecorder()
	handleSettings(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d body=%s", putRec.Code, putRec.Body.String())
	}
	var updated store.Settings
	if err := json.Unmarshal(putRec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.DefaultAgentType != "anthropic" {
		t.Fatalf("DefaultAgentType = %q", updated.DefaultAgentType)
	}
	if updated.LastAgentType != "claude-code" {
		t.Fatalf("LastAgentType = %q", updated.LastAgentType)
	}
	if len(updated.DisabledExtensions) != 1 || updated.DisabledExtensions[0] != "corp-pack" {
		t.Fatalf("DisabledExtensions = %+v", updated.DisabledExtensions)
	}
	if updated.MemoryAgentsMode == nil || updated.MemoryAgentsMode["reviewer"] != "manual" {
		t.Fatalf("MemoryAgentsMode = %+v", updated.MemoryAgentsMode)
	}

	saved, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if saved.DefaultAgentType != "anthropic" {
		t.Fatalf("persisted DefaultAgentType = %q", saved.DefaultAgentType)
	}
}

func TestHandleSettings_invalidTheme(t *testing.T) {
	setupTestServerEnv(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"theme":"neon"}`))
	rec := httptest.NewRecorder()
	handleSettings(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSettings_methodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/settings", nil)
	rec := httptest.NewRecorder()
	handleSettings(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}
