// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"nui/internal/hitl"
	"nui/internal/model"
)

const (
	claudeHitlBridgeScript    = "hitl-bridge.sh"
	claudeVizBridgeScript     = "viz-bridge.sh"
	claudenuiHitlAllowedTool = "mcp__nui-hitl__*"
	claudenuiVizAllowedTool  = "mcp__nui-viz__*"
	claudeWriteAllowedTool    = "Write"
)

func sessionHasNamedMCP(configDir, name string) bool {
	if configDir == "" || strings.TrimSpace(name) == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(configDir, ".claude.json"))
	if err != nil {
		return false
	}
	var cfg struct {
		MCPServers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	_, ok := cfg.MCPServers[name]
	return ok
}

func sessionHasnuiHitlMCP(configDir string) bool {
	return sessionHasNamedMCP(configDir, nuiHitlMCPName)
}

func appendClaudeInteractiveHitlArgs(args []string, req RunRequest) []string {
	if req.HarnessPermissions != hitl.PermissionsInteractive {
		return args
	}
	if sessionHasnuiHitlMCP(req.ConfigDir) {
		args = append(args, "--allowedTools", claudenuiHitlAllowedTool)
	}
	if sessionHasnuiVizMCP(req.ConfigDir) {
		args = append(args, "--allowedTools", claudenuiVizAllowedTool)
		args = append(args, "--allowedTools", claudeWriteAllowedTool)
	}
	return args
}

func claudeAllowedTools(configDir string, servers []model.ADLMCPServer) []string {
	var allowed []string
	if hasnuiHitlMCP(servers) || sessionHasnuiHitlMCP(configDir) {
		allowed = append(allowed, claudenuiHitlAllowedTool)
	}
	if hasNuiVizMCP(servers) || sessionHasnuiVizMCP(configDir) {
		allowed = append(allowed, claudenuiVizAllowedTool, claudeWriteAllowedTool)
	}
	return allowed
}

func claudePreToolUseHooks(configDir string) ([]map[string]any, error) {
	var hooks []map[string]any
	if sessionHasnuiHitlMCP(configDir) {
		bridgePath := filepath.Join(configDir, claudeHitlBridgeScript)
		if err := os.WriteFile(bridgePath, []byte(claudeHitlBridgeSh), 0755); err != nil {
			return nil, err
		}
		hooks = append(hooks, map[string]any{
			"matcher": "AskUserQuestion|PermissionRequest",
			"hooks": []map[string]any{
				{
					"type":    "command",
					"command": bridgePath,
				},
			},
		})
	}
	if sessionHasnuiVizMCP(configDir) {
		bridgePath := filepath.Join(configDir, claudeVizBridgeScript)
		if err := os.WriteFile(bridgePath, []byte(claudeVizBridgeSh), 0755); err != nil {
			return nil, err
		}
		hooks = append(hooks, map[string]any{
			"matcher": "Skill|Bash",
			"hooks": []map[string]any{
				{
					"type":    "command",
					"command": bridgePath,
				},
			},
		})
	}
	return hooks, nil
}

func writeClaudeSessionSettings(configDir string, deps HarnessDeps) error {
	allowed := claudeAllowedTools(configDir, deps.MCPServers)
	allowed = hitl.ToolsForPermissionsAllow(deps.ToolApprovalPolicy, deps.ToolApprovalTools, allowed)
	preToolUse, err := claudePreToolUseHooks(configDir)
	if err != nil {
		return err
	}
	if len(allowed) == 0 && len(preToolUse) == 0 {
		return nil
	}
	settings := map[string]any{}
	if len(allowed) > 0 {
		settings["permissions"] = map[string]any{
			"allow": allowed,
		}
	}
	if len(preToolUse) > 0 {
		settings["hooks"] = map[string]any{
			"PreToolUse": preToolUse,
		}
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "settings.json"), data, 0644)
}

const claudeHitlBridgeSh = `#!/usr/bin/env bash
set -euo pipefail
export PAYLOAD="$(cat)"
export NUI_API_URL="${NUI_API_URL:-http://127.0.0.1:8080}"
export NUI_SESSION_ID="${NUI_SESSION_ID:-}"
export NUI_RUN_ID="${NUI_RUN_ID:-}"
python3 <<'PY'
import json, os, sys, urllib.error, urllib.request

def http_json(method, url, body=None):
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Content-Type": "application/json", "Accept": "application/json"}
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            text = resp.read().decode()
            return json.loads(text) if text.strip() else {}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode().strip()
        raise RuntimeError(f"HITL {method} failed ({exc.code}): {detail}") from exc

def deny(reason):
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": reason or "declined by user",
        }
    }))

def allow_ask_user(questions, answer_map):
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "allow",
            "updatedInput": {
                "questions": questions,
                "answers": answer_map,
            },
        }
    }))

def allow_simple():
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "allow",
        }
    }))

def map_answers(questions, hitl_answers):
    answer_map = {}
    if isinstance(hitl_answers.get("answers"), list):
        for i, q in enumerate(questions):
            if not isinstance(q, dict):
                continue
            qtext = (q.get("question") or q.get("header") or f"Question {i + 1}").strip()
            if i < len(hitl_answers["answers"]):
                answer_map[qtext] = hitl_answers["answers"][i]
    elif hitl_answers.get("answer") is not None:
        if len(questions) == 1 and isinstance(questions[0], dict):
            q = questions[0]
            qtext = (q.get("question") or q.get("header") or "Question").strip()
            answer_map[qtext] = hitl_answers["answer"]
        else:
            answer_map["answer"] = hitl_answers["answer"]
    return answer_map

def main():
    raw = os.environ.get("PAYLOAD", "")
    if not raw.strip():
        deny("empty hook payload")
        return
    payload = json.loads(raw)
    api = os.environ.get("NUI_API_URL", "http://127.0.0.1:8080").rstrip("/")
    tool = payload.get("tool_name") or payload.get("toolName") or ""
    tool_input = payload.get("tool_input") or payload.get("toolInput") or payload.get("input") or {}

    if tool == "AskUserQuestion":
        kind = "question"
        hitl_payload = {
            "title": tool_input.get("title") or "Answer required",
            "message": tool_input.get("message") or "",
            "questions": tool_input.get("questions") or [],
        }
    elif tool == "PermissionRequest":
        kind = "approval"
        hitl_payload = {
            "title": tool_input.get("title") or "Permission required",
            "message": tool_input.get("message") or "",
            "toolName": tool_input.get("toolName") or tool_input.get("tool_name") or "",
            "toolInput": tool_input.get("toolInput") or tool_input.get("tool_input") or {},
        }
    else:
        kind = "question"
        hitl_payload = dict(payload)

    body = {"kind": kind, "payload": hitl_payload}
    session_id = os.environ.get("NUI_SESSION_ID", "").strip()
    run_id = os.environ.get("NUI_RUN_ID", "").strip()
    if session_id:
        body["sessionId"] = session_id
    if run_id:
        body["runId"] = run_id

    try:
        created = http_json("POST", f"{api}/api/hitl/requests", body)
        request_id = created.get("requestId")
        if not request_id:
            deny("HITL request failed: missing requestId")
            return
        resp = http_json("GET", f"{api}/api/hitl/requests/{request_id}/wait")
    except Exception as exc:
        deny(str(exc))
        return

    status = resp.get("status", "")
    answers = resp.get("answers") or {}
    if status in ("declined", "cancelled"):
        deny(str(answers.get("message") or "declined by user"))
        return

    if tool == "AskUserQuestion":
        questions = tool_input.get("questions") or []
        allow_ask_user(questions, map_answers(questions, answers))
        return

    if kind == "approval":
        approved = answers.get("approved", True)
        if answers.get("action") == "reject":
            approved = False
        if approved is False:
            deny(str(answers.get("message") or "declined by user"))
            return

    allow_simple()

if __name__ == "__main__":
    main()
PY
`

const claudeVizBridgeSh = `#!/usr/bin/env bash
set -euo pipefail
export PAYLOAD="$(cat)"
python3 <<'PY'
import json, os, sys

def deny(reason):
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": reason,
        }
    }))

def is_dataviz_skill(tool_input):
    skill = tool_input.get("skill") or tool_input.get("name") or ""
    return "dataviz" in str(skill).lower()

def is_dataviz_bash(tool_input):
    cmd = tool_input.get("command") or ""
    lower = str(cmd).lower()
    return "dataviz" in lower and "bundled-skills" in lower

def main():
    raw = os.environ.get("PAYLOAD", "")
    if not raw.strip():
        return
    payload = json.loads(raw)
    tool = payload.get("tool_name") or payload.get("toolName") or ""
    tool_input = payload.get("tool_input") or payload.get("toolInput") or payload.get("input") or {}

    if tool == "Skill" and is_dataviz_skill(tool_input):
        deny("Use show_visualization on the nui-viz MCP server instead of the dataviz skill. Build self-contained HTML and call show_visualization in the same turn before any closing text.")
        return

    if tool == "Bash" and is_dataviz_bash(tool_input):
        deny("Use show_visualization on the nui-viz MCP server instead of dataviz scripts. Build self-contained HTML and call show_visualization in the same turn.")
        return

if __name__ == "__main__":
    main()
PY
`
