// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nui/internal/agent"
	"nui/internal/model"
	"nui/internal/store"
)

func TestHandleCredentials_getAndPut(t *testing.T) {
	setupTestServerEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	getReq := httptest.NewRequest(http.MethodGet, "/api/credentials", nil)
	getRec := httptest.NewRecorder()
	handleCredentials(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d body=%s", getRec.Code, getRec.Body.String())
	}
	var initial credentialsResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if len(initial.Fields) == 0 {
		t.Fatal("expected credential fields")
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(`{
		"env": {"ANTHROPIC_API_KEY": "sk-from-ui", "OPENAI_API_KEY": ""}
	}`))
	putRec := httptest.NewRecorder()
	handleCredentials(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d body=%s", putRec.Code, putRec.Body.String())
	}
	var updated credentialsResponse
	if err := json.Unmarshal(putRec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	var anth *CredentialField
	for i := range updated.Fields {
		if updated.Fields[i].Key == "ANTHROPIC_API_KEY" {
			anth = &updated.Fields[i]
			break
		}
	}
	if anth == nil {
		t.Fatal("missing ANTHROPIC_API_KEY field")
	}
	if anth.Value != "sk-from-ui" || !anth.Configured {
		t.Fatalf("anthropic field = %+v", anth)
	}

	saved, err := store.LoadSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Env["ANTHROPIC_API_KEY"] != "sk-from-ui" {
		t.Fatalf("persisted = %+v", saved.Env)
	}

	h := model.ADLHarness{Type: "api", Provider: "anthropic"}
	if !agent.APIHarnessAvailable(h) {
		t.Fatal("expected anthropic available via secrets")
	}
}

func TestHandleCredentials_rejectsUnknownKey(t *testing.T) {
	setupTestServerEnv(t)
	req := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(`{
		"env": {"NOT_A_KEY": "x"}
	}`))
	rec := httptest.NewRecorder()
	handleCredentials(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleCredentials_methodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/credentials", nil)
	rec := httptest.NewRecorder()
	handleCredentials(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}
