// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"loop/internal/model"
)

const codexSystemPromptFile = "AGENTS.md"
const codexConfigFile = "config.toml"

type codexHarnessProvisioner struct{}

func (codexHarnessProvisioner) provision(configDir string, deps HarnessDeps) error {
	if err := writeCodexSystemPrompt(configDir, deps.SystemPrompt); err != nil {
		return err
	}
	if err := writeCodexConfig(configDir, deps); err != nil {
		return err
	}
	if err := installHarnessSkills("codex", configDir, deps.WorkingDir, deps.Skills); err != nil {
		return fmt.Errorf("install skills: %w", err)
	}
	return writeHarnessManifest(configDir, "codex", deps, map[string]any{
		"systemPromptFile": codexSystemPromptFile,
		"configFile":       codexConfigFile,
		"configEnv":        envCodexHome,
	})
}

func writeCodexSystemPrompt(configDir, systemPrompt string) error {
	path := filepath.Join(configDir, codexSystemPromptFile)
	if strings.TrimSpace(systemPrompt) == "" {
		_ = os.Remove(path)
		return nil
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(systemPrompt)+"\n"), 0644)
}

func writeCodexConfig(configDir string, deps HarnessDeps) error {
	cfgPath := filepath.Join(configDir, codexConfigFile)
	var sections []string

	if sp := strings.TrimSpace(deps.SystemPrompt); sp != "" {
		sections = append(sections, fmt.Sprintf("developer_instructions = \"\"\"\n%s\n\"\"\"",
			escapeTOMLMultiline(sp)))
	}

	for _, srv := range deps.MCPServers {
		name := strings.TrimSpace(srv.Name)
		if name == "" {
			continue
		}
		block, err := codexMCPServerTOML(name, srv)
		if err != nil {
			return fmt.Errorf("mcp server %q: %w", name, err)
		}
		sections = append(sections, block)
	}

	if len(sections) == 0 {
		_ = os.Remove(cfgPath)
		return nil
	}

	content := strings.Join(sections, "\n\n") + "\n"
	return os.WriteFile(cfgPath, []byte(content), 0644)
}

func codexMCPServerTOML(name string, srv model.ADLMCPServer) (string, error) {
	var lines []string
	lines = append(lines, fmt.Sprintf("[mcp_servers.%s]", name))

	if cmd := strings.TrimSpace(srv.Command); cmd != "" {
		lines = append(lines, fmt.Sprintf("command = %q", cmd))
		if len(srv.Args) > 0 {
			args := make([]string, len(srv.Args))
			for i, a := range srv.Args {
				args[i] = fmt.Sprintf("%q", a)
			}
			lines = append(lines, "args = ["+strings.Join(args, ", ")+"]")
		}
		if len(srv.Env) > 0 {
			lines = append(lines, "env = "+codexTOMLInlineTable(srv.Env))
		}
		return strings.Join(lines, "\n"), nil
	}

	url := strings.TrimSpace(srv.URL)
	if url == "" {
		return "", fmt.Errorf("requires url or command")
	}
	lines = append(lines, fmt.Sprintf("url = %q", url))
	if len(srv.Headers) > 0 {
		lines = append(lines, "http_headers = "+codexTOMLInlineTable(srv.Headers))
	}
	return strings.Join(lines, "\n"), nil
}
