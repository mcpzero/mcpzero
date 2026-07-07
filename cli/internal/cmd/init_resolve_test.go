package cmd

import (
	"errors"
	"testing"

	"github.com/mcpzero/mcpzero/cli/internal/cloud"
)

func TestResolveInitEndpointReuseExisting(t *testing.T) {
	eps := []cloud.Endpoint{
		{ID: "ep_old", Name: "old", CreatedAt: "2026-01-01"},
		{ID: "ep_new", Name: "new", CreatedAt: "2026-07-01"},
	}

	res, err := resolveInitEndpoint(initResolveInput{Endpoints: eps})
	if err != nil {
		t.Fatal(err)
	}
	if res.CreateNew {
		t.Fatal("expected reuse")
	}
	if res.Endpoint.ID != "ep_new" {
		t.Fatalf("got %q, want ep_new (first in personal list)", res.Endpoint.ID)
	}
}

func TestResolveInitEndpointPreferPersonal(t *testing.T) {
	team := "team_1"
	eps := []cloud.Endpoint{
		{ID: "ep_team", Name: "team", TeamID: &team},
		{ID: "ep_personal", Name: "mine"},
	}

	res, err := resolveInitEndpoint(initResolveInput{Endpoints: eps})
	if err != nil {
		t.Fatal(err)
	}
	if res.Endpoint.ID != "ep_personal" {
		t.Fatalf("got %q, want ep_personal", res.Endpoint.ID)
	}
}

func TestResolveInitEndpointExplicitID(t *testing.T) {
	eps := []cloud.Endpoint{{ID: "ep_a", Name: "a"}, {ID: "ep_b", Name: "b"}}

	res, err := resolveInitEndpoint(initResolveInput{Endpoints: eps, EndpointID: "ep_b"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Endpoint.ID != "ep_b" {
		t.Fatalf("got %q", res.Endpoint.ID)
	}
}

func TestResolveInitEndpointUnknownID(t *testing.T) {
	_, err := resolveInitEndpoint(initResolveInput{
		Endpoints:  []cloud.Endpoint{{ID: "ep_a", Name: "a"}},
		EndpointID: "ep_missing",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveInitEndpointForceNew(t *testing.T) {
	eps := []cloud.Endpoint{{ID: "ep_a", Name: "a"}}

	res, err := resolveInitEndpoint(initResolveInput{Endpoints: eps, ForceNew: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.CreateNew {
		t.Fatal("expected create")
	}
}

func TestResolveInitEndpointEmptyCreates(t *testing.T) {
	res, err := resolveInitEndpoint(initResolveInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.CreateNew {
		t.Fatal("expected create when no endpoints")
	}
}

func TestFormatEndpointLimitError(t *testing.T) {
	base := &cloud.APIError{Code: "endpoint_limit_reached", Status: 403}
	err := formatEndpointLimitError(base, []cloud.Endpoint{{ID: "ep_free", Name: "only"}})
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if !errors.Is(err, base) {
		t.Fatal("expected errors.Is to match API error")
	}
	if got := err.Error(); !containsAll(got, "ep_free", "without --new") {
		t.Fatalf("error = %q", got)
	}
}
