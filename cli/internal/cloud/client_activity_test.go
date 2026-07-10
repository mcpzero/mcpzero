package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListActivity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/api/cli/activity" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer crt_test" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		q := r.URL.Query()
		if q.Get("endpoint") != "ep_x" || q.Get("after") != "2026-01-01" {
			http.Error(w, "bad query", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entries": []map[string]any{
				{
					"id": "tr_1", "endpoint_id": "ep_x", "status": "success",
					"created_at": "2026-01-02", "trace_url": "https://dev.mcpzero.io/app/activity/tr_1",
				},
			},
			"total": 1, "limit": 20,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "crt_test")
	entries, err := client.ListActivity(context.Background(), ActivityListOptions{
		EndpointID: "ep_x",
		After:      "2026-01-01",
		Limit:      20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "tr_1" {
		t.Fatalf("entries = %#v", entries)
	}
}
