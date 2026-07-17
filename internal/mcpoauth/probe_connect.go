// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"nui/internal/model"
)

// ProbeConnectFailures attempts to connect to each remote MCP server and returns
// user-facing error messages for servers that fail.
func ProbeConnectFailures(ctx context.Context, servers []model.ADLMCPServer) []string {
	var failures []string
	for _, srv := range servers {
		if IsBuiltin(srv) || !IsRemote(srv) {
			continue
		}
		name := strings.TrimSpace(srv.Name)
		if name == "" {
			name = ServerKey(srv)
		}
		probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		session, err := ConnectRemote(probeCtx, srv)
		cancel()
		if err != nil {
			failures = append(failures, FormatConnectFailure(name, err))
			continue
		}
		listCtx, listCancel := context.WithTimeout(ctx, probeTimeout)
		_, err = session.ListTools(listCtx, &mcp.ListToolsParams{})
		listCancel()
		_ = session.Close()
		if err != nil {
			failures = append(failures, FormatConnectFailure(name, err))
		}
	}
	return failures
}

// FormatConnectFailure returns a user-facing message for an MCP connection error.
func FormatConnectFailure(name string, err error) string {
	if errors.Is(err, ErrNeedsAuth) {
		return fmt.Sprintf("MCP server %q needs authentication — connect in Customize → MCP Servers", name)
	}
	return fmt.Sprintf("MCP server %q failed to connect: %v", name, err)
}
