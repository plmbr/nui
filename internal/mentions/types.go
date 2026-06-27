// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mentions

import "context"

const MaxItems = 20
const MaxFileItems = 100

// Item is one entry in the mention autocomplete menu.
type Item struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	HasChildren bool   `json:"hasChildren"`
	Icon        string `json:"icon,omitempty"`
}

// Breadcrumb is one level in mention menu navigation.
type Breadcrumb struct {
	Label  string `json:"label"`
	Parent string `json:"parent"`
}

// ListRequest is passed to mention providers for lazy listing.
type ListRequest struct {
	SessionID  string
	WorkingDir string
	Parent     string
	Query      string
	Limit      int
}

// ListResponse is returned from mention list operations.
type ListResponse struct {
	Items      []Item
	Breadcrumb []Breadcrumb
}

// ResolveRequest is passed to mention providers to expand a token value.
type ResolveRequest struct {
	SessionID  string
	WorkingDir string
	Value      string
}

// Provider lists and resolves mention items for one source.
type Provider interface {
	ID() string
	DisplayName() string
	List(ctx context.Context, req ListRequest) (ListResponse, error)
	Resolve(ctx context.Context, req ResolveRequest) (string, error)
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > MaxItems {
		return MaxItems
	}
	return limit
}

// NormalizeLimit caps mention list size at MaxItems.
func NormalizeLimit(limit int) int {
	return normalizeLimit(limit)
}

func normalizeFileLimit(limit int) int {
	if limit <= 0 || limit > MaxFileItems {
		return MaxFileItems
	}
	return limit
}
