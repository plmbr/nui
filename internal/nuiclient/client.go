// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package nuiclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:8080"

// Client talks to a running nui REST API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = os.Getenv("NUI_URL")
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 0, // streaming endpoints need no global timeout
		},
	}
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %s", resp.Status)
	}
	return nil
}

type AgentType struct {
	ID            string   `json:"id"`
	Label         string   `json:"label"`
	Description   string   `json:"description,omitempty"`
	Harness       string   `json:"harness"`
	PromptMode    string   `json:"promptMode,omitempty"`
	DefaultPrompt string   `json:"defaultPrompt,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Available     bool     `json:"available"`
	IsBuiltin     bool     `json:"isBuiltin"`
	Source        string   `json:"source,omitempty"` // builtin | user | extension
}

type Settings struct {
	DefaultAgentType   string   `json:"defaultAgentType,omitempty"`
	DefaultHarness     string   `json:"defaultHarness,omitempty"`
	Theme              string   `json:"theme,omitempty"`
	UITheme            string   `json:"uiTheme,omitempty"`
	DisabledExtensions []string `json:"disabledExtensions,omitempty"`
}

type Session struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	WorkingDir   string `json:"workingDir"`
	AgentType    string `json:"agentType"`
	CreatedAt    string `json:"createdAt"`
	ScheduleID   string `json:"scheduleId,omitempty"`
	ScheduleName string `json:"scheduleName,omitempty"`
	LastRunAt    string `json:"lastRunAt,omitempty"`
}

type Schedule struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AgentType     string `json:"agentType"`
	Prompt        string `json:"prompt,omitempty"`
	WorkingDir    string `json:"workingDir,omitempty"`
	Interval      string `json:"interval,omitempty"`
	Cron          string `json:"cron,omitempty"`
	RunAt         string `json:"runAt,omitempty"`
	Enabled       bool   `json:"enabled"`
	LastRunAt     string `json:"lastRunAt,omitempty"`
	NextRunAt     string `json:"nextRunAt,omitempty"`
	LastSessionID string `json:"lastSessionId,omitempty"`
	LastRunID     string `json:"lastRunId,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

type CreateScheduleRequest struct {
	Name       string `json:"name"`
	AgentType  string `json:"agentType"`
	Prompt     string `json:"prompt,omitempty"`
	WorkingDir string `json:"workingDir,omitempty"`
	Interval   string `json:"interval,omitempty"`
	Cron       string `json:"cron,omitempty"`
	RunAt      string `json:"runAt,omitempty"`
}

type PatchScheduleRequest struct {
	Name       *string `json:"name,omitempty"`
	Prompt     *string `json:"prompt,omitempty"`
	WorkingDir *string `json:"workingDir,omitempty"`
	Interval   *string `json:"interval,omitempty"`
	Cron       *string `json:"cron,omitempty"`
	RunAt      *string `json:"runAt,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
}

type RunRecord struct {
	RunID      string `json:"runId"`
	SessionID  string `json:"sessionId"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

func (c *Client) ListAgents(ctx context.Context) ([]AgentType, error) {
	var out []AgentType
	if err := c.getJSON(ctx, "/api/agent-types", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListOrchestratorAgents(ctx context.Context) ([]map[string]any, error) {
	var out []map[string]any
	if err := c.getJSON(ctx, "/api/orchestrator/routable-agents", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SearchOrchestratorAgents(ctx context.Context, query string, limit int) ([]map[string]any, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	path := "/api/orchestrator/search-agents?q=" + url.QueryEscape(query)
	if limit > 0 {
		path += "&limit=" + strconv.Itoa(limit)
	}
	var out []map[string]any
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func (c *Client) GetSettings(ctx context.Context) (Settings, error) {
	var out Settings
	if err := c.getJSON(ctx, "/api/settings", &out); err != nil {
		return Settings{}, err
	}
	return out, nil
}

func (c *Client) UpdateSettings(ctx context.Context, patch Settings) (Settings, error) {
	var out Settings
	if err := c.putJSON(ctx, "/api/settings", patch, http.StatusOK, &out); err != nil {
		return Settings{}, err
	}
	return out, nil
}

// SetDisabledExtensions replaces the disabled-extensions list (empty slice clears all).
func (c *Client) SetDisabledExtensions(ctx context.Context, names []string) (Settings, error) {
	if names == nil {
		names = []string{}
	}
	var out Settings
	if err := c.putJSON(ctx, "/api/settings", map[string]any{
		"disabledExtensions": names,
	}, http.StatusOK, &out); err != nil {
		return Settings{}, err
	}
	return out, nil
}

func (c *Client) GetVersion(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	if err := c.getJSON(ctx, "/api/version", &out); err != nil {
		return "", err
	}
	return out.Version, nil
}

type ExtensionInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	Description string   `json:"description,omitempty"`
	Disabled    bool     `json:"disabled"`
	Harnesses   []string `json:"harnesses,omitempty"`
	MCPServers  []string `json:"mcpServers,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Agents      []string `json:"agents,omitempty"`
}

func (c *Client) ListExtensions(ctx context.Context) ([]ExtensionInfo, error) {
	var out []ExtensionInfo
	if err := c.getJSON(ctx, "/api/extensions", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ExtensionInfo{}
	}
	return out, nil
}

func (c *Client) ReloadExtensions(ctx context.Context) error {
	return c.postJSON(ctx, "/api/extensions/reload", map[string]any{}, http.StatusOK, &map[string]any{})
}

type MCPServerInfo struct {
	Name    string `json:"name"`
	Command string `json:"command,omitempty"`
	URL     string `json:"url,omitempty"`
	Source  string `json:"source,omitempty"` // user | extension
}

func (c *Client) ListMCPServers(ctx context.Context) ([]MCPServerInfo, error) {
	var wrap struct {
		MCPServers []struct {
			Name    string `json:"name"`
			Command string `json:"command,omitempty"`
			URL     string `json:"url,omitempty"`
		} `json:"mcpServers"`
	}
	if err := c.getJSON(ctx, "/api/mcp-servers", &wrap); err != nil {
		return nil, err
	}
	var out []MCPServerInfo
	for _, s := range wrap.MCPServers {
		out = append(out, MCPServerInfo{
			Name:    s.Name,
			Command: s.Command,
			URL:     s.URL,
			Source:  "user",
		})
	}
	exts, err := c.ListExtensions(ctx)
	if err == nil {
		for _, e := range exts {
			if e.Disabled {
				continue
			}
			for _, name := range e.MCPServers {
				out = append(out, MCPServerInfo{Name: name, Source: "extension:" + e.Name})
			}
		}
	}
	if out == nil {
		out = []MCPServerInfo{}
	}
	return out, nil
}

// ResolveDefaultAgentType returns the configured default agent id, or the first
// available builtin, matching the UI pickDefaultAgentTypeId logic.
func (c *Client) ResolveDefaultAgentType(ctx context.Context) (string, error) {
	settings, err := c.GetSettings(ctx)
	if err != nil {
		return "", err
	}
	agents, err := c.ListAgents(ctx)
	if err != nil {
		return "", err
	}
	return pickDefaultAgentTypeID(agents, settings.DefaultAgentType)
}

func pickDefaultAgentTypeID(agents []AgentType, preferredID string) (string, error) {
	var selectable []AgentType
	for _, a := range agents {
		if a.Available {
			selectable = append(selectable, a)
		}
	}
	if len(selectable) == 0 {
		return "", fmt.Errorf("no available agent types")
	}
	if preferredID != "" {
		for _, a := range selectable {
			if a.ID == preferredID {
				return a.ID, nil
			}
		}
	}
	// Prefer installed CLI agents (claude-code first) over API builtins so a
	// keyless provider like Ollama is not chosen when Claude Code is available.
	for _, id := range []string{"claude-code", "pi", "codex", "opencode", "antigravity", "anthropic", "openai", "gemini", "openrouter", "ollama"} {
		for _, a := range selectable {
			if a.ID == id {
				return a.ID, nil
			}
		}
	}
	for _, a := range selectable {
		if a.IsBuiltin {
			return a.ID, nil
		}
	}
	return selectable[0].ID, nil
}

func (c *Client) ListSessions(ctx context.Context) ([]Session, error) {
	var out []Session
	if err := c.getJSON(ctx, "/api/sessions", &out); err != nil {
		return nil, err
	}
	return out, nil
}

type CreateSessionRequest struct {
	Name        string         `json:"name"`
	AgentType   string         `json:"agentType"`
	WorkingDir  string         `json:"workingDir,omitempty"`
	AgentConfig map[string]any `json:"agentConfig,omitempty"`
}

// AgentConfigHarnessOverride builds agentConfig with an optional harness.type override.
func AgentConfigHarnessOverride(harnessType string) map[string]any {
	harnessType = strings.TrimSpace(harnessType)
	if harnessType == "" {
		return nil
	}
	return map[string]any{"harnessType": harnessType}
}

func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (Session, error) {
	var out Session
	if err := c.postJSON(ctx, "/api/sessions", req, http.StatusCreated, &out); err != nil {
		return Session{}, err
	}
	return out, nil
}

func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	return c.deleteJSON(ctx, "/api/sessions/"+sessionID)
}

type LaunchRequest struct {
	AgentType  string `json:"agentType,omitempty"`
	WorkingDir string `json:"workingDir,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
	HideInput  bool   `json:"hideInput,omitempty"`
	Harness    string `json:"harness,omitempty"`
}

func (c *Client) Launch(ctx context.Context, req LaunchRequest) (Session, error) {
	var out Session
	if err := c.postJSON(ctx, "/api/launch", req, http.StatusCreated, &out); err != nil {
		return Session{}, err
	}
	return out, nil
}

type StartRunRequest struct {
	Message string `json:"message,omitempty"`
}

type StartRunResponse struct {
	RunID     string `json:"runId"`
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

func (c *Client) StartRun(ctx context.Context, sessionID string, req StartRunRequest) (StartRunResponse, error) {
	var out StartRunResponse
	if err := c.postJSON(ctx, "/api/sessions/"+sessionID+"/runs", req, http.StatusAccepted, &out); err != nil {
		return StartRunResponse{}, err
	}
	return out, nil
}

func (c *Client) GetRun(ctx context.Context, sessionID, runID string) (RunRecord, error) {
	var out RunRecord
	if err := c.getJSON(ctx, "/api/sessions/"+sessionID+"/runs/"+runID, &out); err != nil {
		return RunRecord{}, err
	}
	return out, nil
}

func (c *Client) StopRun(ctx context.Context, sessionID string, runID string) error {
	url := c.BaseURL + "/api/sessions/" + sessionID + "/stop"
	if runID != "" {
		url += "?runId=" + runID
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) WaitRun(ctx context.Context, sessionID, runID string, pollInterval time.Duration) (RunRecord, error) {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	for {
		rec, err := c.GetRun(ctx, sessionID, runID)
		if err != nil {
			return RunRecord{}, err
		}
		switch rec.Status {
		case "completed", "failed", "cancelled":
			return rec, nil
		}
		select {
		case <-ctx.Done():
			return rec, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// StreamRunEvents tails the run event SSE endpoint until the run finishes.
func (c *Client) StreamRunEvents(ctx context.Context, sessionID, runID string, lastEventID string, onEvent func(data []byte)) (RunRecord, error) {
	url := c.BaseURL + "/api/sessions/" + sessionID + "/runs/" + runID + "/events"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return RunRecord{}, err
	}
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return RunRecord{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return RunRecord{}, fmt.Errorf("events stream failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	var dataLine []byte
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			if len(dataLine) > 0 {
				if onEvent != nil {
					onEvent(dataLine)
				}
				var meta struct {
					Type string `json:"type"`
				}
				_ = json.Unmarshal(dataLine, &meta)
				if meta.Type == "run_finished" {
					return c.GetRun(ctx, sessionID, runID)
				}
			}
			dataLine = nil
			continue
		}
		if bytes.HasPrefix(line, []byte("data: ")) {
			dataLine = bytes.TrimPrefix(line, []byte("data: "))
		}
	}
	if err := sc.Err(); err != nil {
		return RunRecord{}, err
	}
	return c.GetRun(ctx, sessionID, runID)
}

func (c *Client) ListSchedules(ctx context.Context) ([]Schedule, error) {
	var out []Schedule
	if err := c.getJSON(ctx, "/api/schedules", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateSchedule(ctx context.Context, req CreateScheduleRequest) (Schedule, error) {
	var out Schedule
	if err := c.postJSON(ctx, "/api/schedules", req, http.StatusCreated, &out); err != nil {
		return Schedule{}, err
	}
	return out, nil
}

func (c *Client) PatchSchedule(ctx context.Context, id string, req PatchScheduleRequest) (Schedule, error) {
	var out Schedule
	if err := c.patchJSON(ctx, "/api/schedules/"+id, req, http.StatusOK, &out); err != nil {
		return Schedule{}, err
	}
	return out, nil
}

func (c *Client) DeleteSchedule(ctx context.Context, id string) error {
	return c.deleteJSON(ctx, "/api/schedules/"+id)
}

func (c *Client) RunScheduleNow(ctx context.Context, id string) (Schedule, error) {
	var out Schedule
	if err := c.postJSON(ctx, "/api/schedules/"+id+"/run-now", nil, http.StatusAccepted, &out); err != nil {
		return Schedule{}, err
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s failed: %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) putJSON(ctx context.Context, path string, body any, expectStatus int, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectStatus {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT %s failed: %s: %s", path, resp.Status, strings.TrimSpace(string(raw)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) postJSON(ctx context.Context, path string, body any, expectStatus int, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectStatus {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s failed: %s: %s", path, resp.Status, strings.TrimSpace(string(raw)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) patchJSON(ctx context.Context, path string, body any, expectStatus int, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectStatus {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PATCH %s failed: %s: %s", path, resp.Status, strings.TrimSpace(string(raw)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) deleteJSON(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DELETE %s failed: %s: %s", path, resp.Status, strings.TrimSpace(string(raw)))
	}
	return nil
}

type AgentDeployerInfo struct {
	ID          string `json:"id"`
	Extension   string `json:"extension"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type DeployEndpoint struct {
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`
	URL  string `json:"url,omitempty"`
}

type AgentDeployResult struct {
	DeploymentID string          `json:"deploymentId,omitempty"`
	Status       string          `json:"status,omitempty"`
	Message      string          `json:"message,omitempty"`
	Endpoint     *DeployEndpoint `json:"endpoint,omitempty"`
}

func (c *Client) ListAgentDeployers(ctx context.Context) ([]AgentDeployerInfo, error) {
	var wrap struct {
		Deployers []AgentDeployerInfo `json:"deployers"`
	}
	if err := c.getJSON(ctx, "/api/agent-deployers", &wrap); err != nil {
		return nil, err
	}
	if wrap.Deployers == nil {
		return []AgentDeployerInfo{}, nil
	}
	return wrap.Deployers, nil
}

func (c *Client) DeployAgent(ctx context.Context, agentID, deployerID string) (AgentDeployResult, error) {
	var result AgentDeployResult
	path := "/api/agents/" + url.PathEscape(agentID) + "/deploy"
	err := c.postJSON(ctx, path, map[string]string{
		"deployerId": deployerID,
	}, http.StatusOK, &result)
	return result, err
}
