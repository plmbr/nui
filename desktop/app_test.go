// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"net/http"
	"testing"
	"time"
)

func TestStartServerBeforeWindow(t *testing.T) {
	app := &App{port: 18082}
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
