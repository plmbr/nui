// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type antigravityStreamParser struct {
	emittedText     bool
	needsTextSep    bool
	seenToolStarts  map[string]bool
	seenToolEnds    map[string]bool
	seenToolResults map[string]bool
	conversationID  string
	turnError       string
	turnDone        bool
}

func newAntigravityStreamParser() *antigravityStreamParser {
	return &antigravityStreamParser{
		seenToolStarts:  map[string]bool{},
		seenToolEnds:    map[string]bool{},
		seenToolResults: map[string]bool{},
	}
}

func (p *antigravityStreamParser) markTextSepNeeded() {
	if p.emittedText {
		p.needsTextSep = true
	}
}

func (p *antigravityStreamParser) emitText(text string, events chan<- Event) {
	if text == "" {
		return
	}
	if p.emittedText && p.needsTextSep && !strings.HasPrefix(text, "\n") {
		text = "\n\n" + text
	}
	p.needsTextSep = false
	p.emittedText = true
	events <- Event{Type: EventText, Content: text}
}

func (p *antigravityStreamParser) handleLine(line []byte, events chan<- Event) {
	var obj struct {
		Event          string          `json:"event"`
		ConversationID string          `json:"conversation_id"`
		Init           json.RawMessage `json:"init"`
		StepUpdate     json.RawMessage `json:"step_update"`
		Result         json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(line, &obj); err != nil {
		return
	}

	switch obj.Event {
	case "init":
		if obj.ConversationID != "" {
			p.conversationID = obj.ConversationID
		}
		var init struct {
			ConversationID string `json:"conversation_id"`
		}
		if json.Unmarshal(obj.Init, &init) == nil && init.ConversationID != "" {
			p.conversationID = init.ConversationID
		}
	case "step_update":
		p.handleStepUpdate(obj.StepUpdate, events)
	case "result":
		p.handleResult(obj.Result, obj.ConversationID, events)
	}
}

func (p *antigravityStreamParser) handleStepUpdate(raw json.RawMessage, events chan<- Event) {
	var step struct {
		ConversationID string          `json:"conversation_id"`
		StepIndex      int             `json:"step_index"`
		State          string          `json:"state"`
		StepType       string          `json:"step_type"`
		ToolName       string          `json:"tool_name"`
		TextDelta      string          `json:"text_delta"`
		ToolInfo       json.RawMessage `json:"tool_info"`
	}
	if err := json.Unmarshal(raw, &step); err != nil {
		return
	}
	if step.ConversationID != "" {
		p.conversationID = step.ConversationID
	}

	switch step.StepType {
	case "agent_response":
		if step.TextDelta != "" {
			p.emitText(step.TextDelta, events)
		}
	case "tool":
		p.handleToolStep(step.StepIndex, step.ToolName, step.State, step.ToolInfo, events)
	}
}

func (p *antigravityStreamParser) handleToolStep(stepIndex int, toolName, state string, toolInfo json.RawMessage, events chan<- Event) {
	toolID := fmt.Sprintf("agy-%d", stepIndex)
	name := toolName
	var info struct {
		Name       string          `json:"name"`
		Parameters json.RawMessage `json:"parameters"`
		Output     any             `json:"output"`
		Error      any             `json:"error"`
	}
	if len(toolInfo) > 0 && json.Unmarshal(toolInfo, &info) == nil {
		if name == "" {
			name = info.Name
		}
	}

	p.markTextSepNeeded()

	if !p.seenToolStarts[toolID] {
		p.seenToolStarts[toolID] = true
		events <- Event{
			Type:       EventToolCallStart,
			ToolCallID: toolID,
			ToolName:   name,
		}
	}

	if len(info.Parameters) > 0 && string(info.Parameters) != "null" && !p.seenToolEnds[toolID] {
		events <- Event{
			Type:       EventToolCallArgs,
			ToolCallID: toolID,
			ToolArgs:   string(info.Parameters),
		}
	}

	if state != "DONE" {
		return
	}

	if !p.seenToolEnds[toolID] {
		p.seenToolEnds[toolID] = true
		endArgs := ""
		if len(info.Parameters) > 0 && string(info.Parameters) != "null" {
			endArgs = string(info.Parameters)
		}
		events <- Event{
			Type:       EventToolCallEnd,
			ToolCallID: toolID,
			ToolName:   name,
			ToolArgs:   endArgs,
		}
	}

	if p.seenToolResults[toolID] {
		return
	}
	p.seenToolResults[toolID] = true
	result := ""
	if info.Error != nil {
		b, _ := json.Marshal(info.Error)
		result = string(b)
	} else if info.Output != nil {
		switch v := info.Output.(type) {
		case string:
			result = v
		default:
			b, _ := json.Marshal(v)
			result = string(b)
		}
	}
	if result != "" {
		events <- Event{
			Type:       EventToolCallResult,
			ToolCallID: toolID,
			Content:    result,
		}
	}
}

func (p *antigravityStreamParser) handleResult(raw json.RawMessage, topConversationID string, events chan<- Event) {
	var result struct {
		ConversationID string `json:"conversation_id"`
		Status         string `json:"status"`
		Response       string `json:"response"`
		Error          string `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return
	}
	if result.ConversationID != "" {
		p.conversationID = result.ConversationID
	} else if topConversationID != "" {
		p.conversationID = topConversationID
	}

	// Prefer streamed text; fall back to the terminal response if nothing was streamed.
	if !p.emittedText && result.Response != "" {
		p.emitText(result.Response, events)
	}

	status := strings.ToUpper(strings.TrimSpace(result.Status))
	if status != "" && status != "SUCCESS" {
		errMsg := strings.TrimSpace(result.Error)
		if errMsg == "" {
			errMsg = "antigravity turn failed: " + result.Status
		}
		p.turnError = errMsg
	}
	p.turnDone = true
}
