// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"strings"
)

type piStreamParser struct {
	emittedText     bool
	needsTextSep    bool
	seenToolStarts  map[string]bool
	seenToolEnds    map[string]bool
	seenToolResults map[string]bool
}

func newPiStreamParser() *piStreamParser {
	return &piStreamParser{
		seenToolStarts:  map[string]bool{},
		seenToolEnds:    map[string]bool{},
		seenToolResults: map[string]bool{},
	}
}

func (p *piStreamParser) markTextSepNeeded() {
	if p.emittedText {
		p.needsTextSep = true
	}
}

func (p *piStreamParser) emitText(text string, events chan<- Event) {
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

func (p *piStreamParser) handleLine(line []byte, events chan<- Event) {
	var obj struct {
		Type                 string          `json:"type"`
		ID                   string          `json:"id"`
		ToolCallID           string          `json:"toolCallId"`
		ToolName             string          `json:"toolName"`
		Args                 json.RawMessage `json:"args"`
		Result               json.RawMessage `json:"result"`
		ToolResults          json.RawMessage `json:"toolResults"`
		AssistantMessageEvent json.RawMessage `json:"assistantMessageEvent"`
	}
	if err := json.Unmarshal(line, &obj); err != nil {
		return
	}

	switch obj.Type {
	case "session":
		// session ID is captured via get_state after the turn
	case "message_update":
		p.handleMessageUpdate(obj.AssistantMessageEvent, events)
	case "tool_execution_start":
		p.markTextSepNeeded()
		if obj.ToolCallID != "" && !p.seenToolStarts[obj.ToolCallID] {
			p.seenToolStarts[obj.ToolCallID] = true
			events <- Event{
				Type:       EventToolCallStart,
				ToolCallID: obj.ToolCallID,
				ToolName:   obj.ToolName,
			}
		}
	case "tool_execution_end":
		p.emitTool(obj.ToolCallID, obj.ToolName, obj.Args, obj.Result, events)
	case "turn_end":
		var results []json.RawMessage
		if err := json.Unmarshal(obj.ToolResults, &results); err == nil {
			for _, raw := range results {
				var result struct {
					ToolCallID string          `json:"toolCallId"`
					ToolUseID  string          `json:"toolUseId"`
					ID         string          `json:"id"`
					ToolName   string          `json:"toolName"`
					Name       string          `json:"name"`
					Content    json.RawMessage `json:"content"`
				}
				if json.Unmarshal(raw, &result) != nil {
					continue
				}
				toolID := result.ToolCallID
				if toolID == "" {
					toolID = result.ToolUseID
				}
				if toolID == "" {
					toolID = result.ID
				}
				toolName := result.ToolName
				if toolName == "" {
					toolName = result.Name
				}
				content := result.Content
				if len(content) == 0 {
					content = raw
				}
				p.emitTool(toolID, toolName, nil, content, events)
			}
		}
	}
}

func (p *piStreamParser) handleMessageUpdate(raw json.RawMessage, events chan<- Event) {
	var ev struct {
		Type     string `json:"type"`
		Delta    string `json:"delta"`
		ToolCall struct {
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
			Input     json.RawMessage `json:"input"`
		} `json:"toolCall"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		return
	}

	switch ev.Type {
	case "text_delta":
		if ev.Delta != "" {
			p.emitText(ev.Delta, events)
		}
	case "toolcall_start":
		p.markTextSepNeeded()
		if ev.ToolCall.ID != "" && !p.seenToolStarts[ev.ToolCall.ID] {
			p.seenToolStarts[ev.ToolCall.ID] = true
			events <- Event{
				Type:       EventToolCallStart,
				ToolCallID: ev.ToolCall.ID,
				ToolName:   ev.ToolCall.Name,
			}
		}
	case "toolcall_end":
		args := ev.ToolCall.Arguments
		if len(args) == 0 {
			args = ev.ToolCall.Input
		}
		p.emitTool(ev.ToolCall.ID, ev.ToolCall.Name, args, nil, events)
	}
}

func (p *piStreamParser) emitTool(toolID, toolName string, args, result json.RawMessage, events chan<- Event) {
	if toolID == "" {
		return
	}

	p.markTextSepNeeded()

	if !p.seenToolStarts[toolID] {
		p.seenToolStarts[toolID] = true
		events <- Event{
			Type:       EventToolCallStart,
			ToolCallID: toolID,
			ToolName:   toolName,
		}
	}

	if len(args) > 0 && string(args) != "null" && !p.seenToolEnds[toolID] {
		events <- Event{
			Type:       EventToolCallArgs,
			ToolCallID: toolID,
			ToolArgs:   string(args),
		}
	}

	if !p.seenToolEnds[toolID] {
		p.seenToolEnds[toolID] = true
		endArgs := ""
		if len(args) > 0 && string(args) != "null" {
			endArgs = string(args)
		}
		events <- Event{
			Type:       EventToolCallEnd,
			ToolCallID: toolID,
			ToolName:   toolName,
			ToolArgs:   endArgs,
		}
	}

	if len(result) > 0 && string(result) != "null" && !p.seenToolResults[toolID] {
		p.seenToolResults[toolID] = true
		events <- Event{
			Type:       EventToolCallResult,
			ToolCallID: toolID,
			Content:    string(result),
		}
		emitImageEvents(result, "", events)
	}
}
