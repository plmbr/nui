// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"loop/internal/extensions"
	"loop/internal/model"
)

// BuildDeployAssets expands ADL aiAssets into a serializable deploy bundle.
func BuildDeployAssets(def model.ADLDefinition, reg *extensions.Registry) (extensions.DeployAssets, error) {
	deps, err := buildHarnessDeps("", def, nil, "", reg, nil)
	if err != nil {
		return extensions.DeployAssets{}, err
	}
	rules := make([]extensions.DeployRuleAsset, 0, len(deps.ResolvedRules))
	for _, rule := range deps.ResolvedRules {
		rules = append(rules, extensions.DeployRuleAsset{
			Name:    rule.Name,
			Content: rule.Content,
		})
	}
	return extensions.DeployAssets{
		Skills:     deps.Skills,
		MCPServers: deps.MCPServers,
		Rules:      rules,
	}, nil
}
