// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mentions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	BuiltinFilesRoot = "builtin:files"
	fileValuePrefix  = "file:"
	dirValuePrefix   = "dir:"
)

// BuiltinFilesProvider lists files and folders under the session working directory.
type BuiltinFilesProvider struct{}

func (BuiltinFilesProvider) ID() string          { return "builtin:files" }
func (BuiltinFilesProvider) DisplayName() string { return "Files & folders" }

func (BuiltinFilesProvider) List(_ context.Context, req ListRequest) (ListResponse, error) {
	parent := strings.TrimSpace(req.Parent)
	query := strings.TrimSpace(req.Query)

	if parent == "" {
		return ListResponse{
			Items: []Item{{
				Label:       "Files & folders",
				Value:       BuiltinFilesRoot,
				HasChildren: true,
				Icon:        "folder",
			}},
		}, nil
	}

	if parent != BuiltinFilesRoot {
		return ListResponse{}, fmt.Errorf("unknown parent %q", parent)
	}

	items, err := collectFilesBFS(req.WorkingDir, query, normalizeFileLimit(req.Limit))
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{
		Items: items,
		Breadcrumb: []Breadcrumb{
			{Label: "Root", Parent: ""},
			{Label: "Files & folders", Parent: BuiltinFilesRoot},
		},
	}, nil
}

func (BuiltinFilesProvider) Resolve(_ context.Context, req ResolveRequest) (string, error) {
	value := strings.TrimSpace(req.Value)
	switch {
	case strings.HasPrefix(value, fileValuePrefix):
		rel := strings.TrimPrefix(value, fileValuePrefix)
		abs, err := resolveFilePath(req.WorkingDir, rel)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("mention is a directory, not a file: %s", abs)
		}
		return "@" + abs, nil
	case strings.HasPrefix(value, dirValuePrefix):
		rel := strings.TrimPrefix(value, dirValuePrefix)
		abs, err := resolveFilePath(req.WorkingDir, rel)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", fmt.Errorf("mention is not a directory: %s", abs)
		}
		return "@" + abs, nil
	default:
		return "", fmt.Errorf("unsupported mention value %q", value)
	}
}

type walkEntry struct {
	absDir string
	relDir string
}

type walkCandidate struct {
	entry os.DirEntry
	rel   string
	abs   string
	isDir bool
}

func collectFilesBFS(workingDir, query string, limit int) ([]Item, error) {
	absWorking, err := filepath.Abs(strings.TrimSpace(workingDir))
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absWorking)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("working directory is not a directory: %s", absWorking)
	}

	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	queue := []walkEntry{{absDir: absWorking, relDir: "."}}
	items := make([]Item, 0, limit)

	for len(queue) > 0 && len(items) < limit {
		cur := queue[0]
		queue = queue[1:]

		entries, err := os.ReadDir(cur.absDir)
		if err != nil {
			continue
		}

		candidates := make([]walkCandidate, 0, len(entries))
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") && name != "." {
				continue
			}
			rel := name
			if cur.relDir != "." {
				rel = filepath.ToSlash(filepath.Join(cur.relDir, name))
			}
			candidates = append(candidates, walkCandidate{
				entry: entry,
				rel:   rel,
				abs:   filepath.Join(cur.absDir, name),
				isDir: entry.IsDir(),
			})
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].isDir != candidates[j].isDir {
				return candidates[i].isDir
			}
			return strings.ToLower(candidates[i].rel) < strings.ToLower(candidates[j].rel)
		})

		for _, c := range candidates {
			if c.isDir {
				queue = append(queue, walkEntry{absDir: c.abs, relDir: c.rel})
			}
			if !matchesFileQuery(c.rel, c.entry.Name(), lowerQuery) {
				continue
			}
			if c.isDir {
				items = append(items, Item{
					Label:       c.rel + "/",
					Value:       dirValuePrefix + c.rel,
					HasChildren: false,
					Icon:        "folder",
				})
			} else {
				items = append(items, Item{
					Label:       c.rel,
					Value:       fileValuePrefix + c.rel,
					HasChildren: false,
					Icon:        "file",
				})
			}
			if len(items) >= limit {
				return items, nil
			}
		}
	}
	return items, nil
}

func matchesFileQuery(relPath, name, lowerQuery string) bool {
	if lowerQuery == "" {
		return true
	}
	if strings.Contains(strings.ToLower(name), lowerQuery) {
		return true
	}
	return strings.Contains(strings.ToLower(relPath), lowerQuery)
}

func resolveFilePath(workingDir, rel string) (string, error) {
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		return "", fmt.Errorf("working directory is required")
	}
	absWorking, err := filepath.Abs(workingDir)
	if err != nil {
		return "", err
	}
	rel = filepath.FromSlash(strings.TrimSpace(rel))
	if rel == "" || rel == "." {
		return absWorking, nil
	}
	if filepath.IsAbs(rel) {
		clean := filepath.Clean(rel)
		if !pathWithinRoot(absWorking, clean) {
			return "", fmt.Errorf("path is outside working directory")
		}
		return clean, nil
	}
	joined := filepath.Clean(filepath.Join(absWorking, rel))
	if !pathWithinRoot(absWorking, joined) {
		return "", fmt.Errorf("path is outside working directory")
	}
	return joined, nil
}

func pathWithinRoot(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if root == target {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
