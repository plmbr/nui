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
		"disabledExtensions": ["corp-pack"],
		"memoryAgentsMode": {"reviewer": "manual"}
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

func TestHandleSettings_uiTheme(t *testing.T) {
	setupTestServerEnv(t)

	putReq := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"uiTheme":"standard"}`))
	putRec := httptest.NewRecorder()
	handleSettings(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d body=%s", putRec.Code, putRec.Body.String())
	}
	var updated store.Settings
	if err := json.Unmarshal(putRec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.UITheme != "standard" {
		t.Fatalf("UITheme = %q, want standard", updated.UITheme)
	}

	badReq := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"uiTheme":"neon"}`))
	badRec := httptest.NewRecorder()
	handleSettings(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid uiTheme status = %d body=%s", badRec.Code, badRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	getRec := httptest.NewRecorder()
	handleSettings(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", getRec.Code)
	}
	var loaded store.Settings
	if err := json.Unmarshal(getRec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.UITheme != "standard" {
		t.Fatalf("persisted UITheme = %q", loaded.UITheme)
	}
}

func TestHandleSettings_disableSloganAnimation(t *testing.T) {
	setupTestServerEnv(t)

	putReq := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"disableSloganAnimation":true}`))
	putRec := httptest.NewRecorder()
	handleSettings(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d body=%s", putRec.Code, putRec.Body.String())
	}
	var updated store.Settings
	if err := json.Unmarshal(putRec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.DisableSloganAnimation == nil || !*updated.DisableSloganAnimation {
		t.Fatalf("DisableSloganAnimation = %v, want true", updated.DisableSloganAnimation)
	}

	loaded, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DisableSloganAnimation == nil || !*loaded.DisableSloganAnimation {
		t.Fatalf("persisted DisableSloganAnimation = %v", loaded.DisableSloganAnimation)
	}
}

func TestHandleState_recentsRoundTrip(t *testing.T) {
	setupTestServerEnv(t)

	putBody := `{
		"recentSessionIds": ["s1", "s2"],
		"recentAgents": [{"agentType":"claude-code","workingDir":"/tmp","usedAt":"2026-01-01T00:00:00Z"}]
	}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/state", strings.NewReader(putBody))
	putRec := httptest.NewRecorder()
	handleState(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d body=%s", putRec.Code, putRec.Body.String())
	}

	saved, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.RecentSessionIDs) != 2 || saved.RecentSessionIDs[0] != "s1" {
		t.Fatalf("RecentSessionIDs = %+v", saved.RecentSessionIDs)
	}
	if len(saved.RecentAgents) != 1 || saved.RecentAgents[0].AgentType != "claude-code" {
		t.Fatalf("RecentAgents = %+v", saved.RecentAgents)
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
