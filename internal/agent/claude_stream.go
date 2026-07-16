// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"
)

type claudeStreamParser struct {
	emittedText     bool
	needsTextSep    bool
	seenToolStarts  map[string]bool
	seenToolEnds    map[string]bool
	seenToolResults map[string]bool
	blocks          map[int]*claudeBlockState
	sessionID       string
	lastStopReason  string
	emittedDone     bool
	turnComplete    bool
}

type claudeBlockState struct {
	kind     string
	toolID   string
	toolName string
	args     strings.Builder
}

func newClaudeStreamParser() *claudeStreamParser {
	return &claudeStreamParser{
		seenToolStarts:  map[string]bool{},
		seenToolEnds:    map[string]bool{},
		seenToolResults: map[string]bool{},
		blocks:          map[int]*claudeBlockState{},
	}
}

func (p *claudeStreamParser) emitDone(sessionID, parentToolUseID string, events chan<- Event) {
	if parentToolUseID != "" {
		return
	}
	if p.emittedDone || p.hasPendingToolWork() {
		return
	}
	p.emittedDone = true
	p.turnComplete = true
	sid := sessionID
	if sid == "" {
		sid = p.sessionID
	}
	events <- Event{Type: EventDone, SessionID: sid}
}

func (p *claudeStreamParser) hasPendingToolWork() bool {
	for id := range p.seenToolStarts {
		if !p.seenToolResults[id] {
			return true
		}
	}
	return false
}

func (p *claudeStreamParser) completedTurn() bool {
	return p.turnComplete
}

func (p *claudeStreamParser) markTextSepNeeded() {
	if p.emittedText {
		p.needsTextSep = true
	}
}

func (p *claudeStreamParser) emit(parentToolUseID string, ev Event, events chan<- Event) {
	if parentToolUseID != "" {
		ev.ParentToolCallID = parentToolUseID
	}
	events <- ev
}

func (p *claudeStreamParser) emitText(text, parentToolUseID string, events chan<- Event) {
	if text == "" {
		return
	}
	if p.emittedText && p.needsTextSep && !strings.HasPrefix(text, "\n") {
		text = "\n\n" + text
	}
	p.needsTextSep = false
	p.emittedText = true
	p.emit(parentToolUseID, Event{Type: EventText, Content: text}, events)
}

func (p *claudeStreamParser) handleLine(line []byte, events chan<- Event) {
	if len(line) == 0 {
		return
	}

	var envelope struct {
		Type             string          `json:"type"`
		Event            json.RawMessage `json:"event"`
		SessionID        string          `json:"session_id"`
		IsError          bool            `json:"is_error"`
		ErrMsg           string          `json:"error"`
		Result           string          `json:"result"`
		ParentToolUseID  string          `json:"parent_tool_use_id"`
		ToolUseResult    json.RawMessage `json:"tool_use_result"`
		Message          json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return
	}

	if envelope.SessionID != "" {
		p.sessionID = envelope.SessionID
	}

	parentToolUseID := envelope.ParentToolUseID

	switch envelope.Type {
	case "stream_event":
		p.handleStreamEvent(envelope.Event, parentToolUseID, events)
	case "assistant":
		p.handleAssistant(envelope.Message, parentToolUseID, events)
	case "user":
		p.handleUser(parentToolUseID, envelope.ToolUseResult, envelope.Message, events)
	case "result":
		if envelope.IsError {
			msg := envelope.ErrMsg
			if msg == "" {
				msg = envelope.Result
			}
			p.emit(parentToolUseID, Event{Type: EventError, Error: msg}, events)
		} else {
			p.emitDone(envelope.SessionID, parentToolUseID, events)
		}
	}
}

func (p *claudeStreamParser) handleStreamEvent(raw json.RawMessage, parentToolUseID string, events chan<- Event) {
	var ev struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
			StopReason  string `json:"stop_reason"`
		} `json:"delta"`
		ContentBlock struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"content_block"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		return
	}

	switch ev.Type {
	case "message_delta":
		if ev.Delta.StopReason != "" {
			p.lastStopReason = ev.Delta.StopReason
		}
	case "message_stop":
		if ev.Delta.StopReason != "" {
			p.lastStopReason = ev.Delta.StopReason
		}
		if p.lastStopReason == "end_turn" {
			p.emitDone("", parentToolUseID, events)
		}
		p.lastStopReason = ""
	case "content_block_delta":
		if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
			p.emitText(ev.Delta.Text, parentToolUseID, events)
		}
	case "content_block_start":
		block := ev.ContentBlock
		if block.Type == "text" {
			p.markTextSepNeeded()
			return
		}
		if block.Type != "tool_use" || block.ID == "" {
			return
		}
		p.markTextSepNeeded()
		p.blocks[ev.Index] = &claudeBlockState{
			kind:     "tool_use",
			toolID:   block.ID,
			toolName: block.Name,
		}
		if parentToolUseID == "" {
			if p.seenToolStarts[block.ID] {
				return
			}
			p.seenToolStarts[block.ID] = true
		}
		p.emit(parentToolUseID, Event{
			Type:       EventToolCallStart,
			ToolCallID: block.ID,
			ToolName:   block.Name,
		}, events)
	case "content_block_stop":
		state, ok := p.blocks[ev.Index]
		if !ok || state.kind != "tool_use" {
			delete(p.blocks, ev.Index)
			return
		}
		args := state.args.String()
		if args != "" {
			p.emit(parentToolUseID, Event{
				Type:       EventToolCallArgs,
				ToolCallID: state.toolID,
				ToolArgs:   args,
			}, events)
		}
		if parentToolUseID == "" {
			if !p.seenToolEnds[state.toolID] {
				p.seenToolEnds[state.toolID] = true
				p.emit(parentToolUseID, Event{
					Type:       EventToolCallEnd,
					ToolCallID: state.toolID,
					ToolName:   state.toolName,
					ToolArgs:   args,
				}, events)
			}
		} else {
			scopeKey := parentToolUseID + "::" + state.toolID
			if !p.seenToolEnds[scopeKey] {
				p.seenToolEnds[scopeKey] = true
				p.emit(parentToolUseID, Event{
					Type:       EventToolCallEnd,
					ToolCallID: state.toolID,
					ToolName:   state.toolName,
					ToolArgs:   args,
				}, events)
			}
		}
		p.markTextSepNeeded()
		delete(p.blocks, ev.Index)
	}

	if ev.Type == "content_block_delta" && ev.Delta.Type == "input_json_delta" && ev.Delta.PartialJSON != "" {
		if state, ok := p.blocks[ev.Index]; ok && state.kind == "tool_use" {
			state.args.WriteString(ev.Delta.PartialJSON)
		}
	}
}

func (p *claudeStreamParser) handleAssistant(raw json.RawMessage, parentToolUseID string, events chan<- Event) {
	var msg struct {
		Content []json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	for _, blockRaw := range msg.Content {
		var blockType struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(blockRaw, &blockType); err != nil {
			continue
		}

		switch blockType.Type {
		case "text":
			var block struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(blockRaw, &block) != nil || block.Text == "" {
				continue
			}
			// Partial stream_event deltas are authoritative with --include-partial-messages.
			// Full assistant snapshots after tool/HITL pauses can replay prior turns.
			if parentToolUseID == "" && (len(p.seenToolStarts) > 0 || p.emittedText) {
				continue
			}
			p.emitText(block.Text, parentToolUseID, events)
		case "tool_use":
			p.markTextSepNeeded()
			var block struct {
				ID    string         `json:"id"`
				Name  string         `json:"name"`
				Input map[string]any `json:"input"`
			}
			if err := json.Unmarshal(blockRaw, &block); err != nil || block.ID == "" {
				continue
			}
			argsJSON, _ := json.Marshal(block.Input)
			argsStr := string(argsJSON)
			if parentToolUseID == "" {
				if !p.seenToolStarts[block.ID] {
					p.seenToolStarts[block.ID] = true
					p.emit(parentToolUseID, Event{
						Type:       EventToolCallStart,
						ToolCallID: block.ID,
						ToolName:   block.Name,
					}, events)
					p.emit(parentToolUseID, Event{
						Type:       EventToolCallArgs,
						ToolCallID: block.ID,
						ToolArgs:   argsStr,
					}, events)
				}
				if !p.seenToolEnds[block.ID] {
					p.seenToolEnds[block.ID] = true
					p.emit(parentToolUseID, Event{
						Type:       EventToolCallEnd,
						ToolCallID: block.ID,
						ToolName:   block.Name,
						ToolArgs:   argsStr,
					}, events)
				}
			} else {
				p.emit(parentToolUseID, Event{
					Type:       EventToolCallStart,
					ToolCallID: block.ID,
					ToolName:   block.Name,
				}, events)
				p.emit(parentToolUseID, Event{
					Type:       EventToolCallArgs,
					ToolCallID: block.ID,
					ToolArgs:   argsStr,
				}, events)
				p.emit(parentToolUseID, Event{
					Type:       EventToolCallEnd,
					ToolCallID: block.ID,
					ToolName:   block.Name,
					ToolArgs:   argsStr,
				}, events)
			}
		case "image":
			emitImageEvents(blockRaw, parentToolUseID, events)
		}
	}
}

func (p *claudeStreamParser) handleUser(parentToolUseID string, toolUseResult json.RawMessage, messageRaw json.RawMessage, events chan<- Event) {
	toolUseID := parentToolUseID
	if toolUseID == "" {
		toolUseID = toolUseIDFromMessage(messageRaw)
	}
	if toolUseID != "" && len(toolUseResult) > 0 && toolUseResult[0] != 'n' {
		p.emitToolResult(toolUseID, "", toolUseResult, events)
		emitImageEvents(toolUseResult, "", events)
	}

	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(messageRaw, &msg); err != nil {
		return
	}
	var blocks []json.RawMessage
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return
	}
	for _, blockRaw := range blocks {
		var block struct {
			Type      string          `json:"type"`
			ToolUseID string          `json:"tool_use_id"`
			Content   json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(blockRaw, &block); err != nil {
			continue
		}
		if block.Type != "tool_result" || block.ToolUseID == "" {
			continue
		}
		p.emitToolResult(block.ToolUseID, parentToolUseID, block.Content, events)
		emitImageEvents(block.Content, parentToolUseID, events)
	}
}

func (p *claudeStreamParser) emitToolResult(toolUseID, parentToolUseID string, result json.RawMessage, events chan<- Event) {
	if parentToolUseID == "" {
		if p.seenToolResults[toolUseID] {
			return
		}
		p.seenToolResults[toolUseID] = true
	} else if p.seenToolResults[parentToolUseID+"::"+toolUseID] {
		return
	} else {
		p.seenToolResults[parentToolUseID+"::"+toolUseID] = true
	}
	p.emit(parentToolUseID, Event{
		Type:       EventToolCallResult,
		ToolCallID: toolUseID,
		Content:    string(result),
	}, events)
}

func toolUseIDFromMessage(messageRaw json.RawMessage) string {
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(messageRaw, &msg); err != nil {
		return ""
	}
	var blocks []json.RawMessage
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return ""
	}
	for _, blockRaw := range blocks {
		var block struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
		}
		if json.Unmarshal(blockRaw, &block) == nil && block.Type == "tool_result" && block.ToolUseID != "" {
			return block.ToolUseID
		}
	}
	return ""
}
