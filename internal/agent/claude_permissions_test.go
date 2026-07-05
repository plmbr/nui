// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"loop/internal/hitl"
)

func TestParseClaudePermissionRequest(t *testing.T) {
	line := []byte(`{"type":"sdk_control_request","request_id":"perm_1","request":{"subtype":"permission","tool_name":"Bash","tool_input":{"command":"mkdir foo"}}}`)
	perm, ok := parseClaudePermissionRequest(line)
	if !ok {
		t.Fatal("expected permission request")
	}
	if perm.RequestID != "perm_1" || perm.ToolName != "Bash" {
		t.Fatalf("perm = %+v", perm)
	}
	cmd, _ := perm.ToolInput["command"].(string)
	if cmd != "mkdir foo" {
		t.Fatalf("tool input = %+v", perm.ToolInput)
	}
}

func TestWriteClaudePermissionResponseAllow(t *testing.T) {
	var buf bytes.Buffer
	perm := claudePermissionRequest{
		RequestID: "perm_1",
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "echo hi"},
	}
	if err := writeClaudePermissionResponse(&buf, perm, true, ""); err != nil {
		t.Fatal(err)
	}
	var msg map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &msg); err != nil {
		t.Fatal(err)
	}
	resp, _ := msg["response"].(map[string]any)
	inner, _ := resp["response"].(map[string]any)
	if inner["behavior"] != "allow" {
		t.Fatalf("response = %+v", inner)
	}
}

func TestApprovalMessageForToolBash(t *testing.T) {
	msg := approvalMessageForTool("Bash", map[string]any{"command": "mkdir foo"})
	if msg != "mkdir foo" {
		t.Fatalf("msg = %q", msg)
	}
}

func TestParseClaudePermissionRequestIgnoresOtherMessages(t *testing.T) {
	_, ok := parseClaudePermissionRequest([]byte(`{"type":"assistant","message":{}}`))
	if ok {
		t.Fatal("expected non-permission line to be ignored")
	}
}

func TestShouldAutoApproveBeforeHITL(t *testing.T) {
	req := RunRequest{
		ToolApprovalPolicy: hitl.ToolApprovalDenylist,
		ToolApprovalTools:  []string{"Bash", "Write"},
	}
	if !hitl.ShouldAutoApproveTool("Read", req.ToolApprovalPolicy, req.ToolApprovalTools) {
		t.Fatal("Read should auto-approve under denylist")
	}
	if hitl.ShouldAutoApproveTool("Bash", req.ToolApprovalPolicy, req.ToolApprovalTools) {
		t.Fatal("Bash should require approval under denylist")
	}
}

func TestParseClaudePermissionRequestCanUseTool(t *testing.T) {
	line := []byte(`{"type":"control_request","request_id":"perm_2","request":{"subtype":"can_use_tool","tool_name":"Write","input":{"file_path":"foo.txt"}}}`)
	perm, ok := parseClaudePermissionRequest(line)
	if !ok {
		t.Fatal("expected can_use_tool request")
	}
	if perm.RequestID != "perm_2" {
		t.Fatalf("request id = %q", perm.RequestID)
	}
	if !strings.Contains(perm.ToolName, "Write") {
		t.Fatalf("perm = %+v", perm)
	}
}
