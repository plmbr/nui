// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"loop/internal/model"
	"loop/internal/skills"
)

func installHarnessSkills(harnessType, configDir, workingDir string, skillList []model.ADLSkill) error {
	skillList = skills.WithBuiltins(skillList)
	if len(skillList) == 0 {
		return nil
	}
	return skills.MaterializeSkills(skills.Context{WorkingDir: workingDir}, harnessType, configDir, skillList)
}
