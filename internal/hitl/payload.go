// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import "strings"

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
	if questions, ok := out["questions"].([]any); ok {
		normalized := make([]any, len(questions))
		for i, q := range questions {
			normalized[i] = normalizeQuestion(q)
		}
		out["questions"] = normalized
	}
	return out
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
			continue
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
		out["questions"] = nested["questions"]
		return out
	}
	return payload
}

func normalizeQuestion(q any) any {
	m, ok := q.(map[string]any)
	if !ok {
		return q
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	if raw, ok := out["options"].([]any); ok {
		out["options"] = normalizeOptions(raw)
	}
	return out
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
