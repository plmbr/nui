package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"nui/internal/server"
	"nui/ui"
)

func TestInstanceStartBackgroundShutdown(t *testing.T) {
	inst, err := server.NewInstance(server.ListenConfig{
		Port:    18081,
		Host:    "127.0.0.1",
		UIFiles: ui.DistFS(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := inst.StartBackground(); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(inst.URL() + "/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := inst.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}
