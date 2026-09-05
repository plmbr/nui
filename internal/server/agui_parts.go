// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"

	"github.com/google/uuid"
	"nui/internal/agent"
	"nui/internal/model"
	"nui/internal/viz"
)

type assistantPartAccumulator struct {
	parts           []model.ChatMessagePart
	images          []model.ChatImage
	pendingTools    map[string]string
	subagents       map[string]*assistantPartAccumulator
	content         string
	errored         bool
	councilProgress *model.CouncilProgressState
}

func newAssistantPartAccumulator() *assistantPartAccumulator {
	return &assistantPartAccumulator{
		pendingTools: map[string]string{},
		subagents:    map[string]*assistantPartAccumulator{},
	}
}

func (a *assistantPartAccumulator) appendText(delta string) {
	if delta == "" {
		return
	}
	a.content += delta
	if len(a.parts) > 0 && a.parts[len(a.parts)-1].Type == "text" {
		a.parts[len(a.parts)-1].Content += delta
		return
	}
	a.parts = append(a.parts, model.ChatMessagePart{
		Type:    "text",
		ID:      uuid.NewString(),
		Content: delta,
	})
}

func (a *assistantPartAccumulator) applyEvent(ev agent.Event, mcpLookup func(string) (uri, server string, ok bool)) {
	if ev.ParentToolCallID != "" {
		part := a.toolPartForCall(ev.ParentToolCallID)
		if part == nil {
			return
		}
		sub := a.subagents[ev.ParentToolCallID]
		if sub == nil {
			sub = newAssistantPartAccumulator()
			a.subagents[ev.ParentToolCallID] = sub
		}
		sub.applyEvent(agent.Event{
			Type:           ev.Type,
			Content:        ev.Content,
			ToolCallID:     ev.ToolCallID,
			ToolName:       ev.ToolName,
			ToolArgs:       ev.ToolArgs,
			ImageData:      ev.ImageData,
			ImageMediaType: ev.ImageMediaType,
		}, mcpLookup)
		part.SubagentTrace = sub.parts
		return
	}
	switch ev.Type {
	case agent.EventText:
		a.appendText(ev.Content)
	case agent.EventToolCallStart:
		partID := uuid.NewString()
		if ev.ToolCallID != "" {
			a.pendingTools[ev.ToolCallID] = partID
		}
		a.parts = append(a.parts, model.ChatMessagePart{
			Type:       "tool",
			ID:         partID,
			ToolCallID: ev.ToolCallID,
			ToolName:   ev.ToolName,
			ToolArgs:   map[string]any{},
		})
	case agent.EventToolCallArgs:
		if ev.ToolCallID == "" || ev.ToolArgs == "" {
			return
		}
		partID, ok := a.pendingTools[ev.ToolCallID]
		if !ok {
			return
		}
		var args map[string]any
		if err := json.Unmarshal([]byte(ev.ToolArgs), &args); err != nil {
			return
		}
		for i := range a.parts {
			if a.parts[i].Type != "tool" || a.parts[i].ID != partID {
				continue
			}
			a.parts[i].ToolArgs = args
			toolName := ev.ToolName
			if toolName == "" {
				toolName = a.parts[i].ToolName
			}
			if html, title, ok := viz.ParseFromTool(toolName, args); ok {
				html = viz.PrepareHTML(html)
				if viz.VisualizationHTMLReady(html) {
					a.parts[i].VisualizationHTML = html
					a.parts[i].VisualizationTitle = title
				}
			}
			break
		}
	case agent.EventToolCallEnd:
		if ev.ToolCallID == "" {
			return
		}
		partID, ok := a.pendingTools[ev.ToolCallID]
		if !ok {
			return
		}
		var toolInput map[string]any
		if ev.ToolArgs != "" {
			_ = json.Unmarshal([]byte(ev.ToolArgs), &toolInput)
		}
		for i := range a.parts {
			if a.parts[i].Type != "tool" || a.parts[i].ID != partID {
				continue
			}
			if toolInput == nil && len(a.parts[i].ToolArgs) > 0 {
				toolInput = a.parts[i].ToolArgs
			}
			toolName := ev.ToolName
			if toolName == "" {
				toolName = a.parts[i].ToolName
			}
			if html, title, ok := viz.ParseFromTool(toolName, toolInput); ok {
				html = viz.PrepareHTML(html)
				if viz.VisualizationHTMLReady(html) {
					a.parts[i].VisualizationHTML = html
					a.parts[i].VisualizationTitle = title
				}
			}
			if mcpLookup == nil {
				break
			}
			uri, server, found := mcpLookup(ev.ToolName)
			if !found {
				break
			}
			if toolInput == nil {
				toolInput = map[string]any{}
			}
			a.parts[i].MCPAppResourceURI = uri
			a.parts[i].MCPAppServerName = server
			a.parts[i].MCPAppToolInput = toolInput
			break
		}
	case agent.EventToolCallResult:
		if ev.ToolCallID == "" {
			return
		}
		partID, ok := a.pendingTools[ev.ToolCallID]
		if !ok {
			return
		}
		var result any = ev.Content
		var parsed any
		if err := json.Unmarshal([]byte(ev.Content), &parsed); err == nil {
			result = parsed
		}
		for i := range a.parts {
			if a.parts[i].Type == "tool" && a.parts[i].ID == partID {
				a.parts[i].ToolResult = result
				break
			}
		}
	case agent.EventImage:
		if ev.ImageData == "" {
			return
		}
		mediaType := ev.ImageMediaType
		if mediaType == "" {
			mediaType = "image/png"
		}
		a.images = append(a.images, model.ChatImage{
			ID:        uuid.NewString(),
			MediaType: mediaType,
			Data:      ev.ImageData,
		})
	case agent.EventCouncilProgress:
		if ev.Council != nil {
			a.applyCouncilProgress(ev.Council)
		}
	case agent.EventError:
		a.errored = true
	}
}

func (a *assistantPartAccumulator) applyCouncilProgress(ev *agent.CouncilProgress) {
	prev := a.councilProgress
	if prev == nil {
		prev = &model.CouncilProgressState{Members: []model.CouncilMemberProgress{}}
	}
	next := &model.CouncilProgressState{
		Phase:         prev.Phase,
		Round:         prev.Round,
		RoundIndex:    prev.RoundIndex,
		RoundsTotal:   prev.RoundsTotal,
		MembersTotal:  prev.MembersTotal,
		MembersDone:   prev.MembersDone,
		Quorum:        prev.Quorum,
		EstimatedCost: prev.EstimatedCost,
		Members:       append([]model.CouncilMemberProgress{}, prev.Members...),
	}
	if ev.Phase != "" {
		next.Phase = ev.Phase
	}
	if ev.Round != "" {
		next.Round = ev.Round
	}
	if ev.RoundIndex != 0 {
		next.RoundIndex = ev.RoundIndex
	}
	if ev.RoundsTotal != 0 {
		next.RoundsTotal = ev.RoundsTotal
	}
	if ev.MembersTotal != 0 {
		next.MembersTotal = ev.MembersTotal
	}
	if ev.MembersDone != 0 {
		next.MembersDone = ev.MembersDone
	}
	if ev.Quorum != 0 {
		next.Quorum = ev.Quorum
	}
	if ev.EstimatedCost != "" {
		next.EstimatedCost = ev.EstimatedCost
	}
	if ev.MemberID != "" {
		status := "queued"
		idx := -1
		for i := range next.Members {
			if next.Members[i].ID == ev.MemberID {
				idx = i
				status = next.Members[i].Status
				break
			}
		}
		switch ev.Phase {
		case "member_failed":
			status = "failed"
		case "member_completed":
			status = "completed"
		case "member_started":
			status = "running"
		}
		entry := model.CouncilMemberProgress{
			ID:     ev.MemberID,
			Label:  ev.MemberLabel,
			Status: status,
		}
		if entry.Label == "" {
			entry.Label = ev.MemberID
		}
		if idx >= 0 {
			prevMember := next.Members[idx]
			if entry.Label == ev.MemberID && prevMember.Label != "" {
				entry.Label = prevMember.Label
			}
			if ev.MemberLabel != "" {
				entry.Label = ev.MemberLabel
			}
			entry.SessionID = prevMember.SessionID
			entry.RunID = prevMember.RunID
			entry.ElapsedMS = prevMember.ElapsedMS
			if status == "queued" {
				entry.Status = prevMember.Status
			}
		}
		if ev.MemberSessionID != "" {
			entry.SessionID = ev.MemberSessionID
		}
		if ev.RunID != "" {
			entry.RunID = ev.RunID
		}
		if ev.ElapsedMS != 0 {
			entry.ElapsedMS = ev.ElapsedMS
		}
		entry.Error = ev.Error
		if idx >= 0 {
			next.Members[idx] = entry
		} else {
			next.Members = append(next.Members, entry)
		}
	}
	a.councilProgress = next
}

func (a *assistantPartAccumulator) toolPartForCall(toolCallID string) *model.ChatMessagePart {
	partID, ok := a.pendingTools[toolCallID]
	if !ok {
		return nil
	}
	for i := range a.parts {
		if a.parts[i].Type == "tool" && a.parts[i].ID == partID {
			return &a.parts[i]
		}
	}
	return nil
}

func (a *assistantPartAccumulator) toMessage(messageID string) model.ChatMessage {
	msg := model.ChatMessage{
		ID:        messageID,
		Role:      "assistant",
		Content:   a.content,
		CreatedAt: "",
		Error:     a.errored,
	}
	if len(a.parts) > 0 {
		msg.Parts = viz.NormalizeParts(a.parts)
	}
	if len(a.images) > 0 {
		msg.Images = a.images
	}
	if a.councilProgress != nil {
		msg.CouncilProgress = a.councilProgress
	}
	return msg
}
