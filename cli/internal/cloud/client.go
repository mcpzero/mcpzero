package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client calls MCPZERO web APIs authenticated with a CLI refresh token.
type Client struct {
	WebBase      string
	RefreshToken string
	HTTPClient   *http.Client
}

// Endpoint is a remote MCP endpoint owned by the user.
type Endpoint struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Status    string  `json:"status,omitempty"`
	TeamID    *string `json:"team_id"`
	TeamName  *string `json:"team_name"`
	CreatedAt string  `json:"created_at,omitempty"`
}

// APIKeyCreateResult is returned once when a new API key is created.
type APIKeyCreateResult struct {
	ID     string `json:"id"`
	RawKey string `json:"raw_key"`
	Hint   string `json:"hint"`
}

// APIKey is a stored API key (hint only — raw key is not retrievable).
type APIKey struct {
	ID          string   `json:"id"`
	Hint        string   `json:"hint"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
	EndpointIDs []string `json:"endpoint_ids"`
}

// AccountMe describes the authenticated user and personal endpoint quota.
type AccountMe struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Plan   string `json:"plan"`
	PersonalEndpoints struct {
		Used  int `json:"used"`
		Limit int `json:"limit"`
	} `json:"personal_endpoints"`
}

type apiError struct {
	Error string `json:"error"`
}

// NewClient builds a web API client using the saved CLI refresh token.
func NewClient(webBase, refreshToken string) *Client {
	return &Client{
		WebBase:      strings.TrimRight(webBase, "/"),
		RefreshToken: refreshToken,
		HTTPClient:   http.DefaultClient,
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	u, err := url.Parse(c.WebBase)
	if err != nil {
		return fmt.Errorf("parse web base: %w", err)
	}
	u.Path = path

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.RefreshToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp.StatusCode, string(respBody))
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("parse %s response: %w", path, err)
	}
	return nil
}

// ListEndpoints returns endpoints owned by the authenticated user.
func (c *Client) ListEndpoints(ctx context.Context) ([]Endpoint, error) {
	var resp struct {
		Endpoints []Endpoint `json:"endpoints"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/app/api/cli/endpoints", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Endpoints, nil
}

// CreateEndpoint creates a personal or team-scoped endpoint.
func (c *Client) CreateEndpoint(ctx context.Context, name string, teamID *string) (*Endpoint, error) {
	body := map[string]any{"name": name}
	if teamID != nil && *teamID != "" {
		body["team_id"] = *teamID
	}
	var ep Endpoint
	if err := c.doJSON(ctx, http.MethodPost, "/app/api/cli/endpoints", body, &ep); err != nil {
		return nil, err
	}
	return &ep, nil
}

// CreateAPIKey creates an API key scoped to the given endpoint IDs.
func (c *Client) CreateAPIKey(ctx context.Context, endpointIDs []string) (*APIKeyCreateResult, error) {
	body := map[string]any{"endpoint_ids": endpointIDs}
	var result APIKeyCreateResult
	if err := c.doJSON(ctx, http.MethodPost, "/app/api/cli/api-keys", body, &result); err != nil {
		return nil, err
	}
	if result.RawKey == "" {
		return nil, fmt.Errorf("api key response missing raw_key")
	}
	return &result, nil
}

// Me returns account info and plan limits for the CLI user.
func (c *Client) Me(ctx context.Context) (*AccountMe, error) {
	var me AccountMe
	if err := c.doJSON(ctx, http.MethodGet, "/app/api/cli/me", nil, &me); err != nil {
		return nil, err
	}
	return &me, nil
}

// ListAPIKeys returns API keys owned by the user (hints only).
func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	var resp struct {
		Keys []APIKey `json:"keys"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/app/api/cli/api-keys", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Keys, nil
}
