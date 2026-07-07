package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PingGET checks that baseURL+path returns HTTP 2xx.
func PingGET(ctx context.Context, client *http.Client, baseURL, path string) error {
	if client == nil {
		client = http.DefaultClient
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	u.Path = path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// PingGateway checks the MCPZERO gateway /health endpoint.
func PingGateway(ctx context.Context, gwBase string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return PingGET(ctx, nil, gwBase, "/health")
}

// PingWeb checks the dashboard origin responds.
func PingWeb(ctx context.Context, webBase string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return PingGET(ctx, nil, webBase, "/")
}

type gatewayHealth struct {
	Status string `json:"status"`
}

// GatewayHealth fetches gateway /health JSON when available.
func GatewayHealth(ctx context.Context, gwBase string) (*gatewayHealth, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	u, err := url.Parse(strings.TrimRight(gwBase, "/"))
	if err != nil {
		return nil, err
	}
	u.Path = "/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var body gatewayHealth
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return &body, nil
}
