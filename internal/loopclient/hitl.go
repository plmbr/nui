// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package loopclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"loop/internal/hitl"
)

func (c *Client) CreateHITLRequest(ctx context.Context, in hitl.CreateInput) (*hitl.Request, error) {
	var out hitl.Request
	if err := c.postJSON(ctx, "/api/hitl/requests", in, http.StatusCreated, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetHITLRequest(ctx context.Context, requestID string) (*hitl.Request, error) {
	var out hitl.Request
	path := "/api/hitl/requests/" + requestID
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) WaitHITLRequest(ctx context.Context, requestID string) (*hitl.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/hitl/requests/"+requestID+"/wait", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wait HITL request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out hitl.Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RespondHITLRequest(ctx context.Context, requestID string, in hitl.RespondInput) (*hitl.Response, error) {
	var out hitl.Response
	path := "/api/hitl/requests/" + requestID + "/respond"
	if err := c.postJSON(ctx, path, in, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListPendingHITLRequests(ctx context.Context, sessionID string) ([]hitl.Request, error) {
	var out []hitl.Request
	path := "/api/hitl/requests?pending=true"
	if sessionID != "" {
		path += "&sessionId=" + sessionID
	}
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}
