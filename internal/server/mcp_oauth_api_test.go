// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nui/internal/mcpoauth"
)

func TestHandleMCPOAuthRedirectURI(t *testing.T) {
	mcpoauth.SetListenPort(9090)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/mcp-oauth/redirect-uri", nil)
	handleMCPOAuthRedirectURI(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !contains(rec.Body.String(), "9090") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
