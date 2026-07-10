package cmd

import (
	"strings"
	"testing"
)

func TestEndpointSubcommandRouting(t *testing.T) {
	err := runEndpoint(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error, got %v", err)
	}
	err = runEndpoint([]string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "unknown endpoint") {
		t.Fatalf("expected unknown subcommand, got %v", err)
	}
}

func TestEndpointCreateRequiresName(t *testing.T) {
	err := endpointCreate(nil)
	if err == nil || !strings.Contains(err.Error(), "--name") {
		t.Fatalf("expected --name required, got %v", err)
	}
}

func TestEndpointRmRequiresID(t *testing.T) {
	err := endpointRm([]string{"-y"})
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error, got %v", err)
	}
}
