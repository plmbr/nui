// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"encoding/json"
	"strings"
)

// NormalizePayload coerces common agent payload shapes into UI-friendly form.
func NormalizePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	payload = unwrapToolInputPayload(payload)
	out := make(map[string]any, len(payload))
	for k, v := range payload {
		out[k] = v
	}
	if msg := stringField(out, "message"); msg != "" {
		if clean, qs := extractQuestionsFromMessage(msg); len(qs) > 0 {
			out["message"] = clean
			if _, ok := out["questions"]; !ok {
				out["questions"] = qs
			}
		}
	}
	if questions := coerceQuestionsArray(out["questions"]); len(questions) > 0 {
		normalized := make([]any, 0, len(questions))
		for _, q := range questions {
			normalized = append(normalized, normalizeQuestion(q))
		}
		out["questions"] = filterRenderableQuestions(normalized)
	} else {
		delete(out, "questions")
	}
	synthesizeQuestionsIfEmpty(out)
	return out
}

// SalvageAskUserMessage extracts displayable prose from a blocked ask_user payload.
func SalvageAskUserMessage(args map[string]any) string {
	if args == nil {
		return ""
	}
	if msg := strings.TrimSpace(stringField(args, "message")); msg != "" {
		if clean, qs := extractQuestionsFromMessage(msg); len(qs) > 0 {
			return clean
		}
		return msg
	}
	return ""
}

func unwrapToolInputPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	if _, ok := payload["questions"]; ok {
		return payload
	}
	for _, key := range []string{"tool_input", "toolInput", "input"} {
		nested, ok := payload[key].(map[string]any)
		if !ok {
			continue
		}
		if _, hasQuestions := nested["questions"]; !hasQuestions {
			if questionText(nested) == "" && stringField(nested, "message") == "" {
				continue
			}
		}
		out := make(map[string]any, len(payload)+3)
		for k, v := range payload {
			out[k] = v
		}
		if title := stringField(nested, "title"); title != "" {
			out["title"] = title
		}
		if message := stringField(nested, "message"); message != "" {
			out["message"] = message
		}
		if questions, ok := nested["questions"]; ok {
			out["questions"] = questions
		}
		if _, ok := out["questions"]; !ok {
			if q := questionText(nested); q != "" {
				out["questions"] = []any{map[string]any{"question": q}}
			}
		}
		return out
	}
	return payload
}

func coerceQuestionsArray(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		var parsed []any
		if json.Unmarshal([]byte(s), &parsed) == nil {
			return parsed
		}
		return []any{map[string]any{"question": s}}
	default:
		return nil
	}
}

func normalizeQuestion(q any) any {
	switch v := q.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return map[string]any{}
		}
		return map[string]any{"question": s}
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[k] = val
		}
		if text := questionText(out); text != "" {
			out["question"] = text
		} else if header := strings.TrimSpace(stringField(out, "header")); header != "" {
			out["question"] = header
		}
		if header := strings.TrimSpace(stringField(out, "header")); header == "" {
			if id := strings.TrimSpace(stringField(out, "id")); id != "" {
				out["header"] = id
			}
		}
		if raw, ok := out["options"].([]any); ok {
			out["options"] = normalizeOptions(raw)
		}
		return out
	default:
		return q
	}
}

func questionText(m map[string]any) string {
	for _, key := range []string{"question", "prompt", "text"} {
		if s := strings.TrimSpace(stringField(m, key)); s != "" {
			return s
		}
	}
	return ""
}

func filterRenderableQuestions(questions []any) []any {
	out := make([]any, 0, len(questions))
	for _, q := range questions {
		m, ok := q.(map[string]any)
		if !ok {
			continue
		}
		if questionText(m) != "" {
			out = append(out, m)
			continue
		}
		if raw, ok := m["options"].([]any); ok && len(raw) > 0 {
			out = append(out, m)
		}
	}
	return out
}

func extractQuestionsFromMessage(message string) (cleanMessage string, questions []any) {
	s := strings.TrimSpace(message)
	idx := strings.Index(s, "[{")
	if idx < 0 {
		return message, nil
	}
	candidate := strings.TrimSpace(s[idx:])
	var parsed []any
	if json.Unmarshal([]byte(candidate), &parsed) != nil || len(parsed) == 0 {
		return message, nil
	}
	first, ok := parsed[0].(map[string]any)
	if !ok {
		return message, nil
	}
	if questionText(first) == "" && stringField(first, "header") == "" {
		if _, hasOptions := first["options"]; !hasOptions {
			return message, nil
		}
	}
	clean := strings.TrimSpace(s[:idx])
	return clean, parsed
}

func synthesizeQuestionsIfEmpty(out map[string]any) {
	if qs, ok := out["questions"].([]any); ok && len(qs) > 0 {
		return
	}
	for _, key := range []string{"message", "question", "prompt", "title"} {
		if s := strings.TrimSpace(stringField(out, key)); s != "" {
			out["questions"] = []any{map[string]any{"question": s}}
			return
		}
	}
}

func normalizeOptions(raw []any) []any {
	out := make([]any, 0, len(raw))
	for _, item := range raw {
		switch v := item.(type) {
		case string:
			s := strings.TrimSpace(v)
			if s != "" {
				out = append(out, map[string]any{"label": s})
			}
		case map[string]any:
			out = append(out, normalizeOptionObject(v))
		default:
			if s := strings.TrimSpace(fmtAny(v)); s != "" {
				out = append(out, map[string]any{"label": s})
			}
		}
	}
	return out
}

func normalizeOptionObject(v map[string]any) map[string]any {
	label := strings.TrimSpace(stringField(v, "label"))
	if label == "" {
		label = strings.TrimSpace(stringField(v, "name"))
	}
	if label == "" {
		label = strings.TrimSpace(stringField(v, "value"))
	}
	if label == "" {
		label = strings.TrimSpace(stringField(v, "text"))
	}
	if label == "" {
		label = strings.TrimSpace(stringField(v, "id"))
	}
	out := map[string]any{"label": label}
	if desc := strings.TrimSpace(stringField(v, "description")); desc != "" {
		out["description"] = desc
	}
	return out
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func fmtAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}
