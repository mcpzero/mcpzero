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
	Teams []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"teams"`
	ActiveTeamID string `json:"active_team_id"`
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

// DeleteEndpoint removes an endpoint the user owns.
func (c *Client) DeleteEndpoint(ctx context.Context, endpointID string) error {
	body := map[string]any{"id": endpointID}
	return c.doJSON(ctx, http.MethodDelete, "/app/api/cli/endpoints", body, nil)
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

// ActivityEntry is one usage_ledger row returned by the CLI activity API.
type ActivityEntry struct {
	ID         string  `json:"id"`
	EndpointID string  `json:"endpoint_id"`
	ToolName   *string `json:"tool_name"`
	ServerName *string `json:"server_name"`
	Status     string  `json:"status"`
	LatencyMs  *int    `json:"latency_ms"`
	ErrorCode  *string `json:"error_code"`
	ClientIP   *string `json:"client_ip"`
	CreatedAt  string  `json:"created_at"`
	TraceURL   string  `json:"trace_url"`
}

// ActivityListOptions filters GET /app/api/cli/activity.
type ActivityListOptions struct {
	Limit      int
	After      string // created_at cursor (exclusive)
	EndpointID string
	Status     string
	Search     string
}

// ListActivity returns recent Activity rows visible to the authenticated user.
func (c *Client) ListActivity(ctx context.Context, opts ActivityListOptions) ([]ActivityEntry, error) {
	u, err := url.Parse(c.WebBase + "/app/api/cli/activity")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if opts.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.After != "" {
		q.Set("after", opts.After)
	}
	if opts.EndpointID != "" {
		q.Set("endpoint", opts.EndpointID)
	}
	if opts.Status != "" {
		q.Set("status", opts.Status)
	}
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.RefreshToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /app/api/cli/activity: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAPIError(resp.StatusCode, string(body))
	}

	var parsed struct {
		Entries []ActivityEntry `json:"entries"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse activity response: %w", err)
	}
	return parsed.Entries, nil
}
