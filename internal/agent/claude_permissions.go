// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"loop/internal/hitl"
	"loop/internal/loopclient"
)

type claudePermissionRequest struct {
	RequestID string
	ToolName  string
	ToolInput map[string]any
}

func parseClaudePermissionRequest(line []byte) (claudePermissionRequest, bool) {
	var envelope struct {
		Type      string          `json:"type"`
		RequestID string          `json:"request_id"`
		Request   json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return claudePermissionRequest{}, false
	}
	switch envelope.Type {
	case "sdk_control_request", "control_request", "control":
	default:
		return claudePermissionRequest{}, false
	}
	if len(envelope.Request) == 0 {
		return claudePermissionRequest{}, false
	}

	var req struct {
		Subtype   string         `json:"subtype"`
		RequestID string         `json:"request_id"`
		ToolName  string         `json:"tool_name"`
		ToolInput map[string]any `json:"tool_input"`
		Input     map[string]any `json:"input"`
	}
	if err := json.Unmarshal(envelope.Request, &req); err != nil {
		return claudePermissionRequest{}, false
	}
	switch req.Subtype {
	case "permission", "can_use_tool":
	default:
		return claudePermissionRequest{}, false
	}
	requestID := strings.TrimSpace(envelope.RequestID)
	if requestID == "" {
		requestID = strings.TrimSpace(req.RequestID)
	}
	if requestID == "" || req.ToolName == "" {
		return claudePermissionRequest{}, false
	}
	toolInput := req.ToolInput
	if toolInput == nil {
		toolInput = req.Input
	}
	return claudePermissionRequest{
		RequestID: requestID,
		ToolName:  req.ToolName,
		ToolInput: toolInput,
	}, true
}

func waitForClaudeToolApproval(ctx context.Context, req RunRequest, perm claudePermissionRequest) (allow bool, denyMessage string, err error) {
	sessionID := req.LoopSessionID
	if sessionID == "" {
		sessionID = req.SessionID
	}
	apiURL := defaultLoopAPIURL()
	if v := strings.TrimSpace(req.Env[EnvLoopAPIURL]); v != "" {
		apiURL = v
	}

	title := fmt.Sprintf("Allow %s?", perm.ToolName)
	message := approvalMessageForTool(perm.ToolName, perm.ToolInput)
	payload := map[string]any{
		"title":     title,
		"message":   message,
		"toolName":  perm.ToolName,
		"toolInput": perm.ToolInput,
	}

	client := loopclient.New(apiURL)
	hitlReq, err := client.CreateHITLRequest(ctx, hitl.CreateInput{
		SessionID: sessionID,
		RunID:     req.RunID,
		Kind:      hitl.KindApproval,
		Payload:   payload,
	})
	if err != nil {
		return false, "", err
	}

	resp, err := client.WaitHITLRequest(ctx, hitlReq.RequestID)
	if err != nil {
		return false, "", err
	}
	if resp.Status == hitl.StatusDeclined || resp.Status == hitl.StatusCancelled {
		if msg, _ := resp.Answers["message"].(string); strings.TrimSpace(msg) != "" {
			return false, msg, nil
		}
		return false, "declined by user", nil
	}
	if approved, ok := resp.Answers["approved"].(bool); ok && !approved {
		if msg, _ := resp.Answers["message"].(string); strings.TrimSpace(msg) != "" {
			return false, msg, nil
		}
		return false, "declined by user", nil
	}
	return true, "", nil
}

func approvalMessageForTool(toolName string, toolInput map[string]any) string {
	switch toolName {
	case "Bash":
		if cmd, _ := toolInput["command"].(string); strings.TrimSpace(cmd) != "" {
			return cmd
		}
	case "Write", "Edit", "MultiEdit", "NotebookEdit":
		if path, _ := toolInput["file_path"].(string); strings.TrimSpace(path) != "" {
			return path
		}
	}
	if len(toolInput) == 0 {
		return ""
	}
	b, _ := json.Marshal(toolInput)
	return string(b)
}

func writeClaudePermissionResponse(stdin io.Writer, perm claudePermissionRequest, allow bool, denyMessage string) error {
	responseBody := map[string]any{}
	if allow {
		responseBody["behavior"] = "allow"
		if perm.ToolInput != nil {
			responseBody["updatedInput"] = perm.ToolInput
		} else {
			responseBody["updatedInput"] = map[string]any{}
		}
	} else {
		responseBody["behavior"] = "deny"
		if strings.TrimSpace(denyMessage) == "" {
			denyMessage = "declined by user"
		}
		responseBody["message"] = denyMessage
	}

	msg := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": perm.RequestID,
			"response":   responseBody,
		},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	_, err = stdin.Write(payload)
	return err
}

func (s *persistentClaudeSession) handleClaudePermissionRequest(ctx context.Context, req RunRequest, line []byte) (bool, error) {
	perm, ok := parseClaudePermissionRequest(line)
	if !ok {
		return false, nil
	}
	allow, denyMessage, err := waitForClaudeToolApproval(ctx, req, perm)
	if err != nil {
		return true, err
	}
	if err := writeClaudePermissionResponse(s.stdin, perm, allow, denyMessage); err != nil {
		return true, fmt.Errorf("write claude permission response: %w", err)
	}
	return true, nil
}
