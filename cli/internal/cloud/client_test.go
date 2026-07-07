package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCreateEndpointAndAPIKey(t *testing.T) {
	const refresh = "crt_test_token"
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/app/api/cli/endpoints":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["name"] != "demo" {
				t.Fatalf("name = %q", body["name"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   "ep_test123",
				"name": "demo",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/app/api/cli/api-keys":
			var body struct {
				EndpointIDs []string `json:"endpoint_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(body.EndpointIDs) != 1 || body.EndpointIDs[0] != "ep_test123" {
				t.Fatalf("endpoint_ids = %#v", body.EndpointIDs)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":      "key_test",
				"raw_key": "mz_live_secret",
				"hint":    "mz_live_secr...cret",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/app/api/cli/endpoints":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"endpoints": []map[string]string{{"id": "ep_test123", "name": "demo"}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, refresh)
	ctx := context.Background()

	eps, err := client.ListEndpoints(ctx)
	if err != nil {
		t.Fatalf("ListEndpoints: %v", err)
	}
	if len(eps) != 1 || eps[0].ID != "ep_test123" {
		t.Fatalf("endpoints = %#v", eps)
	}

	ep, err := client.CreateEndpoint(ctx, "demo", nil)
	if err != nil {
		t.Fatalf("CreateEndpoint: %v", err)
	}
	if ep.ID != "ep_test123" {
		t.Fatalf("id = %q", ep.ID)
	}

	key, err := client.CreateAPIKey(ctx, []string{ep.ID})
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if key.RawKey != "mz_live_secret" {
		t.Fatalf("raw_key = %q", key.RawKey)
	}

	if gotAuth != "Bearer "+refresh {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestClientAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"endpoint_limit_reached"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "crt_x")
	_, err := client.CreateEndpoint(context.Background(), "demo", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsAPIError(err, "endpoint_limit_reached") {
		t.Fatalf("error = %v", err)
	}
}
