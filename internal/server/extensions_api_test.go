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

	"nui/internal/extensions"
)

func TestHandleExtensions_emptyWhenNil(t *testing.T) {
	prev := extensions.Default
	extensions.Default = nil
	t.Cleanup(func() { extensions.Default = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/extensions", nil)
	rec := httptest.NewRecorder()
	handleExtensions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body []extensions.ExtensionInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Fatalf("expected empty list, got %+v", body)
	}
}

func TestHandleExtensions_listsInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	extDir := filepath.Join(home, ".nui", "extensions", "test-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.plmbr.dev/extension/v1
name: test-pack
version: 1.0.0
displayName: Test Pack
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	prev := extensions.Default
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	extensions.Default = reg
	t.Cleanup(func() { extensions.Default = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/extensions", nil)
	rec := httptest.NewRecorder()
	handleExtensions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body []extensions.ExtensionInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body[0].Name != "test-pack" {
		t.Fatalf("extensions = %+v", body)
	}
}

func TestHandleExtensions_methodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/extensions", nil)
	rec := httptest.NewRecorder()
	handleExtensions(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandleExtensionsReload_loadsRegistry(t *testing.T) {
	home := withTempHome(t)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	extDir := filepath.Join(home, ".nui", "extensions", "reload-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.plmbr.dev/extension/v1
name: reload-pack
version: 1.0.0
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	prev := extensions.Default
	extensions.Default = nil
	t.Cleanup(func() { extensions.Default = prev })

	req := httptest.NewRequest(http.MethodPost, "/api/extensions/reload", nil)
	rec := httptest.NewRecorder()
	handleExtensionsReload(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if extensions.Default == nil {
		t.Fatal("expected extensions.Default to be set after reload")
	}
	info := extensions.Default.Info()
	if len(info) != 1 || info[0].Name != "reload-pack" {
		t.Fatalf("info = %+v", info)
	}
}

func TestHandleExtensionEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	extDir := filepath.Join(home, ".nui", "extensions", "test-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.plmbr.dev/extension/v1
name: test-pack
version: 1.0.0
displayName: Test Pack
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	prev := extensions.Default
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	extensions.Default = reg
	t.Cleanup(func() { extensions.Default = prev })

	putReq := httptest.NewRequest(http.MethodPut, "/api/extensions/test-pack/env", strings.NewReader(`{
		"env": {"EXT_TOKEN": "secret", "NUI_API_URL": "nope"}
	}`))
	putRec := httptest.NewRecorder()
	handleExtensionPath(putRec, putReq)
	if putRec.Code != http.StatusBadRequest {
		t.Fatalf("reserved status = %d body=%s", putRec.Code, putRec.Body.String())
	}

	putReq = httptest.NewRequest(http.MethodPut, "/api/extensions/test-pack/env", strings.NewReader(`{
		"env": {"EXT_TOKEN": "secret", "NUI_MY_EXTENSION_TOKEN": "ok"}
	}`))
	putRec = httptest.NewRecorder()
	handleExtensionPath(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d body=%s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/extensions/test-pack/env", nil)
	getRec := httptest.NewRecorder()
	handleExtensionPath(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", getRec.Code)
	}
	var body extensionEnvResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Env["EXT_TOKEN"] != "secret" {
		t.Fatalf("env = %+v", body.Env)
	}
	if body.Env["NUI_MY_EXTENSION_TOKEN"] != "ok" {
		t.Fatalf("extension-owned key missing: %+v", body.Env)
	}
	if len(body.Keys) != 2 {
		t.Fatalf("keys = %v", body.Keys)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/extensions", nil)
	listRec := httptest.NewRecorder()
	handleExtensions(listRec, listReq)
	var list []extensions.ExtensionInfo
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || len(list[0].EnvKeys) != 2 {
		t.Fatalf("list envKeys = %+v", list)
	}
}
