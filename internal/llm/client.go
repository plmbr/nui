// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Context is an alias for context.Context used in Provider methods.
type Context = context.Context

// httpClient has no overall timeout so SSE streams can run indefinitely,
// but it times out on response headers to catch dead servers quickly.
var httpClient = &http.Client{
	Transport: &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

func postJSON(ctx context.Context, url string, headers map[string]string, body any) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return httpClient.Do(req)
}

func getJSON(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return httpClient.Do(req)
}

func readErrorBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return strings.TrimSpace(string(b))
}

func checkHTTPError(provider string, resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}
	body := readErrorBody(resp)
	if err := classifyHTTPError(provider, resp.StatusCode, body); err != nil {
		return err
	}
	return fmt.Errorf("[%s] unexpected status %d: %s", provider, resp.StatusCode, body)
}
