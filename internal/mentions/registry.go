// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mentions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

var mentionTokenPattern = regexp.MustCompile(`@([^\s@]+)`)

// Registry routes mention list and resolve requests to providers.
type Registry struct {
	mu        sync.RWMutex
	builtin   Provider
	extension ExtensionSource
}

// ExtensionSource supplies extension-backed mention providers.
type ExtensionSource interface {
	ListExtensionRoots() []Item
	ListExtension(ctx context.Context, extName, providerID string, req ListRequest) (ListResponse, error)
	ResolveExtension(ctx context.Context, extName, providerID string, req ResolveRequest) (string, error)
	MatchExtensionValue(value string) (extName, providerID string, ok bool)
	MatchExtensionParent(parent string) (extName, providerID string, ok bool)
}

// DefaultRegistry is the process-wide mention registry.
var DefaultRegistry = NewRegistry(nil)

func NewRegistry(ext ExtensionSource) *Registry {
	return &Registry{
		builtin:   BuiltinFilesProvider{},
		extension: ext,
	}
}

func (r *Registry) SetExtensionSource(ext ExtensionSource) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.extension = ext
}

func (r *Registry) List(ctx context.Context, req ListRequest) (ListResponse, error) {
	req.Limit = normalizeLimit(req.Limit)
	parent := strings.TrimSpace(req.Parent)

	if parent == "" {
		return r.listRoots(ctx, req)
	}
	if parent == BuiltinFilesRoot {
		return r.builtin.List(ctx, req)
	}
	if strings.HasPrefix(parent, fileValuePrefix) {
		return ListResponse{}, fmt.Errorf("file mentions cannot be expanded")
	}
	if r.extension != nil {
		if extName, providerID, ok := r.extension.MatchExtensionParent(parent); ok {
			if !allowedExtensionMention(parent, req.AllowedExtensionRoots) {
				return ListResponse{}, fmt.Errorf("mention provider not enabled for this agent")
			}
			return r.extension.ListExtension(ctx, extName, providerID, req)
		}
	}
	return ListResponse{}, fmt.Errorf("unknown mention parent %q", parent)
}

func (r *Registry) listRoots(ctx context.Context, req ListRequest) (ListResponse, error) {
	resp, err := r.builtin.List(ctx, req)
	if err != nil {
		return ListResponse{}, err
	}
	items := append([]Item(nil), resp.Items...)
	if r.extension != nil {
		for _, item := range r.extension.ListExtensionRoots() {
			if !allowedExtensionMention(item.Value, req.AllowedExtensionRoots) {
				continue
			}
			items = append(items, item)
		}
	}
	items = filterItems(items, req.Query, req.Limit)
	return ListResponse{
		Items:      items,
		Breadcrumb: []Breadcrumb{{Label: "Root", Parent: ""}},
	}, nil
}

func filterItems(items []Item, query string, limit int) []Item {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		if len(items) > limit {
			return items[:limit]
		}
		return items
	}
	filtered := make([]Item, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Label), query) ||
			strings.Contains(strings.ToLower(item.Value), query) {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) > limit {
		return filtered[:limit]
	}
	return filtered
}

func (r *Registry) ResolveMessage(ctx context.Context, workingDir, message string, allowed map[string]bool) (string, error) {
	if strings.TrimSpace(message) == "" {
		return message, nil
	}
	matches := mentionTokenPattern.FindAllStringSubmatchIndex(message, -1)
	if len(matches) == 0 {
		return message, nil
	}
	var b strings.Builder
	last := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		valueStart, valueEnd := match[2], match[3]
		if start > 0 && !unicode.IsSpace(rune(message[start-1])) {
			continue
		}
		b.WriteString(message[last:start])
		value := message[valueStart:valueEnd]
		resolved, err := r.Resolve(ctx, ResolveRequest{
			WorkingDir:            workingDir,
			Value:                 value,
			AllowedExtensionRoots: allowed,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mentions] resolve %q: %v\n", value, err)
			b.WriteString(message[start:end])
		} else {
			b.WriteString(resolved)
		}
		last = end
	}
	b.WriteString(message[last:])
	return b.String(), nil
}

func (r *Registry) Resolve(ctx context.Context, req ResolveRequest) (string, error) {
	value := strings.TrimSpace(req.Value)
	if value == "" {
		return "", fmt.Errorf("empty mention value")
	}
	switch {
	case value == BuiltinFilesRoot:
		return "", fmt.Errorf("mention %q is not selectable", value)
	case strings.HasPrefix(value, fileValuePrefix), strings.HasPrefix(value, dirValuePrefix):
		return r.builtin.Resolve(ctx, req)
	case filepath.IsAbs(value):
		return resolveAbsoluteFileMention(value)
	}
	if r.extension != nil {
		if extName, providerID, ok := r.extension.MatchExtensionValue(value); ok {
			if !allowedExtensionMention(value, req.AllowedExtensionRoots) {
				return "", fmt.Errorf("mention provider not enabled for this agent")
			}
			return r.extension.ResolveExtension(ctx, extName, providerID, req)
		}
	}
	return "", fmt.Errorf("unknown mention value %q", value)
}

func allowedExtensionMention(value string, allowed map[string]bool) bool {
	if len(allowed) == 0 {
		return false
	}
	for root := range allowed {
		if value == root || strings.HasPrefix(value, root+":") {
			return true
		}
	}
	return false
}
