// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"nui/internal/agents"
	"nui/internal/model"
	"nui/internal/nuiclient"
	"nui/internal/uiaction"
)

// RunOrchestrator starts the nui-orchestrator MCP server on stdio.
func RunOrchestrator(ctx context.Context, baseURL string) error {
	client := nuiclient.New(baseURL)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "nui-orchestrator",
		Version: "1.0.0",
	}, nil)

	registerOrchestratorTools(server, client)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerOrchestratorTools(server *mcp.Server, client *nuiclient.Client) {
	server.AddTool(&mcp.Tool{
		Name:        "search_agents",
		Description: "Rank launchable agents for a task using semantic/TF-IDF search. Prefer this over list_agents when choosing which agent to launch — returns a small top-k list with scores.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "User intent or task to match against agent names, descriptions, and tags",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Max results to return (default 8, max 20)",
				},
			},
			"required": []string{"query"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		query := stringArg(args, "query")
		if query == "" {
			return toolError(fmt.Errorf("query is required"))
		}
		limit := intArg(args, "limit")
		out, err := client.SearchOrchestratorAgents(ctx, query, limit)
		if err != nil {
			return toolError(err)
		}
		return toolJSON(out)
	})

	server.AddTool(&mcp.Tool{
		Name:        "list_agents",
		Description: "List nui agent types (full inventory). Prefer search_agents when choosing an agent for a user task. Set routable_only=true to list only launchable agents.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"routable_only": map[string]any{
					"type":        "boolean",
					"description": "If true, return only agents suitable as launch_session targets",
				},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		if boolArg(args, "routable_only") {
			out, err := client.ListOrchestratorAgents(ctx)
			if err != nil {
				return toolError(err)
			}
			return toolJSON(out)
		}
		out, err := client.ListAgents(ctx)
		if err != nil {
			return toolError(err)
		}
		return toolJSON(out)
	})

	server.AddTool(&mcp.Tool{
		Name:        "launch_session",
		Description: "Create a new nui session with the chosen agent. Prompt is optional: omit it to open an idle chat (user-mode) or use defaultPrompt (auto-mode). Never pass meta instructions like \"launch X\" as the prompt — only the task text, if any.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_type":  map[string]any{"type": "string", "description": "Agent id from list_agents"},
				"prompt":      map[string]any{"type": "string", "description": "Optional task text to run as the first message"},
				"working_dir": map[string]any{"type": "string", "description": "Working directory (optional)"},
				"name":        map[string]any{"type": "string", "description": "Optional session name"},
			},
			"required": []string{"agent_type"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		agentType := stringArg(args, "agent_type")
		prompt := stringArg(args, "prompt")
		if agentType == "" {
			return toolError(fmt.Errorf("agent_type is required"))
		}
		if !agents.IsOrchestratorRoutingTarget(agentType) {
			return toolError(fmt.Errorf("agent %q cannot be used as an orchestrator routing target", agentType))
		}
		available, err := client.ListAgents(ctx)
		if err != nil {
			return toolError(err)
		}
		var found *nuiclient.AgentType
		for i := range available {
			a := available[i]
			if a.ID != agentType {
				continue
			}
			if !a.Available {
				return toolError(fmt.Errorf("agent %q is not available for launcher sessions", agentType))
			}
			found = &a
			break
		}
		if found == nil {
			return toolError(fmt.Errorf("unknown agent id %q", agentType))
		}
		if strings.TrimSpace(prompt) == "" && found.PromptMode == model.ADLPromptModeAuto {
			prompt = model.ResolveADLLaunchPrompt(model.ADLDefinition{
				PromptMode:    found.PromptMode,
				DefaultPrompt: found.DefaultPrompt,
			}, "")
		}
		workingDir := defaultWorkingDir(stringArg(args, "working_dir"))
		if workingDir == "" {
			return toolError(fmt.Errorf("working directory required"))
		}
		sess, err := client.CreateSession(ctx, nuiclient.CreateSessionRequest{
			Name:       stringArg(args, "name"),
			AgentType:  agentType,
			WorkingDir: workingDir,
		})
		if err != nil {
			return toolError(err)
		}
		return toolJSON(map[string]any{
			"session": sess,
			"prompt":  prompt,
		})
	})

	server.AddTool(&mcp.Tool{
		Name:        "control_ui",
		Description: "Drive the nui SPA: navigate to panels or change theme. Prefer this over telling the user to click UI controls.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"actions": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type":   map[string]any{"type": "string", "description": "navigate | set_theme | refresh_ui"},
							"target": map[string]any{"type": "string", "description": "For navigate: customize | new_session | launch | schedules"},
							"theme":  map[string]any{"type": "string", "description": "For set_theme: dark | light"},
						},
						"required": []string{"type"},
					},
					"description": "UI actions to apply in order",
				},
			},
			"required": []string{"actions"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		raw, ok := args["actions"]
		if !ok {
			return toolError(fmt.Errorf("actions is required"))
		}
		data, err := json.Marshal(raw)
		if err != nil {
			return toolError(fmt.Errorf("invalid actions"))
		}
		var actions []uiaction.Action
		if err := json.Unmarshal(data, &actions); err != nil {
			return toolError(fmt.Errorf("invalid actions: %w", err))
		}
		if len(actions) == 0 {
			return toolError(fmt.Errorf("actions must be non-empty"))
		}
		for i, a := range actions {
			if msg := uiaction.Validate(a); msg != "" {
				return toolError(fmt.Errorf("actions[%d]: %s", i, msg))
			}
		}
		return toolJSON(map[string]any{"ok": true, "actions": actions})
	})

	server.AddTool(&mcp.Tool{
		Name:        "describe_environment",
		Description: "Summarize this nui install: version, defaultHarness, theme, and inventory counts.",
		InputSchema: emptyObjectSchema(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		settings, err := client.GetSettings(ctx)
		if err != nil {
			return toolError(err)
		}
		version, _ := client.GetVersion(ctx)
		agentsList, _ := client.ListAgents(ctx)
		exts, _ := client.ListExtensions(ctx)
		mcpServers, _ := client.ListMCPServers(ctx)
		disabled := 0
		for _, e := range exts {
			if e.Disabled {
				disabled++
			}
		}
		harnesses := agents.SelectableHarnessRefs()
		return toolJSON(map[string]any{
			"version":            version,
			"defaultHarness":     settings.DefaultHarness,
			"theme":              settings.Theme,
			"agentCount":         len(agentsList),
			"extensionCount":     len(exts),
			"disabledExtensions": disabled,
			"mcpServerCount":     len(mcpServers),
			"harnessCount":       len(harnesses),
		})
	})

	server.AddTool(&mcp.Tool{
		Name:        "list_extensions",
		Description: "List installed nui extensions and whether each is disabled.",
		InputSchema: emptyObjectSchema(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		out, err := client.ListExtensions(ctx)
		if err != nil {
			return toolError(err)
		}
		return toolJSON(out)
	})

	server.AddTool(&mcp.Tool{
		Name:        "list_mcp_servers",
		Description: "List configured MCP servers (user + extension-contributed).",
		InputSchema: emptyObjectSchema(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		out, err := client.ListMCPServers(ctx)
		if err != nil {
			return toolError(err)
		}
		return toolJSON(out)
	})

	server.AddTool(&mcp.Tool{
		Name:        "list_harnesses",
		Description: "List available harness runtimes on this system, including the current defaultHarness.",
		InputSchema: emptyObjectSchema(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		settings, _ := client.GetSettings(ctx)
		var list []map[string]any
		for _, h := range agents.SelectableHarnessRefs() {
			list = append(list, map[string]any{
				"ref":   h.Ref,
				"label": h.Label,
				"group": h.Group,
			})
		}
		exts, _ := client.ListExtensions(ctx)
		seen := map[string]bool{}
		for _, item := range list {
			if ref, _ := item["ref"].(string); ref != "" {
				seen[ref] = true
			}
		}
		for _, e := range exts {
			if e.Disabled {
				continue
			}
			for _, id := range e.Harnesses {
				if seen[id] {
					continue
				}
				seen[id] = true
				list = append(list, map[string]any{
					"ref":   id,
					"label": id,
					"group": "extension:" + e.Name,
				})
			}
		}
		return toolJSON(map[string]any{
			"defaultHarness": settings.DefaultHarness,
			"harnesses":      list,
		})
	})

	server.AddTool(&mcp.Tool{
		Name:        "set_extension_enabled",
		Description: "Enable or disable an installed extension by name.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":    map[string]any{"type": "string", "description": "Extension name"},
				"enabled": map[string]any{"type": "boolean", "description": "true to enable, false to disable"},
			},
			"required": []string{"name", "enabled"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		name := stringArg(args, "name")
		if name == "" {
			return toolError(fmt.Errorf("name is required"))
		}
		enabled, ok := boolArgOK(args, "enabled")
		if !ok {
			return toolError(fmt.Errorf("enabled is required"))
		}
		exts, err := client.ListExtensions(ctx)
		if err != nil {
			return toolError(err)
		}
		found := false
		for _, e := range exts {
			if e.Name == name {
				found = true
				break
			}
		}
		if !found {
			return toolError(fmt.Errorf("extension %q not found", name))
		}
		settings, err := client.GetSettings(ctx)
		if err != nil {
			return toolError(err)
		}
		disabled := make([]string, 0, len(settings.DisabledExtensions))
		for _, d := range settings.DisabledExtensions {
			if d == name {
				continue
			}
			disabled = append(disabled, d)
		}
		if !enabled {
			disabled = append(disabled, name)
		}
		updated, err := client.SetDisabledExtensions(ctx, disabled)
		if err != nil {
			return toolError(err)
		}
		_ = client.ReloadExtensions(ctx)
		return toolJSON(map[string]any{
			"ok":                 true,
			"name":               name,
			"enabled":            enabled,
			"disabledExtensions": updated.DisabledExtensions,
			"actions": []uiaction.Action{
				{Type: uiaction.TypeRefreshUI},
			},
		})
	})
}

func boolArg(args map[string]any, key string) bool {
	v, ok := boolArgOK(args, key)
	return ok && v
}

func boolArgOK(args map[string]any, key string) (bool, bool) {
	if args == nil {
		return false, false
	}
	v, ok := args[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func intArg(args map[string]any, key string) int {
	if args == nil {
		return 0
	}
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}
