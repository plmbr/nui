// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mentions

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

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
	tokens := findMentionTokens(message)
	if len(tokens) == 0 {
		return message, nil
	}
	var b strings.Builder
	last := 0
	for _, token := range tokens {
		b.WriteString(message[last:token.start])
		resolved, err := r.resolveToken(ctx, workingDir, token, allowed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mentions] resolve %q: %v\n", token.value, err)
			b.WriteString(message[token.start:token.end])
		} else {
			b.WriteString(resolved)
		}
		last = token.end
	}
	b.WriteString(message[last:])
	return b.String(), nil
}

// resolveToken resolves one mention. For an unquoted file or dir mention it
// tries the name as written, then with trailing punctuation
// removed, and uses the first name that exists; any other failure, such as a
// path outside the working directory, ends the search. Punctuation trimmed off
// is written back after the resolved path so the sentence around it is
// unchanged. Quoted and extension mentions are resolved exactly as written.
func (r *Registry) resolveToken(ctx context.Context, workingDir string, token mentionToken, allowed map[string]bool) (string, error) {
	req := ResolveRequest{WorkingDir: workingDir, Value: token.value, AllowedExtensionRoots: allowed}
	prefix, trims := "", false
	switch {
	case token.quoted:
	case strings.HasPrefix(token.value, fileValuePrefix):
		prefix, trims = fileValuePrefix, true
	case strings.HasPrefix(token.value, dirValuePrefix):
		prefix, trims = dirValuePrefix, true
	}
	if !trims {
		return r.Resolve(ctx, req)
	}
	var err error
	for _, candidate := range pathCandidates(prefix, token.value) {
		req.Value = candidate
		var resolved string
		resolved, err = r.Resolve(ctx, req)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return resolved + token.value[len(candidate):], nil
	}
	return "", err
}

func (r *Registry) Resolve(ctx context.Context, req ResolveRequest) (string, error) {
	value := req.Value
	if strings.TrimSpace(value) == "" {
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
