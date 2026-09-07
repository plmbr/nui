// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"nui/internal/skills"
)

const (
	nuiSkillsMCPName = "nui-skills"
	envNuiSkillsRoot = "NUI_SKILLS_ROOT"
)

// RunSkills starts the nui-skills MCP server on stdio.
func RunSkills(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    nuiSkillsMCPName,
		Version: "1.0.0",
	}, nil)

	registerSkillsTools(server)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerSkillsTools(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        "list_skills",
		Description: "List available nui skills with name, description, and absolute skill root path",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entries, err := listMaterializedSkills()
		if err != nil {
			return toolError(err)
		}
		return toolJSON(map[string]any{"skills": entries})
	})

	server.AddTool(&mcp.Tool{
		Name:        "load_skill",
		Description: "Load the full SKILL.md instructions for a skill. Returns body text and absolute skillRoot for use with nui-fs / nui-bash.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Skill directory name under NUI_SKILLS_ROOT",
				},
			},
			"required": []string{"name"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		name := stringArg(args, "name")
		if name == "" {
			// Models sometimes pass skillName instead of name.
			name = stringArg(args, "skillName")
		}
		if name == "" {
			return toolError(fmt.Errorf("name is required"))
		}
		root, err := skillRootPath(name)
		if err != nil {
			return toolError(err)
		}
		data, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
		if err != nil {
			return toolError(fmt.Errorf("skill %q: %w", name, err))
		}
		body := skills.StripFrontmatter(string(data))
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: body},
			},
			StructuredContent: map[string]any{
				"name":      name,
				"skillRoot": root,
				"body":      body,
			},
		}, nil
	})
}

type skillListEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SkillRoot   string `json:"skillRoot"`
}

func skillsRoot() string {
	return strings.TrimSpace(os.Getenv(envNuiSkillsRoot))
}

func listMaterializedSkills() ([]skillListEntry, error) {
	root := skillsRoot()
	if root == "" {
		return nil, fmt.Errorf("NUI_SKILLS_ROOT is not set")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []skillListEntry{}, nil
		}
		return nil, err
	}
	var out []skillListEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		dir := filepath.Join(root, name)
		data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			continue
		}
		abs, _ := filepath.Abs(dir)
		desc := skillDescriptionFromMarkdown(string(data))
		out = append(out, skillListEntry{
			Name:        name,
			Description: desc,
			SkillRoot:   abs,
		})
	}
	return out, nil
}

func skillRootPath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid skill name %q", name)
	}
	root := skillsRoot()
	if root == "" {
		return "", fmt.Errorf("NUI_SKILLS_ROOT is not set")
	}
	dir := filepath.Join(root, name)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if abs != rootAbs && !strings.HasPrefix(abs, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("skill path escapes NUI_SKILLS_ROOT")
	}
	if _, err := os.Stat(filepath.Join(abs, "SKILL.md")); err != nil {
		return "", fmt.Errorf("skill %q not found", name)
	}
	return abs, nil
}

func skillDescriptionFromMarkdown(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	rest := strings.TrimPrefix(content, "---")
	end := strings.Index(rest, "---")
	if end < 0 {
		return ""
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "description:") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		return strings.Trim(val, `"'`)
	}
	return ""
}
