// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nui/internal/llm"
)

const (
	subAgentToolName = "run_sub_agent"

	subAgentsPhaseDelegating     = "delegating"
	subAgentsPhaseMemberStarted  = councilPhaseMemberStarted
	subAgentsPhaseMemberCompleted = councilPhaseMemberCompleted
	subAgentsPhaseMemberFailed   = councilPhaseMemberFailed
	subAgentsPhaseComplete       = councilPhaseComplete
)

type chairAction struct {
	Action string `json:"action"` // delegate | finish
	Agent  string `json:"agent,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Answer string `json:"answer,omitempty"`
}

func (a *ADLAgent) runSubAgents(ctx context.Context, req RunRequest, events chan<- Event) error {
	resolve := req.ResolveADL
	if resolve == nil {
		return fmt.Errorf("subAgents requires ResolveADL")
	}
	cfg := a.def.Orchestration
	if cfg == nil || len(cfg.Members) == 0 {
		return fmt.Errorf("subAgents has no members")
	}
	members, err := a.resolveCouncilMembers(resolve)
	if err != nil {
		return err
	}
	byID := map[string]resolvedCouncilMember{}
	for _, m := range members {
		byID[m.id] = m
	}

	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 20
	}
	timeout := parseCouncilTimeout(cfg.MemberTimeout)
	sessionMode := strings.TrimSpace(cfg.SessionMode)
	if sessionMode == "" {
		sessionMode = "persistent"
	}

	memberSessions := map[string]string{}
	var completedSteps int

	events <- Event{
		Type: EventCouncilProgress,
		Council: &CouncilProgress{
			Phase:        subAgentsPhaseDelegating,
			Round:        "subAgents",
			MembersTotal: len(members),
		},
	}

	ensureMemberSession := func(member resolvedCouncilMember) (string, error) {
		if sid := memberSessions[member.id]; sid != "" {
			return sid, nil
		}
		if req.EnsureCouncilMemberSession == nil {
			return "", nil
		}
		sid, err := req.EnsureCouncilMemberSession(member.id, member.label, member.id)
		if err != nil {
			return "", fmt.Errorf("subAgents: ensure member session %q: %w", member.id, err)
		}
		memberSessions[member.id] = sid
		return sid, nil
	}

	runMember := func(ctx context.Context, memberID, prompt string) (string, error) {
		member, ok := byID[memberID]
		if !ok {
			return "", fmt.Errorf("unknown member %q; allowed: %s", memberID, memberIDList(members))
		}
		childSessionID, err := ensureMemberSession(member)
		if err != nil {
			return "", err
		}
		events <- Event{
			Type: EventCouncilProgress,
			Council: &CouncilProgress{
				Phase:           subAgentsPhaseMemberStarted,
				Round:           "subAgents",
				MemberID:        member.id,
				MemberLabel:     member.label,
				MemberSessionID: childSessionID,
				MembersTotal:    len(members),
				MembersDone:     completedSteps,
			},
		}
		start := time.Now()
		var runID string
		out, rid, err := a.runCouncilMember(ctx, req, member, prompt, childSessionID, sessionMode, timeout, func(id string) {
			runID = id
			events <- Event{
				Type: EventCouncilProgress,
				Council: &CouncilProgress{
					Phase:           subAgentsPhaseMemberStarted,
					Round:           "subAgents",
					MemberID:        member.id,
					MemberLabel:     member.label,
					MemberSessionID: childSessionID,
					RunID:           id,
					MembersTotal:    len(members),
					MembersDone:     completedSteps,
				},
			}
		})
		if rid != "" {
			runID = rid
		}
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			events <- Event{
				Type: EventCouncilProgress,
				Council: &CouncilProgress{
					Phase:           subAgentsPhaseMemberFailed,
					Round:           "subAgents",
					MemberID:        member.id,
					MemberLabel:     member.label,
					MemberSessionID: childSessionID,
					RunID:           runID,
					MembersTotal:    len(members),
					MembersDone:     completedSteps,
					ElapsedMS:       elapsed,
					Error:           err.Error(),
				},
			}
			return "", err
		}
		completedSteps++
		events <- Event{
			Type: EventCouncilProgress,
			Council: &CouncilProgress{
				Phase:           subAgentsPhaseMemberCompleted,
				Round:           "subAgents",
				MemberID:        member.id,
				MemberLabel:     member.label,
				MemberSessionID: childSessionID,
				RunID:           runID,
				MembersTotal:    len(members),
				MembersDone:     completedSteps,
				ElapsedMS:       elapsed,
			},
		}
		return out, nil
	}

	chairPrompt := appendSystemPromptBlock(req.SystemPrompt, subAgentsChairInstructions(members))
	chairReq := req
	chairReq.SystemPrompt = chairPrompt

	if strings.TrimSpace(a.def.Harness.Type) == "api" {
		chairReq.ExtraTools = []llm.Tool{runSubAgentTool(members)}
		var turns int
		chairReq.HandleExtraTool = func(ctx context.Context, name string, args map[string]any) (string, bool, error) {
			if name != subAgentToolName {
				return "", false, nil
			}
			turns++
			if turns > maxTurns {
				return "", true, fmt.Errorf("subAgents: exceeded maxTurns (%d)", maxTurns)
			}
			agentID, _ := args["agent"].(string)
			prompt, _ := args["prompt"].(string)
			agentID = strings.TrimSpace(agentID)
			prompt = strings.TrimSpace(prompt)
			if agentID == "" || prompt == "" {
				return "", true, fmt.Errorf("run_sub_agent requires agent and prompt")
			}
			out, err := runMember(ctx, agentID, prompt)
			if err != nil {
				return fmt.Sprintf("error: %v", err), true, nil
			}
			return out, true, nil
		}
		if err := a.runStep(ctx, chairReq, a.def.Harness, nil, events); err != nil {
			return err
		}
		events <- Event{
			Type: EventCouncilProgress,
			Council: &CouncilProgress{
				Phase:        subAgentsPhaseComplete,
				Round:        "subAgents",
				MembersTotal: len(members),
			},
		}
		return nil
	}

	return a.runSubAgentsPromptLoop(ctx, chairReq, members, maxTurns, runMember, events)
}

func (a *ADLAgent) runSubAgentsPromptLoop(
	ctx context.Context,
	req RunRequest,
	members []resolvedCouncilMember,
	maxTurns int,
	runMember func(ctx context.Context, memberID, prompt string) (string, error),
	events chan<- Event,
) error {
	history := strings.Builder{}
	history.WriteString("User goal:\n")
	history.WriteString(req.Message)
	history.WriteString("\n")

	for turn := 1; turn <= maxTurns; turn++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		msg := history.String() + "\n" + subAgentsTurnInstruction(turn, maxTurns)
		turnReq := req
		turnReq.Message = msg
		// Mute protocol JSON text — only emit a clean finish answer to the UI.
		collecting := &collectingEvents{upstream: events, muteText: true}
		ch := collecting.start()
		err := a.runStep(ctx, turnReq, a.def.Harness, nil, ch)
		collecting.finish()
		if err != nil {
			return err
		}
		action, ok := parseChairAction(collecting.text)
		if !ok {
			// Treat free-form final answer as finish when no structured action.
			if t := strings.TrimSpace(collecting.text); t != "" {
				events <- Event{Type: EventText, Content: t}
			}
			events <- Event{
				Type: EventCouncilProgress,
				Council: &CouncilProgress{
					Phase:        subAgentsPhaseComplete,
					Round:        "subAgents",
					MembersTotal: len(members),
				},
			}
			return nil
		}
		switch strings.ToLower(strings.TrimSpace(action.Action)) {
		case "finish":
			if ans := strings.TrimSpace(action.Answer); ans != "" {
				events <- Event{Type: EventText, Content: ans}
			}
			events <- Event{
				Type: EventCouncilProgress,
				Council: &CouncilProgress{
					Phase:        subAgentsPhaseComplete,
					Round:        "subAgents",
					MembersTotal: len(members),
				},
			}
			return nil
		case "delegate":
			agentID := strings.TrimSpace(action.Agent)
			prompt := strings.TrimSpace(action.Prompt)
			if agentID == "" || prompt == "" {
				return fmt.Errorf("subAgents: delegate action requires agent and prompt")
			}
			out, err := runMember(ctx, agentID, prompt)
			if err != nil {
				history.WriteString(fmt.Sprintf("\nDelegation to %s failed: %v\n", agentID, err))
				continue
			}
			history.WriteString(fmt.Sprintf("\n--- Result from %s ---\n%s\n", agentID, out))
		default:
			return fmt.Errorf("subAgents: unknown action %q", action.Action)
		}
	}
	return fmt.Errorf("subAgents: exceeded maxTurns (%d)", maxTurns)
}

func memberIDList(members []resolvedCouncilMember) string {
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.id)
	}
	return strings.Join(ids, ", ")
}

func runSubAgentTool(members []resolvedCouncilMember) llm.Tool {
	ids := make([]string, 0, len(members))
	desc := strings.Builder{}
	desc.WriteString("Delegate a task to a sub-agent. Available agents:\n")
	for _, m := range members {
		ids = append(ids, m.id)
		desc.WriteString(fmt.Sprintf("- %s (%s)\n", m.id, m.label))
	}
	return llm.Tool{
		Type: "function",
		Function: llm.Function{
			Name:        subAgentToolName,
			Description: desc.String(),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agent": map[string]any{
						"type":        "string",
						"description": "Registry agent id to run",
						"enum":        ids,
					},
					"prompt": map[string]any{
						"type":        "string",
						"description": "Task prompt for the sub-agent",
					},
				},
				"required": []string{"agent", "prompt"},
			},
		},
	}
}

func subAgentsChairInstructions(members []resolvedCouncilMember) string {
	var b strings.Builder
	b.WriteString("You are an adaptive orchestrator. For each user goal:\n")
	b.WriteString("1. Pick exactly ONE sub-agent that is best suited to start (or continue) the work.\n")
	b.WriteString("2. Delegate a focused task to that agent only — never fan out to all agents at once.\n")
	b.WriteString("3. Read the result, then either finish with the final answer OR delegate again to the same or a different sub-agent.\n")
	b.WriteString("4. Repeat until the user goal is fully answered.\n\n")
	b.WriteString("Available sub-agents:\n")
	for _, m := range members {
		b.WriteString(fmt.Sprintf("- %s (%s)\n", m.id, m.label))
	}
	b.WriteString("\nUse the run_sub_agent tool to delegate one agent at a time. When the goal is complete, respond with the final answer and do not call more tools.\n")
	b.WriteString("If tools are unavailable, emit a single JSON object as your entire reply (one action only):\n")
	b.WriteString(`{"action":"delegate","agent":"<id>","prompt":"<task>"}` + "\n")
	b.WriteString(`{"action":"finish","answer":"<final answer>"}` + "\n")
	return b.String()
}

func subAgentsTurnInstruction(turn, maxTurns int) string {
	return fmt.Sprintf(
		"Turn %d/%d. Choose the next step for ONE agent only. Reply with ONLY a JSON action object: "+
			`{"action":"delegate","agent":"<id>","prompt":"<task>"} or {"action":"finish","answer":"<final answer>"}. `+
			"Do not emit multiple actions. You may reuse the previous agent if that is best.",
		turn, maxTurns,
	)
}

func parseChairAction(text string) (chairAction, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return chairAction{}, false
	}
	// Walk concatenated / embedded JSON values (models sometimes emit
	// {"action":"delegate",...}{"action":"finish",...} in one reply).
	if action, ok := decodeChairActions(trimmed); ok {
		return action, true
	}
	if start := strings.Index(trimmed, "{"); start >= 0 {
		if end := strings.LastIndex(trimmed, "}"); end > start {
			if action, ok := decodeChairActions(trimmed[start : end+1]); ok {
				return action, true
			}
		}
	}
	return chairAction{}, false
}

// decodeChairActions parses one or more JSON objects and returns the first
// valid chair action (one action per turn is expected).
func decodeChairActions(s string) (chairAction, bool) {
	dec := json.NewDecoder(strings.NewReader(s))
	for {
		var action chairAction
		if err := dec.Decode(&action); err != nil {
			break
		}
		switch strings.ToLower(strings.TrimSpace(action.Action)) {
		case "delegate", "finish":
			return action, true
		}
	}
	return chairAction{}, false
}
