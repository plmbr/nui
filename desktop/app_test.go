// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"nui/ui"
)

func TestStartServerBeforeWindow(t *testing.T) {
	app := &App{port: 18082, uiFS: ui.DistFS()}
	if err := app.startServer(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		app.shutdown(t.Context())
	})

	base, startErr := app.getBaseURL()
	if startErr != "" {
		t.Fatalf("startErr=%q", startErr)
	}
	if base == "" {
		t.Fatal("expected baseURL after startServer")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(base + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestStartServerScansWhenPortBusy(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:18082")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	app := &App{port: 18082, uiFS: ui.DistFS()}
	if err := app.startServer(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		app.shutdown(t.Context())
	})

	base, startErr := app.getBaseURL()
	if startErr != "" {
		t.Fatalf("startErr=%q", startErr)
	}
	if !strings.Contains(base, ":18083") {
		t.Fatalf("expected server on :18083, got %q", base)
	}
}
