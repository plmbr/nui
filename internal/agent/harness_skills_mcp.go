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

- Skill catalog in this prompt lists **name + description** only. When a skill is relevant, call **load_skill** on **nui-skills** for full instructions and the absolute **skillRoot**.
- Host shell: **bash** on **nui-bash** (default cwd **NUI_WORKING_DIR**).
- Host files: **nui-fs** **read** / **glob** / **write** / **edit** (paths absolute, ~/…, or relative to **NUI_WORKING_DIR**). **read** is for files, not directories.
- Report tool results honestly. Mutating filesystem ops and bash may require human approval in the nui UI.
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
