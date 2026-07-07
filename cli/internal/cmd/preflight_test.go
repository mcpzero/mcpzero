package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
)

func TestRunTunnelPreflight(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer gw.Close()

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	if err := auth.SaveCredentials(auth.Credentials{
		RefreshToken: "crt_test",
		UserID:       "u1",
		Email:        "a@b.c",
		GWBase:       gw.URL,
		WebBase:      "https://mcpzero.io",
	}); err != nil {
		t.Fatal(err)
	}

	err := runTunnelPreflight(context.Background(), tunnelPreflightInput{
		EndpointID: "ep_test",
		GWBase:     gw.URL,
		MgmtKey:    "test-mgmt-key",
	})
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
}

func TestRunTunnelPreflightCloud(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/api/cli/me":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"user_id": "u1", "email": "a@b.c", "plan": "free",
				"personal_endpoints": map[string]int{"used": 1, "limit": 1},
			})
		case "/app/api/cli/endpoints":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"endpoints": []map[string]string{{"id": "ep_test", "name": "demo"}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer web.Close()

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	if err := auth.SaveCredentials(auth.Credentials{
		RefreshToken: "crt_test",
		UserID:       "u1",
		Email:        "a@b.c",
		GWBase:       "https://gw.example.com",
		WebBase:      web.URL,
	}); err != nil {
		t.Fatal(err)
	}

	if err := runTunnelPreflightCloud(context.Background(), "ep_test"); err != nil {
		t.Fatalf("preflight cloud: %v", err)
	}
}

func TestRunTunnelPreflightCloudUnknownEndpoint(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/api/cli/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"plan": "free"})
		case "/app/api/cli/endpoints":
			_ = json.NewEncoder(w).Encode(map[string]any{"endpoints": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer web.Close()

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	if err := auth.SaveCredentials(auth.Credentials{
		RefreshToken: "crt_test",
		UserID:       "u1",
		Email:        "a@b.c",
		GWBase:       "https://gw.example.com",
		WebBase:      web.URL,
	}); err != nil {
		t.Fatal(err)
	}

	err := runTunnelPreflightCloud(context.Background(), "ep_missing")
	if err == nil {
		t.Fatal("expected error")
	}
}
