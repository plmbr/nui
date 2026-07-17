// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package storageext

import (
	"context"
	"fmt"
	"os"
	"strings"

	"nui/internal/extensions"
	"nui/internal/memory"
	"nui/internal/model"
	"nui/internal/store"
)

// Coordinator routes persistence to extension handlers or built-in storage.
type Coordinator struct {
	Registry *extensions.Registry
}

// Default is the process-wide persistence coordinator.
var Default *Coordinator

func NewCoordinator(reg *extensions.Registry) *Coordinator {
	c := &Coordinator{Registry: reg}
	Default = c
	return c
}

func mergeContent(parts ...string) string {
	var blocks []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			blocks = append(blocks, p)
		}
	}
	return strings.Join(blocks, "\n\n")
}

func (c *Coordinator) asyncCall(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil {
			fmt.Fprintf(os.Stderr, "warn: storage %s: %v\n", label, err)
		}
	}()
}

func (c *Coordinator) client(extName string) (*extensions.StorageRPCClient, error) {
	if c == nil || c.Registry == nil {
		return nil, fmt.Errorf("no registry")
	}
	return c.Registry.StorageClient(extName)
}

// --- memory.Store ---

func (c *Coordinator) ReadUser() (string, error) {
	if c == nil || c.Registry == nil || !c.Registry.HasUserMemoryHandler() {
		return readBuiltinUser()
	}
	ctx := context.Background()
	var parts []string
	for _, ref := range c.Registry.UserMemoryHandlers() {
		client, err := c.client(ref.ExtensionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: storage user read %s/%s: %v\n", ref.ExtensionName, ref.Handler.ID, err)
			continue
		}
		content, err := client.ReadUserMemory(ctx, ref.Handler.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: storage user read %s/%s: %v\n", ref.ExtensionName, ref.Handler.ID, err)
			continue
		}
		parts = append(parts, content)
	}
	return mergeContent(parts...), nil
}

func (c *Coordinator) ReadAgent(agentID string) (string, error) {
	if c == nil || c.Registry == nil || !c.Registry.HasAgentMemoryHandler(agentID) {
		return readBuiltinAgent(agentID)
	}
	ctx := context.Background()
	var parts []string
	for _, ref := range c.Registry.AgentMemoryHandlers(agentID) {
		client, err := c.client(ref.ExtensionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: storage agent read %s/%s: %v\n", ref.ExtensionName, ref.Handler.ID, err)
			continue
		}
		content, err := client.ReadAgentMemory(ctx, ref.Handler.ID, agentID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: storage agent read %s/%s: %v\n", ref.ExtensionName, ref.Handler.ID, err)
			continue
		}
		parts = append(parts, content)
	}
	return mergeContent(parts...), nil
}

func (c *Coordinator) WriteUser(content string) error {
	if c == nil || c.Registry == nil || !c.Registry.HasUserMemoryHandler() {
		return writeBuiltinUser(content)
	}
	c.fanOutUserWrite(content, "replace")
	return nil
}

func (c *Coordinator) WriteAgent(agentID, content string) error {
	if c == nil || c.Registry == nil || !c.Registry.HasAgentMemoryHandler(agentID) {
		return writeBuiltinAgent(agentID, content)
	}
	c.fanOutAgentWrite(agentID, content, "replace")
	return nil
}

func (c *Coordinator) Update(scope, agentID, content, writeMode string) (string, error) {
	writeMode = strings.TrimSpace(strings.ToLower(writeMode))
	if writeMode == "" {
		writeMode = "replace"
	}
	scope = strings.TrimSpace(strings.ToLower(scope))
	switch scope {
	case "user":
		if c != nil && c.Registry != nil && c.Registry.HasUserMemoryHandler() {
			next := content
			if writeMode == "append" {
				existing, err := c.ReadUser()
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(existing) != "" {
					next = strings.TrimSpace(existing) + "\n\n" + strings.TrimSpace(content)
				}
			}
			c.fanOutUserWrite(next, "replace")
			path, _ := memory.UserPath()
			return path, nil
		}
		return updateBuiltin(scope, agentID, content, writeMode)
	case "agent":
		if strings.TrimSpace(agentID) == "" {
			return "", fmt.Errorf("agent scope requires agent id")
		}
		if c != nil && c.Registry != nil && c.Registry.HasAgentMemoryHandler(agentID) {
			next := content
			if writeMode == "append" {
				existing, err := c.ReadAgent(agentID)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(existing) != "" {
					next = strings.TrimSpace(existing) + "\n\n" + strings.TrimSpace(content)
				}
			}
			c.fanOutAgentWrite(agentID, next, "replace")
			path, err := memory.AgentPath(agentID)
			return path, err
		}
		return updateBuiltin(scope, agentID, content, writeMode)
	default:
		return "", fmt.Errorf("scope must be user or agent")
	}
}

func (c *Coordinator) DeleteUser() error {
	if c == nil || c.Registry == nil || !c.Registry.HasUserMemoryHandler() {
		return deleteBuiltinUser()
	}
	ctx := context.Background()
	for _, ref := range c.Registry.UserMemoryHandlers() {
		ref := ref
		c.asyncCall(fmt.Sprintf("user delete %s/%s", ref.ExtensionName, ref.Handler.ID), func() error {
			client, err := c.client(ref.ExtensionName)
			if err != nil {
				return err
			}
			return client.DeleteUserMemory(ctx, ref.Handler.ID)
		})
	}
	return nil
}

func (c *Coordinator) DeleteAgent(agentID string) error {
	if c == nil || c.Registry == nil || !c.Registry.HasAgentMemoryHandler(agentID) {
		return deleteBuiltinAgent(agentID)
	}
	ctx := context.Background()
	for _, ref := range c.Registry.AgentMemoryHandlers(agentID) {
		ref := ref
		c.asyncCall(fmt.Sprintf("agent delete %s/%s", ref.ExtensionName, ref.Handler.ID), func() error {
			client, err := c.client(ref.ExtensionName)
			if err != nil {
				return err
			}
			return client.DeleteAgentMemory(ctx, ref.Handler.ID, agentID)
		})
	}
	return nil
}

func (c *Coordinator) fanOutUserWrite(content, writeMode string) {
	if c == nil || c.Registry == nil {
		return
	}
	ctx := context.Background()
	for _, ref := range c.Registry.UserMemoryHandlers() {
		ref := ref
		c.asyncCall(fmt.Sprintf("user write %s/%s", ref.ExtensionName, ref.Handler.ID), func() error {
			client, err := c.client(ref.ExtensionName)
			if err != nil {
				return err
			}
			return client.WriteUserMemory(ctx, ref.Handler.ID, content, writeMode)
		})
	}
}

func (c *Coordinator) fanOutAgentWrite(agentID, content, writeMode string) {
	if c == nil || c.Registry == nil {
		return
	}
	ctx := context.Background()
	for _, ref := range c.Registry.AgentMemoryHandlers(agentID) {
		ref := ref
		c.asyncCall(fmt.Sprintf("agent write %s/%s", ref.ExtensionName, ref.Handler.ID), func() error {
			client, err := c.client(ref.ExtensionName)
			if err != nil {
				return err
			}
			return client.WriteAgentMemory(ctx, ref.Handler.ID, agentID, content, writeMode)
		})
	}
}

// --- session persistence ---

func (c *Coordinator) HasSessionHandler(agentType string) bool {
	return c != nil && c.Registry != nil && c.Registry.HasSessionHandler(agentType)
}

func (c *Coordinator) ReadSession(sessionID, agentType, workingDir string) ([]model.ChatMessage, string, error) {
	if !c.HasSessionHandler(agentType) {
		return nil, "", fmt.Errorf("no session handler")
	}
	ctx := context.Background()
	for _, ref := range c.Registry.SessionHandlers(agentType) {
		client, err := c.client(ref.ExtensionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: storage session read %s/%s: %v\n", ref.ExtensionName, ref.Handler.ID, err)
			continue
		}
		msgs, agentSessionID, err := client.ReadSession(ctx, ref.Handler.ID, sessionID, agentType, workingDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: storage session read %s/%s: %v\n", ref.ExtensionName, ref.Handler.ID, err)
			continue
		}
		return msgs, agentSessionID, nil
	}
	return []model.ChatMessage{}, "", nil
}

func (c *Coordinator) WriteSession(sessionID, agentType, workingDir, agentSessionID string, messages []model.ChatMessage) {
	if !c.HasSessionHandler(agentType) {
		return
	}
	ctx := context.Background()
	copied := append([]model.ChatMessage(nil), messages...)
	for _, ref := range c.Registry.SessionHandlers(agentType) {
		ref := ref
		c.asyncCall(fmt.Sprintf("session write %s/%s", ref.ExtensionName, ref.Handler.ID), func() error {
			client, err := c.client(ref.ExtensionName)
			if err != nil {
				return err
			}
			return client.WriteSession(ctx, ref.Handler.ID, sessionID, agentType, agentSessionID, workingDir, copied)
		})
	}
}

func (c *Coordinator) DeleteSession(sessionID, agentType, workingDir, agentSessionID string) {
	if !c.HasSessionHandler(agentType) {
		return
	}
	ctx := context.Background()
	for _, ref := range c.Registry.SessionHandlers(agentType) {
		ref := ref
		c.asyncCall(fmt.Sprintf("session delete %s/%s", ref.ExtensionName, ref.Handler.ID), func() error {
			client, err := c.client(ref.ExtensionName)
			if err != nil {
				return err
			}
			return client.DeleteSession(ctx, ref.Handler.ID, sessionID, agentType, agentSessionID, workingDir)
		})
	}
}

// builtin helpers (duplicate memory package unexported funcs via exported paths)
func readBuiltinUser() (string, error) {
	return memory.ReadBuiltinUser()
}
func readBuiltinAgent(agentID string) (string, error) {
	return memory.ReadBuiltinAgent(agentID)
}
func writeBuiltinUser(content string) error {
	return memory.WriteBuiltinUser(content)
}
func writeBuiltinAgent(agentID, content string) error {
	return memory.WriteBuiltinAgent(agentID, content)
}
func deleteBuiltinUser() error {
	return memory.DeleteBuiltinUser()
}
func deleteBuiltinAgent(agentID string) error {
	return memory.DeleteBuiltinAgent(agentID)
}
func updateBuiltin(scope, agentID, content, writeMode string) (string, error) {
	return memory.UpdateBuiltin(scope, agentID, content, writeMode)
}

// ListSummary returns memory metadata, preferring extension-backed reads when handlers exist.
func (c *Coordinator) ListSummary(settings store.Settings) (memory.Summary, error) {
	out, err := memory.ListSummary(settings)
	if err != nil {
		return out, err
	}
	if c == nil || c.Registry == nil {
		return out, nil
	}
	if c.Registry.HasUserMemoryHandler() {
		out.User.UpdatedAt = ""
		if content, readErr := c.ReadUser(); readErr == nil {
			out.User.Size = int64(len(content))
		}
	}
	var agents []memory.AgentEntry
	for _, ag := range out.Agents {
		if c.Registry.HasAgentMemoryHandler(ag.AgentID) {
			continue
		}
		agents = append(agents, ag)
	}
	out.Agents = agents
	return out, nil
}
