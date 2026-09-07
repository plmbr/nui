// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	nuiBashMCPName   = "nui-bash"
	defaultBashTimeout = 60 * time.Second
	maxBashOutput    = 200_000
)

// RunBash starts the nui-bash MCP server on stdio.
func RunBash(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    nuiBashMCPName,
		Version: "1.0.0",
	}, nil)

	registerBashTools(server)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerBashTools(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        "bash",
		Description: "Run a shell command on the host. Default cwd is NUI_WORKING_DIR. May require human approval.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Shell command to run",
				},
				"cwd": map[string]any{
					"type":        "string",
					"description": "Optional working directory (default NUI_WORKING_DIR)",
				},
				"timeout_sec": map[string]any{
					"type":        "integer",
					"description": "Optional timeout in seconds (default 60)",
				},
			},
			"required": []string{"command"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		command := stringArg(args, "command")
		if command == "" {
			return toolError(fmt.Errorf("command is required"))
		}
		cwd := stringArg(args, "cwd")
		if cwd == "" {
			cwd = workingDir()
		}
		if cwd != "" {
			abs, err := resolveHostPath(cwd)
			if err != nil {
				return toolError(err)
			}
			cwd = abs
		}
		timeout := defaultBashTimeout
		if sec := intArg(args, "timeout_sec"); sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		cmd := exec.CommandContext(runCtx, "bash", "-lc", command)
		if cwd != "" {
			cmd.Dir = cwd
		}
		cmd.Env = os.Environ()
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()

		out := truncateOutput(stdout.String())
		errOut := truncateOutput(stderr.String())
		exitCode := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				exitCode = ee.ExitCode()
			} else if runCtx.Err() != nil {
				return toolError(fmt.Errorf("bash timed out after %s", timeout))
			} else {
				return toolError(err)
			}
		}
		var b strings.Builder
		if out != "" {
			b.WriteString(out)
		}
		if errOut != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(errOut)
		}
		if b.Len() == 0 {
			fmt.Fprintf(&b, "exit %d", exitCode)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: b.String()}},
			StructuredContent: map[string]any{
				"exitCode": exitCode,
				"cwd":      cwd,
				"stdout":   out,
				"stderr":   errOut,
			},
			IsError: exitCode != 0,
		}, nil
	})
}

func truncateOutput(s string) string {
	if len(s) <= maxBashOutput {
		return s
	}
	return s[:maxBashOutput] + "\n...[truncated]..."
}
