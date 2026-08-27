// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"context"
	"fmt"
	"strings"

	"nui/internal/mentions"
	"nui/internal/store"
)

// MentionRPCClient talks to an extension mention provider over stdio JSON-RPC.
type MentionRPCClient struct {
	rpc          *StdioRPC
	programmatic *programmaticHost
	providerID   string
}

func NewMentionRPCClient(extDir, extName, providerID string, rt RuntimeConfig) (*MentionRPCClient, error) {
	if len(rt.Command) == 0 {
		return nil, fmt.Errorf("mention runtime command is required")
	}
	sdkDir, err := MentionSDKDir()
	if err != nil {
		return nil, fmt.Errorf("mention sdk: %w", err)
	}
	cmd := expandCommand(rt.Command, extDir)
	env := store.ExtensionProcessEnv(extName, map[string]string{
		"NUI_EXTENSION_DIR":        extDir,
		"NUI_EXTENSION_NAME":       extName,
		"NUI_MENTION_PROVIDER_ID":  providerID,
		"NUI_MENTION_SDK_DIR":      sdkDir,
	})
	rpc, err := StartStdioRPC(cmd, env, runtimeCwd(extDir, &rt))
	if err != nil {
		return nil, err
	}
	client := &MentionRPCClient{rpc: rpc, providerID: providerID}
	var info struct {
		ID string `json:"id"`
	}
	if err := client.rpc.Call("mention.info", map[string]any{}, &info); err != nil {
		_ = rpc.Close()
		return nil, err
	}
	return client, nil
}

func (c *MentionRPCClient) Close() error {
	if c.programmatic != nil {
		return nil
	}
	if c.rpc == nil {
		return nil
	}
	_ = c.rpc.Call("mention.shutdown", map[string]any{}, nil)
	return c.rpc.killProcess()
}

func (c *MentionRPCClient) List(ctx context.Context, req mentions.ListRequest, providerID string) (mentions.ListResponse, error) {
	if c.programmatic != nil {
		return c.programmatic.ListMentions(ctx, providerID, req)
	}
	_ = ctx
	var result struct {
		Items      []mentions.Item      `json:"items"`
		Breadcrumb []mentions.Breadcrumb `json:"breadcrumb"`
	}
	params := map[string]any{
		"providerId": providerID,
		"parent":     strings.TrimSpace(req.Parent),
		"query":      strings.TrimSpace(req.Query),
		"limit":      mentions.NormalizeLimit(req.Limit),
		"workingDir": req.WorkingDir,
		"sessionId":  req.SessionID,
	}
	if err := c.rpc.Call("mention.list", params, &result); err != nil {
		return mentions.ListResponse{}, err
	}
	if result.Items == nil {
		result.Items = []mentions.Item{}
	}
	return mentions.ListResponse{Items: result.Items, Breadcrumb: result.Breadcrumb}, nil
}

func (c *MentionRPCClient) Resolve(ctx context.Context, req mentions.ResolveRequest, providerID string) (string, error) {
	if c.programmatic != nil {
		return c.programmatic.ResolveMention(ctx, providerID, req)
	}
	_ = ctx
	var result struct {
		Text string `json:"text"`
	}
	params := map[string]any{
		"providerId": providerID,
		"value":      req.Value,
		"workingDir": req.WorkingDir,
		"sessionId":  req.SessionID,
	}
	if err := c.rpc.Call("mention.resolve", params, &result); err != nil {
		return "", err
	}
	return result.Text, nil
}
