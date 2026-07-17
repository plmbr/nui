// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"nui/internal/nuiclient"
)

func ensureNuiServer(ctx context.Context, client *nuiclient.Client, spawn bool) error {
	if err := client.Health(ctx); err == nil {
		return nil
	}
	if !spawn {
		return fmt.Errorf("nui server not reachable at %s (start with `nui ui` or pass --spawn)", client.BaseURL)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "ui", "--port", "8080", "--no-browser")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn nui ui: %w", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.Health(ctx); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for nui server at %s", client.BaseURL)
}
