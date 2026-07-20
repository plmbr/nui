// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
