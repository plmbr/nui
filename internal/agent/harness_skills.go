// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"nui/internal/model"
	"nui/internal/skills"
)

func installHarnessSkills(harnessType, configDir, workingDir string, skillList []model.ADLSkill) error {
	skillList = skills.WithBuiltins(skillList)
	if len(skillList) == 0 {
		return nil
	}
	return skills.MaterializeSkills(skills.Context{WorkingDir: workingDir}, harnessType, configDir, skillList)
}
