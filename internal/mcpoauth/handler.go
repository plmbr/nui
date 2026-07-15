// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
	"loop/internal/model"
)

// ErrNeedsAuth indicates the user must complete OAuth via the Loop UI.
var ErrNeedsAuth = errors.New("mcp server needs authentication")

// loopOAuthHandler provides stored tokens for MCP HTTP transports.
type loopOAuthHandler struct {
	serverKey string
	srv       model.ADLMCPServer
}

func (h *loopOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	cred, ok := LoadToken(h.serverKey)
	if !ok || cred.Token == nil || strings.TrimSpace(cred.Token.AccessToken) == "" {
		return nil, nil
	}
	return oauth2.StaticTokenSource(cred.Token), nil
}

func (h *loopOAuthHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	drainResponse(resp)
	return fmt.Errorf("%w: connect %q in Customize → MCP Servers", ErrNeedsAuth, h.serverKey)
}

// HandlerForServer returns an OAuth handler when the server uses Loop-managed OAuth tokens.
func HandlerForServer(srv model.ADLMCPServer) auth.OAuthHandler {
	if !IsRemote(srv) || IsBuiltin(srv) || hasStaticAuthHeader(srv) {
		return nil
	}
	key := ServerKey(srv)
	if !HasValidToken(key) && srv.Auth == nil {
		return nil
	}
	if HasValidToken(key) || srv.Auth != nil {
		return &loopOAuthHandler{serverKey: key, srv: srv}
	}
	return nil
}

// HTTPClientForServer returns an HTTP client with static headers applied.
func HTTPClientForServer(srv model.ADLMCPServer) *http.Client {
	if len(srv.Headers) == 0 {
		return nil
	}
	headers := map[string]string{}
	for k, v := range srv.Headers {
		headers[k] = v
	}
	return &http.Client{
		Transport: &staticHeaderTransport{
			base:    http.DefaultTransport,
			headers: headers,
		},
	}
}

type staticHeaderTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *staticHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	cloned := req.Clone(req.Context())
	for k, v := range t.headers {
		cloned.Header.Set(k, v)
	}
	return base.RoundTrip(cloned)
}

// ConnectRemote creates an MCP client session for a remote server with OAuth/static auth.
func ConnectRemote(ctx context.Context, srv model.ADLMCPServer) (*mcp.ClientSession, error) {
	url := strings.TrimSpace(srv.URL)
	if url == "" {
		return nil, fmt.Errorf("mcp server %q: url is required", srv.Name)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "loop", Version: "1.0.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: HTTPClientForServer(srv),
		OAuthHandler: HandlerForServer(srv),
	}
	return client.Connect(ctx, transport, nil)
}

func buildHandlerConfig(srv model.ADLMCPServer, redirect string, fetcher auth.AuthorizationCodeFetcher) (*auth.AuthorizationCodeHandlerConfig, error) {
	cfg := &auth.AuthorizationCodeHandlerConfig{
		RedirectURL:              redirect,
		AuthorizationCodeFetcher: fetcher,
		Client:                   &http.Client{Timeout: probeTimeout},
	}
	if srv.Auth != nil {
		if id := strings.TrimSpace(srv.Auth.ClientID); id != "" {
			creds := &oauthex.ClientCredentials{ClientID: id}
			if secret := strings.TrimSpace(srv.Auth.ClientSecret); secret != "" {
				creds.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: secret}
			}
			cfg.PreregisteredClient = creds
		}
	}
	if cfg.PreregisteredClient == nil {
		cfg.DynamicClientRegistrationConfig = &auth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				ClientName:              "Loop",
				RedirectURIs:            []string{redirect},
				GrantTypes:              []string{"authorization_code", "refresh_token"},
				ResponseTypes:           []string{"code"},
				TokenEndpointAuthMethod: "none",
			},
		}
	}
	return cfg, nil
}

func fetcherForFlow(flow *pendingFlow) auth.AuthorizationCodeFetcher {
	return func(ctx context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
		if state := ParseStateFromAuthURL(args.URL); state != "" {
			BindFlowState(state, flow.ID)
		}
		select {
		case flow.authURLCh <- args.URL:
		default:
		}
		select {
		case res := <-flow.resultCh:
			if res == nil {
				return nil, fmt.Errorf("authorization cancelled")
			}
			return res, nil
		case err := <-flow.errCh:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-flow.done:
			return nil, fmt.Errorf("authorization flow closed")
		}
	}
}

// StartFlow begins an interactive OAuth flow for a remote MCP server.
func StartFlow(ctx context.Context, srv model.ADLMCPServer) (flowID, authURL, redirectURI string, err error) {
	if !IsRemote(srv) {
		return "", "", "", fmt.Errorf("oauth applies only to remote MCP servers")
	}
	if err := validateOAuthStart(ctx, srv); err != nil {
		return "", "", "", err
	}
	redirect, err := RedirectURI()
	if err != nil {
		return "", "", "", err
	}
	flow := newPendingFlow(srv, redirect)
	registerFlow(flow)

	handlerCfg, err := buildHandlerConfig(srv, redirect, fetcherForFlow(flow))
	if err != nil {
		return "", "", "", err
	}
	handler, err := auth.NewAuthorizationCodeHandler(handlerCfg)
	if err != nil {
		return "", "", "", err
	}

	// The HTTP request context is cancelled when /start returns; keep the flow alive
	// until the browser callback delivers the authorization code.
	flowCtx := context.WithoutCancel(ctx)

	go func() {
		defer removeFlow(flow.ID)
		defer close(flow.done)

		fail := func(err error) {
			if err != nil {
				setFlowOutcome(flow.ID, FlowStatusFailed, flow.ServerKey, err.Error())
				select {
				case flow.errCh <- err:
				default:
				}
			}
		}

		probe, probeErr := ProbeServer(flowCtx, srv.URL)
		if probeErr != nil {
			fail(probeErr)
			return
		}
		if !NeedsAuth(probe.Response) {
			drainResponse(probe.Response)
			fail(fmt.Errorf("server did not request authentication (status %d)", probe.Response.StatusCode))
			return
		}
		authErr := handler.Authorize(flowCtx, probe.Request, probe.Response)
		if authErr != nil {
			fail(authErr)
			return
		}
		ts, tsErr := handler.TokenSource(flowCtx)
		if tsErr != nil {
			fail(tsErr)
			return
		}
		tok, tokErr := ts.Token()
		if tokErr != nil {
			fail(tokErr)
			return
		}
		saveErr := SaveToken(flow.ServerKey, storedCredential{
			Token:      tok,
			ServerURL:  strings.TrimSpace(srv.URL),
			ServerName: strings.TrimSpace(srv.Name),
		})
		if saveErr != nil {
			fail(saveErr)
			return
		}
		setFlowOutcome(flow.ID, FlowStatusCompleted, flow.ServerKey, "")
	}()

	select {
	case authURL = <-flow.authURLCh:
		if state := ParseStateFromAuthURL(authURL); state != "" {
			BindFlowState(state, flow.ID)
		}
		return flow.ID, authURL, redirect, nil
	case err = <-flow.errCh:
		return "", "", "", err
	case <-ctx.Done():
		return "", "", "", ctx.Err()
	}
}

// DeliverCallback completes a flow with an authorization code from the OAuth redirect.
func DeliverCallback(code, state string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		return fmt.Errorf("code and state are required")
	}
	flow, ok := flowByState(state)
	if !ok {
		return fmt.Errorf("unknown or expired oauth state")
	}
	select {
	case flow.resultCh <- &auth.AuthorizationResult{Code: code, State: state}:
		return nil
	default:
		return fmt.Errorf("oauth flow already completed")
	}
}

// DeliverCallbackURL parses a pasted redirect URL and completes the flow.
func DeliverCallbackURL(flowID, callbackURL string) error {
	flow, ok := flowByID(flowID)
	if !ok {
		return fmt.Errorf("unknown or expired oauth flow")
	}
	_ = flow
	u, err := url.Parse(strings.TrimSpace(callbackURL))
	if err != nil {
		return fmt.Errorf("invalid callback URL: %w", err)
	}
	q := u.Query()
	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		return fmt.Errorf("callback URL must include code and state query parameters")
	}
	bindFlowState(state, flowID)
	return DeliverCallback(code, state)
}

// BindFlowState records the OAuth state parameter for callback routing.
func BindFlowState(state, flowID string) {
	bindFlowState(state, flowID)
}

// ParseStateFromAuthURL extracts the state query parameter from an authorization URL.
func ParseStateFromAuthURL(authURL string) string {
	u, err := url.Parse(authURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("state")
}
