// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"

	"nui/internal/hitl"
)

// OrchestrationGate executes ADL type: hitl workflow steps.
type OrchestrationGate interface {
	CreateOrchestrationGate(ctx context.Context, in hitl.CreateInput) (*hitl.Request, error)
	Wait(ctx context.Context, requestID string) (*hitl.Response, error)
}

var orchestrationGate OrchestrationGate

func SetOrchestrationGate(g OrchestrationGate) {
	orchestrationGate = g
}

func orchestrationGateFn() OrchestrationGate {
	return orchestrationGate
}
