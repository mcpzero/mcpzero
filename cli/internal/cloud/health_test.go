package cloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingGateway(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	if err := PingGateway(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}

	health, err := GatewayHealth(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "ok" {
		t.Fatalf("status = %q", health.Status)
	}
}

func TestPingGatewayFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if err := PingGateway(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error")
	}
}
