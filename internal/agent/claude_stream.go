// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type claudeStreamParser struct {
	emittedText      bool
	seenToolStarts   map[string]bool
	seenToolEnds     map[string]bool
	seenToolResults  map[string]bool
	blocks           map[int]*claudeBlockState
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
		ParentToolUseID  string          `json:"parent_tool_use_id"`
		ToolUseResult    json.RawMessage `json:"tool_use_result"`
		Message          json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return
	}

	switch envelope.Type {
	case "stream_event":
		p.handleStreamEvent(envelope.Event, events)
	case "assistant":
		p.handleAssistant(envelope.Message, events)
	case "user":
		p.handleUser(envelope.ParentToolUseID, envelope.ToolUseResult, envelope.Message, events)
	case "result":
		if envelope.IsError {
			events <- Event{Type: EventError, Error: envelope.ErrMsg}
		} else {
			events <- Event{Type: EventDone, SessionID: envelope.SessionID}
		}
	}
}

func (p *claudeStreamParser) handleStreamEvent(raw json.RawMessage, events chan<- Event) {
	var ev struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
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
	case "content_block_delta":
		if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
			p.emittedText = true
			events <- Event{Type: EventText, Content: ev.Delta.Text}
		}
	case "content_block_start":
		block := ev.ContentBlock
		if block.Type != "tool_use" || block.ID == "" {
			return
		}
		p.blocks[ev.Index] = &claudeBlockState{
			kind:     "tool_use",
			toolID:   block.ID,
			toolName: block.Name,
		}
		if p.seenToolStarts[block.ID] {
			return
		}
		p.seenToolStarts[block.ID] = true
		events <- Event{
			Type:       EventToolCallStart,
			ToolCallID: block.ID,
			ToolName:   block.Name,
		}
	case "content_block_stop":
		state, ok := p.blocks[ev.Index]
		if !ok || state.kind != "tool_use" {
			delete(p.blocks, ev.Index)
			return
		}
		args := state.args.String()
		if args != "" {
			events <- Event{
				Type:       EventToolCallArgs,
				ToolCallID: state.toolID,
				ToolArgs:   args,
			}
		}
		if !p.seenToolEnds[state.toolID] {
			p.seenToolEnds[state.toolID] = true
			events <- Event{
				Type:       EventToolCallEnd,
				ToolCallID: state.toolID,
				ToolName:   state.toolName,
				ToolArgs:   args,
			}
		}
		delete(p.blocks, ev.Index)
	}

	if ev.Type == "content_block_delta" && ev.Delta.Type == "input_json_delta" && ev.Delta.PartialJSON != "" {
		if state, ok := p.blocks[ev.Index]; ok && state.kind == "tool_use" {
			state.args.WriteString(ev.Delta.PartialJSON)
		}
	}
}

func (p *claudeStreamParser) handleAssistant(raw json.RawMessage, events chan<- Event) {
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
			if json.Unmarshal(blockRaw, &block) == nil && block.Text != "" && !p.emittedText {
				events <- Event{Type: EventText, Content: block.Text}
			}
		case "tool_use":
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
			if !p.seenToolStarts[block.ID] {
				p.seenToolStarts[block.ID] = true
				events <- Event{
					Type:       EventToolCallStart,
					ToolCallID: block.ID,
					ToolName:   block.Name,
				}
				events <- Event{
					Type:       EventToolCallArgs,
					ToolCallID: block.ID,
					ToolArgs:   argsStr,
				}
			}
			if !p.seenToolEnds[block.ID] {
				p.seenToolEnds[block.ID] = true
				events <- Event{
					Type:       EventToolCallEnd,
					ToolCallID: block.ID,
					ToolName:   block.Name,
					ToolArgs:   argsStr,
				}
			}
		case "image":
			if md := imageBlockMarkdown(blockRaw); md != "" {
				events <- Event{Type: EventText, Content: md}
			}
		}
	}
}

func (p *claudeStreamParser) handleUser(parentToolUseID string, toolUseResult json.RawMessage, messageRaw json.RawMessage, events chan<- Event) {
	toolUseID := parentToolUseID
	if toolUseID == "" {
		toolUseID = toolUseIDFromMessage(messageRaw)
	}
	if toolUseID != "" && len(toolUseResult) > 0 && toolUseResult[0] != 'n' {
		p.emitToolResult(toolUseID, toolUseResult, events)
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
		p.emitToolResult(block.ToolUseID, block.Content, events)
		for _, md := range imageContentMarkdown(block.Content) {
			events <- Event{Type: EventText, Content: md}
		}
	}
}

func (p *claudeStreamParser) emitToolResult(toolUseID string, result json.RawMessage, events chan<- Event) {
	if p.seenToolResults[toolUseID] {
		return
	}
	p.seenToolResults[toolUseID] = true
	events <- Event{
		Type:       EventToolCallResult,
		ToolCallID: toolUseID,
		Content:    string(result),
	}
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

func imageBlockMarkdown(blockRaw json.RawMessage) string {
	var block struct {
		Source struct {
			Type      string `json:"type"`
			MediaType string `json:"media_type"`
			Data      string `json:"data"`
			URL       string `json:"url"`
		} `json:"source"`
	}
	if err := json.Unmarshal(blockRaw, &block); err != nil {
		return ""
	}
	return imageSourceMarkdown(block.Source.Type, block.Source.MediaType, block.Source.Data, block.Source.URL)
}

func imageContentMarkdown(content json.RawMessage) []string {
	var out []string

	var blocks []json.RawMessage
	if err := json.Unmarshal(content, &blocks); err == nil {
		for _, blockRaw := range blocks {
			var block struct {
				Type   string `json:"type"`
				Source struct {
					Type      string `json:"type"`
					MediaType string `json:"media_type"`
					Data      string `json:"data"`
					URL       string `json:"url"`
				} `json:"source"`
			}
			if json.Unmarshal(blockRaw, &block) != nil || block.Type != "image" {
				continue
			}
			if md := imageSourceMarkdown(block.Source.Type, block.Source.MediaType, block.Source.Data, block.Source.URL); md != "" {
				out = append(out, md)
			}
		}
		return out
	}

	var obj map[string]any
	if err := json.Unmarshal(content, &obj); err != nil {
		return out
	}
	walkImages(obj, &out)
	return out
}

func walkImages(v any, out *[]string) {
	switch val := v.(type) {
	case map[string]any:
		if typ, _ := val["type"].(string); typ == "image" {
			source, _ := val["source"].(map[string]any)
			if source != nil {
				srcType, _ := source["type"].(string)
				mediaType, _ := source["media_type"].(string)
				data, _ := source["data"].(string)
				url, _ := source["url"].(string)
				if md := imageSourceMarkdown(srcType, mediaType, data, url); md != "" {
					*out = append(*out, md)
				}
			}
		}
		for _, child := range val {
			walkImages(child, out)
		}
	case []any:
		for _, child := range val {
			walkImages(child, out)
		}
	}
}

func imageSourceMarkdown(srcType, mediaType, data, url string) string {
	switch srcType {
	case "base64":
		if data == "" {
			return ""
		}
		if mediaType == "" {
			mediaType = "image/png"
		}
		return fmt.Sprintf("\n\n![image](data:%s;base64,%s)\n\n", mediaType, data)
	case "url":
		if url == "" {
			return ""
		}
		return fmt.Sprintf("\n\n![image](%s)\n\n", url)
	default:
		if url != "" {
			return fmt.Sprintf("\n\n![image](%s)\n\n", url)
		}
		return ""
	}
}
