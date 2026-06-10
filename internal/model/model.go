// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

type Project struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	WorkingDir string `json:"workingDir"`
	AgentType  string `json:"agentType"`
	CreatedAt  string `json:"createdAt"`
}

type ChatMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}
