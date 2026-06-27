// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"loop/internal/mentions"
)

const extMentionPrefix = "ext:"

// MentionProvidersContribution declares extension mention providers.
type MentionProvidersContribution struct {
	Source  Source         `yaml:"source"`
	Runtime *RuntimeConfig `yaml:"runtime,omitempty"`
}

// MentionProviderEntry is one mention provider in a list file.
type MentionProviderEntry struct {
	ID          string `yaml:"id"                    json:"id"`
	DisplayName string `yaml:"displayName"           json:"displayName"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

func extMentionRootValue(extName, providerID string) string {
	return extMentionPrefix + extName + ":" + providerID
}

func parseExtMentionKey(key string) (extName, providerID string, ok bool) {
	if !strings.HasPrefix(key, extMentionPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(key, extMentionPrefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	extName = parts[0]
	providerID, _, _ = strings.Cut(parts[1], ":")
	if providerID == "" {
		return "", "", false
	}
	return extName, providerID, true
}

// MentionExtensionSource adapts Registry to mentions.ExtensionSource.
type MentionExtensionSource struct {
	Registry *Registry
}

func (s MentionExtensionSource) ListExtensionRoots() []mentions.Item {
	if s.Registry == nil {
		return nil
	}
	s.Registry.mu.RLock()
	defer s.Registry.mu.RUnlock()
	var items []mentions.Item
	for name, ext := range s.Registry.extensions {
		if s.Registry.isDisabled(name) {
			continue
		}
		for _, p := range ext.MentionProviders {
			label := strings.TrimSpace(p.DisplayName)
			if label == "" {
				label = p.ID
			}
			items = append(items, mentions.Item{
				Label:       label,
				Value:       extMentionRootValue(ext.Manifest.Name, p.ID),
				HasChildren: true,
				Icon:        "extension",
			})
		}
	}
	return items
}

func (s MentionExtensionSource) ListExtension(ctx context.Context, extName, providerID string, req mentions.ListRequest) (mentions.ListResponse, error) {
	client, err := s.Registry.mentionClient(extName, providerID)
	if err != nil {
		return mentions.ListResponse{}, err
	}
	return client.List(ctx, req, providerID)
}

func (s MentionExtensionSource) ResolveExtension(ctx context.Context, extName, providerID string, req mentions.ResolveRequest) (string, error) {
	client, err := s.Registry.mentionClient(extName, providerID)
	if err != nil {
		return "", err
	}
	return client.Resolve(ctx, req, providerID)
}

func (s MentionExtensionSource) MatchExtensionValue(value string) (extName, providerID string, ok bool) {
	return parseExtMentionKey(value)
}

func (s MentionExtensionSource) MatchExtensionParent(parent string) (extName, providerID string, ok bool) {
	return parseExtMentionKey(parent)
}

type mentionClientCache struct {
	mu      sync.Mutex
	entries map[string]*mentionClientEntry
}

type mentionClientEntry struct {
	client   *MentionRPCClient
	lastUsed time.Time
}

func newMentionClientCache() *mentionClientCache {
	return &mentionClientCache{entries: map[string]*mentionClientEntry{}}
}

func (c *mentionClientCache) get(key string, open func() (*MentionRPCClient, error)) (*MentionRPCClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	const idle = 60 * time.Second
	now := time.Now()
	if entry, ok := c.entries[key]; ok {
		if now.Sub(entry.lastUsed) > idle {
			_ = entry.client.Close()
			delete(c.entries, key)
		} else {
			entry.lastUsed = now
			return entry.client, nil
		}
	}
	client, err := open()
	if err != nil {
		return nil, err
	}
	c.entries[key] = &mentionClientEntry{client: client, lastUsed: now}
	return client, nil
}

func (c *mentionClientCache) closeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.entries {
		_ = entry.client.Close()
		delete(c.entries, key)
	}
}

func (r *Registry) mentionClient(extName, providerID string) (*MentionRPCClient, error) {
	r.mu.RLock()
	ext, ok := r.extensions[extName]
	if r.isDisabled(extName) || !ok {
		r.mu.RUnlock()
		return nil, fmt.Errorf("extension %q not found", extName)
	}
	if ext.mentionRuntime == nil {
		r.mu.RUnlock()
		return nil, fmt.Errorf("extension %q has no mention runtime", extName)
	}
	extDir := ext.Dir
	runtime := *ext.mentionRuntime
	r.mu.RUnlock()

	if r.mentionCache == nil {
		r.mentionCache = newMentionClientCache()
	}
	key := extName + ":" + providerID
	return r.mentionCache.get(key, func() (*MentionRPCClient, error) {
		return NewMentionRPCClient(extDir, extName, providerID, runtime)
	})
}

// MentionSource returns the extension mention adapter for the mentions registry.
func (r *Registry) MentionSource() mentions.ExtensionSource {
	return MentionExtensionSource{Registry: r}
}

func loadMentionProvidersFromFile(path string) ([]MentionProviderEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		MentionProviders []MentionProviderEntry `json:"mentionProviders" yaml:"mentionProviders"`
	}
	if err := decodeListFile(data, path, &wrap); err != nil {
		return nil, err
	}
	for i, p := range wrap.MentionProviders {
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("mentionProviders[%d]: id is required", i)
		}
	}
	return wrap.MentionProviders, nil
}
