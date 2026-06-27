// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"testing"
	"time"
)

const mentionLikeHost = `
import json, sys
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    req = json.loads(line)
    rid = req.get("id")
    method = req.get("method", "")
    if method == "mention.info":
        print(json.dumps({"jsonrpc": "2.0", "id": rid, "result": {"id": "test"}}), flush=True)
    elif method == "mention.shutdown":
        print(json.dumps({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}}), flush=True)
        sys.exit(0)
    elif rid is not None:
        print(json.dumps({
            "jsonrpc": "2.0",
            "id": rid,
            "error": {"code": -32601, "message": "Method not found: " + method},
        }), flush=True)
`

func TestStdioRPCCloseDoesNotDeadlock(t *testing.T) {
	rpc, err := StartStdioRPC([]string{"python3", "-c", mentionLikeHost}, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var info struct {
		ID string `json:"id"`
	}
	if err := rpc.Call("mention.info", map[string]any{}, &info); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		_ = rpc.Call("mention.shutdown", map[string]any{}, nil)
		_ = rpc.killProcess()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close path deadlocked")
	}
}
