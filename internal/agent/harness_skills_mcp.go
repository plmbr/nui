// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"path/filepath"
	"strings"

	"nui/internal/model"
	"nui/internal/store"
)

const (
	nuiSkillsMCPName = "nui-skills"
	envNuiSkillsRoot = "NUI_SKILLS_ROOT"
	envNuiWorkingDir = "NUI_WORKING_DIR"
)

func nuiSkillsMCPServer(sessionID, workingDir string) (model.ADLMCPServer, error) {
	exe, err := nuiExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	skillsRoot := ""
	if sessionID != "" {
		if configDir, dirErr := store.SessionConfigDir(sessionID); dirErr == nil {
			skillsRoot = filepath.Join(configDir, "skills")
		}
	}
	env := map[string]string{}
	if skillsRoot != "" {
		env[envNuiSkillsRoot] = skillsRoot
	}
	if wd := strings.TrimSpace(workingDir); wd != "" {
		env[envNuiWorkingDir] = wd
	}
	return model.ADLMCPServer{
		Name:    nuiSkillsMCPName,
		Command: exe,
		Args:    []string{"skills-mcp"},
		Env:     env,
	}, nil
}

func hasNuiSkillsMCP(servers []model.ADLMCPServer) bool {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == nuiSkillsMCPName {
			return true
		}
	}
	return false
}

func appendNuiSkillsMCP(servers []model.ADLMCPServer, sessionID, workingDir string) ([]model.ADLMCPServer, error) {
	if hasNuiSkillsMCP(servers) {
		return servers, nil
	}
	srv, err := nuiSkillsMCPServer(sessionID, workingDir)
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}

const apiToolsSystemPromptAppendix = `
## nui API tools (skills, filesystem, bash)

- **Default: no tools.** Greetings and ordinary chat get a plain-text reply with zero tool calls — do not run bash/fs/skills to explore the workspace first.
- Skill catalog in this prompt lists **name + description** only. When a skill is relevant to the user's request, call **load_skill** on **nui-skills** to load full instructions and the absolute **skillRoot**.
- Shell commands (**pwd**, **ls**, **cat**, scripts, git, etc.): only when the user asks — call **bash** on **nui-bash** with a **command** string. Do not use **nui-fs** **read** for shell commands or directories.
- Files: **nui-fs** **read** / **glob** / **write** / **edit**. **read** is for file contents only (not directories, not shell).
- Prefer these tools over inventing scripts when work is needed. Report tool results honestly — never invent sandbox or permission messages.
- Mutating filesystem ops and bash require human approval in the nui UI.
`

func appendAPIToolsSystemPrompt(systemPrompt string) string {
	block := strings.TrimSpace(apiToolsSystemPromptAppendix)
	if block == "" {
		return systemPrompt
	}
	base := strings.TrimSpace(systemPrompt)
	if base == "" {
		return block
	}
	return base + "\n\n" + block
}

func appendAPIWorkspaceMCPs(servers []model.ADLMCPServer, sessionID, workingDir string) ([]model.ADLMCPServer, error) {
	var err error
	servers, err = appendNuiSkillsMCP(servers, sessionID, workingDir)
	if err != nil {
		return servers, err
	}
	servers, err = appendNuiFSMCP(servers, workingDir)
	if err != nil {
		return servers, err
	}
	return appendNuiBashMCP(servers, workingDir)
}
