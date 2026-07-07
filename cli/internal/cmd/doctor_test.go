package cmd

import (
	"testing"

	"github.com/mcpzero/mcpzero/cli/internal/cloud"
)

func TestValidateEndpointID(t *testing.T) {
	if err := validateEndpointID("ep_abc"); err != nil {
		t.Fatal(err)
	}
	if err := validateEndpointID("epc_team"); err != nil {
		t.Fatal(err)
	}
	if err := validateEndpointID(""); err == nil {
		t.Fatal("expected error")
	}
	if err := validateEndpointID("bad"); err == nil {
		t.Fatal("expected error")
	}
}

func TestEndpointOwned(t *testing.T) {
	eps := []cloud.Endpoint{{ID: "ep_a"}, {ID: "ep_b"}}
	if !endpointOwned(eps, "ep_b") {
		t.Fatal("expected owned")
	}
	if endpointOwned(eps, "ep_z") {
		t.Fatal("expected not owned")
	}
}
