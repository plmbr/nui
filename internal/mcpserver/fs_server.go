// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	nuiFSMCPName     = "nui-fs"
	envNuiWorkingDir = "NUI_WORKING_DIR"
)

// RunFS starts the nui-fs MCP server on stdio.
func RunFS(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    nuiFSMCPName,
		Version: "1.0.0",
	}, nil)

	registerFSTools(server)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerFSTools(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        "read",
		Description: "Read a file from the host filesystem. Paths may be absolute, ~/..., or relative to NUI_WORKING_DIR.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path to read",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "Optional 1-based start line (inclusive)",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Optional max number of lines to return",
				},
			},
			"required": []string{"path"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		path := stringArg(args, "path")
		if path == "" {
			return toolError(fmt.Errorf("path is required"))
		}
		abs, err := resolveHostPath(path)
		if err != nil {
			return toolError(err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return toolError(err)
		}
		if info.IsDir() {
			return toolError(fmt.Errorf("%s is a directory; use nui-bash bash (e.g. pwd, ls) or nui-fs glob — read is for files only", abs))
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return toolError(err)
		}
		content := string(data)
		offset := intArg(args, "offset")
		limit := intArg(args, "limit")
		if offset > 0 || limit > 0 {
			content = sliceLines(content, offset, limit)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: content}},
			StructuredContent: map[string]any{
				"path": abs,
			},
		}, nil
	})

	server.AddTool(&mcp.Tool{
		Name:        "write",
		Description: "Write contents to a file on the host filesystem (creates parent directories). May require human approval.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path to write",
				},
				"contents": map[string]any{
					"type":        "string",
					"description": "Full file contents",
				},
			},
			"required": []string{"path", "contents"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		path := stringArg(args, "path")
		if path == "" {
			return toolError(fmt.Errorf("path is required"))
		}
		contents, ok := args["contents"].(string)
		if !ok {
			if args["contents"] == nil {
				return toolError(fmt.Errorf("contents is required"))
			}
			contents = fmt.Sprint(args["contents"])
		}
		abs, err := resolveHostPath(path)
		if err != nil {
			return toolError(err)
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			return toolError(err)
		}
		if err := os.WriteFile(abs, []byte(contents), 0644); err != nil {
			return toolError(err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Wrote %s", abs)}},
			StructuredContent: map[string]any{
				"path": abs,
			},
		}, nil
	})

	server.AddTool(&mcp.Tool{
		Name:        "edit",
		Description: "Replace the first occurrence of old_string with new_string in a file. May require human approval.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path to edit",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "Exact text to find",
				},
				"new_string": map[string]any{
					"type":        "string",
					"description": "Replacement text",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		path := stringArg(args, "path")
		oldStr, _ := args["old_string"].(string)
		newStr, _ := args["new_string"].(string)
		if path == "" {
			return toolError(fmt.Errorf("path is required"))
		}
		if oldStr == "" {
			return toolError(fmt.Errorf("old_string is required"))
		}
		abs, err := resolveHostPath(path)
		if err != nil {
			return toolError(err)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return toolError(err)
		}
		content := string(data)
		if !strings.Contains(content, oldStr) {
			return toolError(fmt.Errorf("old_string not found in %s", abs))
		}
		updated := strings.Replace(content, oldStr, newStr, 1)
		if err := os.WriteFile(abs, []byte(updated), 0644); err != nil {
			return toolError(err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Edited %s", abs)}},
			StructuredContent: map[string]any{
				"path": abs,
			},
		}, nil
	})

	server.AddTool(&mcp.Tool{
		Name:        "glob",
		Description: "Find files matching a glob pattern under a root directory (default NUI_WORKING_DIR).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "Glob pattern (e.g. **/*.go)",
				},
				"root": map[string]any{
					"type":        "string",
					"description": "Optional root directory (default NUI_WORKING_DIR)",
				},
			},
			"required": []string{"pattern"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		pattern := stringArg(args, "pattern")
		if pattern == "" {
			return toolError(fmt.Errorf("pattern is required"))
		}
		root := stringArg(args, "root")
		if root == "" {
			root = workingDir()
		}
		if root == "" {
			return toolError(fmt.Errorf("root or NUI_WORKING_DIR is required"))
		}
		absRoot, err := resolveHostPath(root)
		if err != nil {
			return toolError(err)
		}
		matches, err := filepath.Glob(filepath.Join(absRoot, pattern))
		if err != nil {
			// Try recursive walk for ** patterns
			matches, err = globWalk(absRoot, pattern)
			if err != nil {
				return toolError(err)
			}
		}
		const maxMatches = 500
		if len(matches) > maxMatches {
			matches = matches[:maxMatches]
		}
		return toolJSON(map[string]any{
			"root":    absRoot,
			"pattern": pattern,
			"matches": matches,
		})
	})
}

func workingDir() string {
	return strings.TrimSpace(os.Getenv(envNuiWorkingDir))
}

func resolveHostPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("path is required")
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			return filepath.Clean(home), nil
		}
		return filepath.Clean(filepath.Join(home, p[2:])), nil
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	base := workingDir()
	if base == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		base = cwd
	}
	return filepath.Clean(filepath.Join(base, p)), nil
}

func sliceLines(content string, offset, limit int) string {
	lines := strings.Split(content, "\n")
	start := 0
	if offset > 0 {
		start = offset - 1
		if start > len(lines) {
			return ""
		}
	}
	end := len(lines)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return strings.Join(lines[start:end], "\n")
}

func globWalk(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		ok, matchErr := filepath.Match(pattern, rel)
		if matchErr != nil {
			return matchErr
		}
		if !ok {
			ok, _ = filepath.Match(pattern, filepath.Base(path))
		}
		if ok {
			matches = append(matches, path)
		}
		return nil
	})
	return matches, err
}
