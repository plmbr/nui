// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"nui/internal/browser"
	"nui/internal/nuiclient"
	"nui/internal/server"
)

func attachToRunningServer(ctx context.Context, port int, opts server.StartOptions) error {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := nuiclient.New(baseURL)

	hasLaunch := strings.TrimSpace(opts.AgentType) != "" || strings.TrimSpace(opts.Prompt) != ""
	if !hasLaunch && !opts.Open {
		fmt.Fprintf(os.Stderr, "nui server already running at %s\n", baseURL)
		return nil
	}

	var sessionID string
	if hasLaunch {
		sess, err := client.Launch(ctx, nuiclient.LaunchRequest{
			AgentType:  opts.AgentType,
			WorkingDir: opts.WorkingDir,
			Prompt:     opts.Prompt,
			HideInput:  opts.HideInput,
			Harness:    opts.Harness,
		})
		if err != nil {
			return err
		}
		sessionID = sess.ID
		fmt.Fprintf(os.Stderr, "Created session %q (%s)\n", sess.Name, sess.ID)
	}

	if opts.Open {
		var url string
		if hasLaunch {
			url = fmt.Sprintf("%s/sessions/%s", baseURL, sessionID)
		} else {
			url = baseURL + server.LaunchPath
		}
		if err := browser.Open(url); err != nil {
			fmt.Fprintf(os.Stderr, "warn: open browser: %v\n", err)
		}
	}
	return nil
}
