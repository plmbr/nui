// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
)

type openCodeStreamParser struct {
	seenToolStarts  map[string]bool
	seenToolEnds    map[string]bool
	seenToolResults map[string]bool
}

func newOpenCodeStreamParser() *openCodeStreamParser {
	return &openCodeStreamParser{
		seenToolStarts:  map[string]bool{},
		seenToolEnds:    map[string]bool{},
		seenToolResults: map[string]bool{},
	}
}

func (p *openCodeStreamParser) handleLine(line []byte, events chan<- Event) {
	var obj struct {
		Type string `json:"type"`
		Part struct {
			Type   string `json:"type"`
			Text   string `json:"text"`
			CallID string `json:"callID"`
			ID     string `json:"id"`
			Tool   string `json:"tool"`
			State  struct {
				Status string          `json:"status"`
				Input  json.RawMessage `json:"input"`
				Output json.RawMessage `json:"output"`
				Metadata json.RawMessage `json:"metadata"`
			} `json:"state"`
		} `json:"part"`
	}
	if err := json.Unmarshal(line, &obj); err != nil {
		return
	}

	if obj.Type == "text" && obj.Part.Type == "text" && obj.Part.Text != "" {
		events <- Event{Type: EventText, Content: obj.Part.Text}
		return
	}
	if obj.Type != "tool_use" {
		return
	}
	if obj.Part.State.Status != "completed" {
		return
	}

	toolID := obj.Part.CallID
	if toolID == "" {
		toolID = obj.Part.ID
	}
	if toolID == "" {
		return
	}

	toolName := obj.Part.Tool
	argsJSON, _ := json.Marshal(obj.Part.State.Input)
	if string(argsJSON) == "null" {
		argsJSON = []byte("{}")
	}

	if !p.seenToolStarts[toolID] {
		p.seenToolStarts[toolID] = true
		events <- Event{
			Type:       EventToolCallStart,
			ToolCallID: toolID,
			ToolName:   toolName,
		}
	}
	events <- Event{
		Type:       EventToolCallArgs,
		ToolCallID: toolID,
		ToolArgs:   string(argsJSON),
	}
	if !p.seenToolEnds[toolID] {
		p.seenToolEnds[toolID] = true
		events <- Event{
			Type:       EventToolCallEnd,
			ToolCallID: toolID,
			ToolName:   toolName,
			ToolArgs:   string(argsJSON),
		}
	}
	if p.seenToolResults[toolID] {
		return
	}
	p.seenToolResults[toolID] = true

	resultPayload := obj.Part.State.Output
	if len(resultPayload) > 0 && resultPayload[0] == '"' {
		var decoded string
		if json.Unmarshal(resultPayload, &decoded) == nil {
			var parsed any
			if json.Unmarshal([]byte(decoded), &parsed) == nil {
				resultPayload, _ = json.Marshal(parsed)
			} else {
				resultPayload = []byte(decoded)
			}
		}
	} else if len(obj.Part.State.Metadata) > 0 && string(obj.Part.State.Metadata) != "null" {
		resultPayload, _ = json.Marshal(map[string]any{
			"output":   rawJSONToAny(obj.Part.State.Output),
			"metadata": rawJSONToAny(obj.Part.State.Metadata),
		})
	}

	events <- Event{
		Type:       EventToolCallResult,
		ToolCallID: toolID,
		Content:    string(resultPayload),
	}
	emitImageEvents(resultPayload, "", events)
	emitImageEvents(obj.Part.State.Metadata, "", events)
	if stateRaw, err := json.Marshal(obj.Part.State); err == nil {
		emitImageEvents(stateRaw, "", events)
	}
}

func rawJSONToAny(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var v any
	if json.Unmarshal(raw, &v) == nil {
		return v
	}
	return string(raw)
}
