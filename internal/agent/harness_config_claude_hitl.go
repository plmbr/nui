// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"os"
	"path/filepath"

	"loop/internal/hitl"
)

const (
	claudeHitlBridgeScript    = "hitl-bridge.sh"
	claudeLoopHitlAllowedTool = "mcp__loop-hitl__*"
)

func sessionHasLoopHitlMCP(configDir string) bool {
	if configDir == "" {
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
	_, ok := cfg.MCPServers[loopHitlMCPName]
	return ok
}

func appendClaudeInteractiveHitlArgs(args []string, req RunRequest) []string {
	if req.HarnessPermissions != hitl.PermissionsInteractive {
		return args
	}
	if !sessionHasLoopHitlMCP(req.ConfigDir) {
		return args
	}
	return append(args, "--allowedTools", claudeLoopHitlAllowedTool)
}

func writeClaudeHITLHooks(configDir string) error {
	bridgePath := filepath.Join(configDir, claudeHitlBridgeScript)
	if err := os.WriteFile(bridgePath, []byte(claudeHitlBridgeSh), 0755); err != nil {
		return err
	}
	settingsPath := filepath.Join(configDir, "settings.json")
	settings := map[string]any{
		"permissions": map[string]any{
			"allow": []string{claudeLoopHitlAllowedTool},
		},
		"hooks": map[string]any{
			"PreToolUse": []map[string]any{
				{
					"matcher": "AskUserQuestion|PermissionRequest",
					"hooks": []map[string]any{
						{
							"type":    "command",
							"command": bridgePath,
						},
					},
				},
			},
		},
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0644)
}

const claudeHitlBridgeSh = `#!/usr/bin/env bash
set -euo pipefail
export PAYLOAD="$(cat)"
export LOOP_API_URL="${LOOP_API_URL:-http://127.0.0.1:8080}"
export LOOP_SESSION_ID="${LOOP_SESSION_ID:-}"
export LOOP_RUN_ID="${LOOP_RUN_ID:-}"
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
    api = os.environ.get("LOOP_API_URL", "http://127.0.0.1:8080").rstrip("/")
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
    session_id = os.environ.get("LOOP_SESSION_ID", "").strip()
    run_id = os.environ.get("LOOP_RUN_ID", "").strip()
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
