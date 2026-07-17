// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const probeTimeout = 15 * time.Second

// ProbeResult holds the outcome of probing a remote MCP endpoint.
type ProbeResult struct {
	Request  *http.Request
	Response *http.Response
}

// ProbeServer issues a request to the MCP endpoint expecting 401 for OAuth servers.
func ProbeServer(ctx context.Context, serverURL string) (*ProbeResult, error) {
	url := strings.TrimSpace(serverURL)
	if url == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"nui","version":"1.0.0"}}}`))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	client := &http.Client{Timeout: probeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return &ProbeResult{Request: req, Response: resp}, nil
}

// NeedsAuth reports whether the probe response indicates OAuth is required.
func NeedsAuth(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return true
	default:
		return false
	}
}

func drainResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
