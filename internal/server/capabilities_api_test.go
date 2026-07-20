// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleCapabilities(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)
	rec := httptest.NewRecorder()
	handleCapabilities(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body Capabilities
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	// bwrap may or may not be available depending on host; structure must be present.
	if body.Sandbox.Bwrap.Available && body.Sandbox.Bwrap.Path == "" {
		t.Fatalf("bwrap available but path empty: %+v", body.Sandbox.Bwrap)
	}
}

func TestHandleCapabilities_methodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/capabilities", nil)
	rec := httptest.NewRecorder()
	handleCapabilities(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}
