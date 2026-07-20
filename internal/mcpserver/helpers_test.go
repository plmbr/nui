// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestParseArgs(t *testing.T) {
	raw := map[string]any{"message": "hello", "wait": true}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Arguments: data},
	}
	args := parseArgs(req)
	if stringArg(args, "message") != "hello" {
		t.Fatalf("message = %q", stringArg(args, "message"))
	}
	if args["wait"] != true {
		t.Fatalf("wait = %v", args["wait"])
	}
}

func TestParseArgs_nilRequest(t *testing.T) {
	args := parseArgs(nil)
	if len(args) != 0 {
		t.Fatalf("args = %+v", args)
	}
}

func TestDefaultWorkingDir_explicit(t *testing.T) {
	if defaultWorkingDir("/tmp/foo") != "/tmp/foo" {
		t.Fatal("expected explicit dir")
	}
}

func evalPath(p string) string {
	clean := filepath.Clean(p)
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return clean
	}
	return resolved
}

func TestEmptyObjectSchema(t *testing.T) {
	schema := emptyObjectSchema()
	if schema["type"] != "object" {
		t.Fatalf("schema = %+v", schema)
	}
}

func TestToolJSON(t *testing.T) {
	res, err := toolJSON(map[string]string{"ok": "yes"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("content len = %d", len(res.Content))
	}
}

func TestRegisterAgentTools_doesNotPanic(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: nuiAgentMCPName, Version: "test"}, nil)
	registerAgentTools(server)
}

func TestRegisterHITLTools_doesNotPanic(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "nui-hitl", Version: "test"}, nil)
	registerHITLTools(server, nil)
}

func TestDefaultWorkingDir_fromCwd(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if evalPath(defaultWorkingDir("")) != evalPath(cwd) {
		t.Fatalf("defaultWorkingDir = %q cwd = %q", defaultWorkingDir(""), cwd)
	}
}
