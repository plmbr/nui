// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"loop/internal/loopclient"
)

func ensureLoopServer(ctx context.Context, client *loopclient.Client, spawn bool) error {
	if err := client.Health(ctx); err == nil {
		return nil
	}
	if !spawn {
		return fmt.Errorf("loop server not reachable at %s (start with `loop ui` or pass --spawn)", client.BaseURL)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "ui", "--port", "8080", "--no-browser")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn loop ui: %w", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.Health(ctx); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for loop server at %s", client.BaseURL)
}
