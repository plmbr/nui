// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"strings"
)

const (
	StorageKindSessionHistory = "sessionHistory"
	StorageKindAgentMemory    = "agentMemory"
	StorageKindUserMemory     = "userMemory"
)

// StorageContribution declares extension persistence handlers.
type StorageContribution struct {
	Source  Source         `yaml:"source"`
	Runtime *RuntimeConfig `yaml:"runtime,omitempty"`
}

// StorageHandlerEntry is one storage handler in a list file.
type StorageHandlerEntry struct {
	ID         string   `yaml:"id"                    json:"id"`
	Kind       string   `yaml:"kind"                  json:"kind"`
	AgentTypes []string `yaml:"agentTypes,omitempty"  json:"agentTypes,omitempty"`
}

// StorageHandlerRef binds a handler entry to its extension.
type StorageHandlerRef struct {
	ExtensionName string
	Handler       StorageHandlerEntry
}

func validateStorageHandler(entry StorageHandlerEntry, index int) error {
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("storageHandlers[%d]: id is required", index)
	}
	kind := strings.TrimSpace(entry.Kind)
	switch kind {
	case StorageKindSessionHistory, StorageKindAgentMemory:
		if len(entry.AgentTypes) == 0 {
			return fmt.Errorf("storageHandlers[%d]: %s requires agentTypes", index, kind)
		}
	case StorageKindUserMemory:
		if len(entry.AgentTypes) > 0 {
			return fmt.Errorf("storageHandlers[%d]: userMemory must not set agentTypes", index)
		}
	default:
		return fmt.Errorf("storageHandlers[%d]: kind must be sessionHistory, agentMemory, or userMemory", index)
	}
	return nil
}

func handlerMatchesAgentType(entry StorageHandlerEntry, agentID string) bool {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return false
	}
	for _, t := range entry.AgentTypes {
		if strings.TrimSpace(t) == agentID {
			return true
		}
	}
	return false
}

func loadStorageHandlersFromFile(path string) ([]StorageHandlerEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		StorageHandlers []StorageHandlerEntry `json:"storageHandlers" yaml:"storageHandlers"`
	}
	if err := decodeListFile(data, path, &wrap); err != nil {
		return nil, err
	}
	for i, h := range wrap.StorageHandlers {
		if err := validateStorageHandler(h, i); err != nil {
			return nil, err
		}
	}
	return wrap.StorageHandlers, nil
}

// LoadStorageHandlersFromFile loads and validates storage handlers from a list file.
func LoadStorageHandlersFromFile(path string) ([]StorageHandlerEntry, error) {
	return loadStorageHandlersFromFile(path)
}

func validateProgrammaticStorageHandlers(handlers []StorageHandlerEntry, extName string) error {
	for i, h := range handlers {
		if err := validateStorageHandler(h, i); err != nil {
			return fmt.Errorf("extension %s: %w", extName, err)
		}
	}
	return nil
}

func (r *Registry) handlersByKind(kind string, agentID string) []StorageHandlerRef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []StorageHandlerRef
	for name, ext := range r.extensions {
		if r.isDisabled(name) {
			continue
		}
		for _, h := range ext.StorageHandlers {
			if h.Kind != kind {
				continue
			}
			switch kind {
			case StorageKindUserMemory:
				out = append(out, StorageHandlerRef{ExtensionName: name, Handler: h})
			default:
				if handlerMatchesAgentType(h, agentID) {
					out = append(out, StorageHandlerRef{ExtensionName: name, Handler: h})
				}
			}
		}
	}
	return out
}

// SessionHandlers returns session history handlers for agentType.
func (r *Registry) SessionHandlers(agentType string) []StorageHandlerRef {
	if r == nil {
		return nil
	}
	return r.handlersByKind(StorageKindSessionHistory, agentType)
}

// AgentMemoryHandlers returns agent memory handlers for agentID.
func (r *Registry) AgentMemoryHandlers(agentID string) []StorageHandlerRef {
	if r == nil {
		return nil
	}
	return r.handlersByKind(StorageKindAgentMemory, agentID)
}

// UserMemoryHandlers returns user memory handlers.
func (r *Registry) UserMemoryHandlers() []StorageHandlerRef {
	if r == nil {
		return nil
	}
	return r.handlersByKind(StorageKindUserMemory, "")
}

// HasSessionHandler reports whether any session history handler matches agentType.
func (r *Registry) HasSessionHandler(agentType string) bool {
	return len(r.SessionHandlers(agentType)) > 0
}

// HasAgentMemoryHandler reports whether any agent memory handler matches agentID.
func (r *Registry) HasAgentMemoryHandler(agentID string) bool {
	return len(r.AgentMemoryHandlers(agentID)) > 0
}

// HasUserMemoryHandler reports whether any user memory handler is registered.
func (r *Registry) HasUserMemoryHandler() bool {
	return len(r.UserMemoryHandlers()) > 0
}
