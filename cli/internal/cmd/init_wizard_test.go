package cmd

import (
	"testing"

	"github.com/mcpzero/mcpzero/cli/internal/cloud"
)

func TestParseEndpointMenuChoice(t *testing.T) {
	eps := []cloud.Endpoint{
		{ID: "ep_a", Name: "alpha"},
		{ID: "ep_b", Name: "beta"},
	}

	got, err := parseEndpointMenuChoice("", eps, true)
	if err != nil || got.EndpointID != "ep_a" {
		t.Fatalf("default: %#v err=%v", got, err)
	}

	got, err = parseEndpointMenuChoice("2", eps, true)
	if err != nil || got.EndpointID != "ep_b" {
		t.Fatalf("pick 2: %#v err=%v", got, err)
	}

	got, err = parseEndpointMenuChoice("3", eps, true)
	if err != nil || !got.CreateNew {
		t.Fatalf("create: %#v err=%v", got, err)
	}

	_, err = parseEndpointMenuChoice("9", eps, true)
	if err == nil {
		t.Fatal("expected error for out of range")
	}
}

func TestParseAPIKeyMenuChoice(t *testing.T) {
	got, err := parseAPIKeyMenuChoice("")
	if err != nil || !got.CreateNew {
		t.Fatalf("default create: %#v", got)
	}
	got, err = parseAPIKeyMenuChoice("paste")
	if err != nil || !got.Paste {
		t.Fatalf("paste: %#v", got)
	}
}

func TestKeysForEndpoint(t *testing.T) {
	keys := []cloud.APIKey{
		{ID: "k1", Hint: "a", Status: "active", EndpointIDs: []string{"ep_x"}},
		{ID: "k2", Hint: "b", Status: "revoked", EndpointIDs: []string{"ep_x"}},
		{ID: "k3", Hint: "c", Status: "active", EndpointIDs: []string{"ep_y"}},
		{ID: "k4", Hint: "d", Status: "active", EndpointIDs: nil},
	}
	scoped := keysForEndpoint(keys, "ep_x")
	if len(scoped) != 2 || scoped[0].ID != "k1" || scoped[1].ID != "k4" {
		t.Fatalf("scoped = %#v", scoped)
	}
}

func TestPersonalEndpointQuota(t *testing.T) {
	me := &cloud.AccountMe{}
	me.PersonalEndpoints.Used = 1
	me.PersonalEndpoints.Limit = 1
	used, limit, canCreate := personalEndpointQuota(me)
	if used != 1 || limit != 1 || canCreate {
		t.Fatalf("used=%d limit=%d canCreate=%v", used, limit, canCreate)
	}
}

func TestParseCursorMenuChoice(t *testing.T) {
	got, err := parseCursorMenuChoice("")
	if err != nil || !got.Write || got.Project {
		t.Fatalf("default global: %#v err=%v", got, err)
	}
	got, err = parseCursorMenuChoice("0")
	if err != nil || got.Write {
		t.Fatalf("skip: %#v err=%v", got, err)
	}
	got, err = parseCursorMenuChoice("2")
	if err != nil || !got.Write || !got.Project {
		t.Fatalf("project: %#v err=%v", got, err)
	}
	if _, err := parseCursorMenuChoice("9"); err == nil {
		t.Fatal("expected error for invalid choice")
	}
}

func TestParseMenuChoice(t *testing.T) {
	if n, err := parseMenuChoice("", 3); err != nil || n != 1 {
		t.Fatalf("default: n=%d err=%v", n, err)
	}
	if n, err := parseMenuChoice("2", 3); err != nil || n != 2 {
		t.Fatalf("2: n=%d err=%v", n, err)
	}
	if _, err := parseMenuChoice("0", 3); err == nil {
		t.Fatal("expected error")
	}
}
