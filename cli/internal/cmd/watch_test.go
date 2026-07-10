package cmd

import (
	"testing"

	"github.com/mcpzero/mcpzero/cli/internal/cloud"
)

func TestReverseEntries(t *testing.T) {
	in := []cloud.ActivityEntry{
		{ID: "a", CreatedAt: "3"},
		{ID: "b", CreatedAt: "2"},
		{ID: "c", CreatedAt: "1"},
	}
	out := reverseEntries(in)
	if out[0].ID != "c" || out[2].ID != "a" {
		t.Fatalf("got %#v", out)
	}
}

func TestReverseEntriesEmpty(t *testing.T) {
	if len(reverseEntries(nil)) != 0 {
		t.Fatal("expected empty")
	}
}
