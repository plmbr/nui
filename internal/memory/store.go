// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package memory

import (
	"fmt"
	"os"
	"strings"
)

func readBuiltinUser() (string, error) {
	return readFile(UserPath)
}

func readBuiltinAgent(agentID string) (string, error) {
	path, err := AgentPath(agentID)
	if err != nil {
		return "", err
	}
	return readFile(func() (string, error) { return path, nil })
}

func writeBuiltinUser(content string) error {
	path, err := UserPath()
	if err != nil {
		return err
	}
	return writeFile(path, content)
}

func writeBuiltinAgent(agentID, content string) error {
	path, err := AgentPath(agentID)
	if err != nil {
		return err
	}
	return writeFile(path, content)
}

func deleteBuiltinUser() error {
	path, err := UserPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func deleteBuiltinAgent(agentID string) error {
	path, err := AgentPath(agentID)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Store abstracts memory persistence.
type Store interface {
	ReadUser() (string, error)
	ReadAgent(agentID string) (string, error)
	WriteUser(content string) error
	WriteAgent(agentID, content string) error
	Update(scope, agentID, content, writeMode string) (string, error)
	DeleteUser() error
	DeleteAgent(agentID string) error
}

type builtinStore struct{}

func (builtinStore) ReadUser() (string, error)                { return readBuiltinUser() }
func (builtinStore) ReadAgent(agentID string) (string, error) { return readBuiltinAgent(agentID) }
func (builtinStore) WriteUser(content string) error           { return writeBuiltinUser(content) }
func (builtinStore) WriteAgent(agentID, content string) error { return writeBuiltinAgent(agentID, content) }
func (builtinStore) DeleteUser() error                        { return deleteBuiltinUser() }
func (builtinStore) DeleteAgent(agentID string) error         { return deleteBuiltinAgent(agentID) }

func (builtinStore) Update(scope, agentID, content, writeMode string) (string, error) {
	return updateBuiltin(scope, agentID, content, writeMode)
}

func updateBuiltin(scope, agentID, content, writeMode string) (string, error) {
	scope = strings.TrimSpace(strings.ToLower(scope))
	writeMode = strings.TrimSpace(strings.ToLower(writeMode))
	if writeMode == "" {
		writeMode = "replace"
	}
	if writeMode != "replace" && writeMode != "append" {
		return "", fmt.Errorf("mode must be replace or append")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("content is required")
	}

	var path string
	var existing string
	var err error
	switch scope {
	case "user":
		path, err = UserPath()
		if err != nil {
			return "", err
		}
		existing, err = readBuiltinUser()
	case "agent":
		if strings.TrimSpace(agentID) == "" {
			return "", fmt.Errorf("agent scope requires agent id")
		}
		path, err = AgentPath(agentID)
		if err != nil {
			return "", err
		}
		existing, err = readBuiltinAgent(agentID)
	default:
		return "", fmt.Errorf("scope must be user or agent")
	}
	if err != nil {
		return "", err
	}
	next := content
	if writeMode == "append" {
		if existing != "" {
			next = existing + "\n\n" + content
		}
	}
	if err := writeFile(path, next); err != nil {
		return "", err
	}
	return path, nil
}

var currentStore Store = builtinStore{}

// ReadBuiltinUser reads user memory from builtin files only.
func ReadBuiltinUser() (string, error) { return readBuiltinUser() }

// ReadBuiltinAgent reads agent memory from builtin files only.
func ReadBuiltinAgent(agentID string) (string, error) { return readBuiltinAgent(agentID) }

// WriteBuiltinUser writes user memory to builtin files only.
func WriteBuiltinUser(content string) error { return writeBuiltinUser(content) }

// WriteBuiltinAgent writes agent memory to builtin files only.
func WriteBuiltinAgent(agentID, content string) error { return writeBuiltinAgent(agentID, content) }

// DeleteBuiltinUser removes the builtin user memory file.
func DeleteBuiltinUser() error { return deleteBuiltinUser() }

// DeleteBuiltinAgent removes the builtin agent memory file.
func DeleteBuiltinAgent(agentID string) error { return deleteBuiltinAgent(agentID) }

// UpdateBuiltin applies replace/append to builtin files only.
func UpdateBuiltin(scope, agentID, content, writeMode string) (string, error) {
	return updateBuiltin(scope, agentID, content, writeMode)
}

// SetStore replaces the active memory store.
func SetStore(s Store) {
	if s == nil {
		currentStore = builtinStore{}
		return
	}
	currentStore = s
}

// ResetStore restores the builtin file-only store (for tests).
func ResetStore() {
	currentStore = builtinStore{}
}
