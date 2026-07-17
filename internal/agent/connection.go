// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"nui/internal/store"
)

// SanitizeConnectionID turns an agent or project id into a safe connection file basename.
func SanitizeConnectionID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "harness"
	}
	replacer := strings.NewReplacer(":", "-", "/", "-", "\\", "-")
	return replacer.Replace(id)
}

// WaitForConnectionInfo polls until a harness writes its connection file or timeout.
func WaitForConnectionInfo(connectionID string, timeout time.Duration) (ConnectionInfo, error) {
	path, err := store.ConnectionFilePath(connectionID)
	if err != nil {
		return ConnectionInfo{}, err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			var conn ConnectionInfo
			if json.Unmarshal(data, &conn) == nil && conn.Port > 0 {
				return conn, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return ConnectionInfo{}, fmt.Errorf("timed out waiting for connection file %q", connectionID)
}

// ConnectionHostPort reads host/port from a connection file (used by HTTP harness startup).
func ConnectionHostPort(connectionID string, timeout time.Duration) (host string, port int, err error) {
	conn, err := WaitForConnectionInfo(connectionID, timeout)
	if err != nil {
		return "", 0, err
	}
	host = conn.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return host, conn.Port, nil
}
