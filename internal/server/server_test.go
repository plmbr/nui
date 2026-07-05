// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsUIRoute(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/launch", true},
		{"/customize", true},
		{"/schedules", true},
		{"/sessions/abc", true},
		{"/sessions/new", true},
		{"/api/sessions", false},
		{"/health", false},
	}
	for _, tc := range tests {
		if got := isUIRoute(tc.path); got != tc.want {
			t.Errorf("isUIRoute(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %q", rec.Body.String())
	}
}
