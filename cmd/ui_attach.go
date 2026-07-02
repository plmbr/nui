// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"loop/internal/browser"
	"loop/internal/loopclient"
	"loop/internal/server"
)

func attachToRunningServer(ctx context.Context, port int, opts server.StartOptions) error {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := loopclient.New(baseURL)

	hasLaunch := strings.TrimSpace(opts.AgentType) != "" || opts.Prompt != "" || opts.Open
	if !hasLaunch {
		fmt.Fprintf(os.Stderr, "Loop server already running at %s\n", baseURL)
		return nil
	}

	sess, err := client.Launch(ctx, loopclient.LaunchRequest{
		AgentType:  opts.AgentType,
		WorkingDir: opts.WorkingDir,
		Prompt:     opts.Prompt,
		HideInput:  opts.HideInput,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Created session %q (%s)\n", sess.Name, sess.ID)

	if opts.Open {
		url := fmt.Sprintf("%s/sessions/%s", baseURL, sess.ID)
		if err := browser.Open(url); err != nil {
			fmt.Fprintf(os.Stderr, "warn: open browser: %v\n", err)
		}
	}
	return nil
}
