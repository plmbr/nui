// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"

	"nui/internal/agent"
)

func TestAssistantPartAccumulator_councilProgress(t *testing.T) {
	acc := newAssistantPartAccumulator()
	acc.applyEvent(agent.Event{
		Type: agent.EventCouncilProgress,
		Council: &agent.CouncilProgress{
			Phase:        "round_started",
			Round:        "position",
			RoundIndex:   1,
			RoundsTotal:  2,
			MembersTotal: 2,
			MembersDone:  0,
		},
	}, nil)
	acc.applyEvent(agent.Event{
		Type: agent.EventCouncilProgress,
		Council: &agent.CouncilProgress{
			Phase:           "member_started",
			Round:           "position",
			MemberID:        "analyst",
			MemberLabel:     "Analyst",
			MemberSessionID: "sess-a",
			RunID:           "run-a",
			MembersTotal:    2,
			MembersDone:     0,
		},
	}, nil)
	acc.applyEvent(agent.Event{
		Type: agent.EventCouncilProgress,
		Council: &agent.CouncilProgress{
			Phase:           "member_completed",
			MemberID:        "analyst",
			MemberSessionID: "sess-a",
			RunID:           "run-a",
			MembersDone:     1,
			ElapsedMS:       1200,
		},
	}, nil)

	msg := acc.toMessage("msg-1")
	if msg.CouncilProgress == nil {
		t.Fatal("expected councilProgress on message")
	}
	cp := msg.CouncilProgress
	if cp.Phase != "member_completed" {
		t.Fatalf("phase: got %q", cp.Phase)
	}
	if len(cp.Members) != 1 {
		t.Fatalf("members: got %d", len(cp.Members))
	}
	m := cp.Members[0]
	if m.ID != "analyst" || m.Status != "completed" || m.SessionID != "sess-a" || m.RunID != "run-a" {
		t.Fatalf("member: %+v", m)
	}
	if m.ElapsedMS != 1200 {
		t.Fatalf("elapsed: got %d", m.ElapsedMS)
	}
}
