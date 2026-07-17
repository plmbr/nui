// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"nui/internal/model"
)

// ErrClientRegistrationRequired indicates the server does not support dynamic client registration.
var ErrClientRegistrationRequired = fmt.Errorf("this MCP server requires a pre-registered OAuth client ID")

func validateOAuthStart(ctx context.Context, srv model.ADLMCPServer) error {
	probe, err := ProbeServer(ctx, srv.URL)
	if err != nil {
		return err
	}
	defer drainResponse(probe.Response)
	if !NeedsAuth(probe.Response) {
		return fmt.Errorf("server did not request authentication (status %d)", probe.Response.StatusCode)
	}

	client := &http.Client{Timeout: probeTimeout}
	wwwChallenges, err := oauthex.ParseWWWAuthenticate(probe.Response.Header[http.CanonicalHeaderKey("WWW-Authenticate")])
	if err != nil {
		return fmt.Errorf("parse WWW-Authenticate: %w", err)
	}
	resourceURL := strings.TrimSpace(srv.URL)
	prm, err := fetchProtectedResourceMetadata(ctx, wwwChallenges, resourceURL, client)
	if err != nil {
		return err
	}
	if len(prm.AuthorizationServers) == 0 {
		return fmt.Errorf("%w: no authorization server found in protected resource metadata", ErrClientRegistrationRequired)
	}
	asm, err := auth.GetAuthServerMetadata(ctx, prm.AuthorizationServers[0], client)
	if err != nil {
		return fmt.Errorf("fetch authorization server metadata: %w", err)
	}
	if asm != nil && strings.TrimSpace(asm.RegistrationEndpoint) != "" {
		return nil
	}

	clientID := ""
	clientSecret := ""
	if srv.Auth != nil {
		clientID = strings.TrimSpace(srv.Auth.ClientID)
		clientSecret = strings.TrimSpace(srv.Auth.ClientSecret)
	}
	if clientID == "" {
		return fmt.Errorf("%w: register an OAuth app with your provider, add its Client ID under OAuth client ID, save, then Connect again. GitHub Copilot MCP does not support automatic client registration", ErrClientRegistrationRequired)
	}
	if clientSecret == "" {
		return fmt.Errorf("%w: this provider requires an OAuth client secret (e.g. GitHub). Add it under OAuth client secret, save, then Connect again", ErrClientRegistrationRequired)
	}
	return nil
}

func fetchProtectedResourceMetadata(ctx context.Context, challenges []oauthex.Challenge, resourceURL string, client *http.Client) (*oauthex.ProtectedResourceMetadata, error) {
	for _, metaURL := range protectedResourceMetadataURLs(challenges, resourceURL) {
		prm, err := oauthex.GetProtectedResourceMetadata(ctx, metaURL.URL, metaURL.Resource, client)
		if err != nil || prm == nil {
			continue
		}
		return prm, nil
	}
	u, err := url.Parse(resourceURL)
	if err != nil {
		return nil, fmt.Errorf("parse server URL: %w", err)
	}
	u.Path = ""
	return &oauthex.ProtectedResourceMetadata{
		AuthorizationServers: []string{u.String()},
		Resource:             resourceURL,
	}, nil
}

type prmLookupURL struct {
	URL      string
	Resource string
}

func protectedResourceMetadataURLs(challenges []oauthex.Challenge, resourceURL string) []prmLookupURL {
	var urls []prmLookupURL
	for _, c := range challenges {
		if u := c.Params["resource_metadata"]; u != "" {
			urls = append(urls, prmLookupURL{URL: u, Resource: resourceURL})
		}
	}
	ru, err := url.Parse(resourceURL)
	if err != nil {
		return urls
	}
	mu := *ru
	mu.Path = "/.well-known/oauth-protected-resource/" + strings.TrimLeft(ru.Path, "/")
	urls = append(urls, prmLookupURL{URL: mu.String(), Resource: resourceURL})
	mu.Path = "/.well-known/oauth-protected-resource"
	ru.Path = ""
	urls = append(urls, prmLookupURL{URL: mu.String(), Resource: ru.String()})
	return urls
}
