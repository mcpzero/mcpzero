package planlimits

import "testing"

func TestRateLimitRPM(t *testing.T) {
	if RateLimitRPM("free") != 30 {
		t.Fatal("free")
	}
	if RateLimitRPM("team") != 60 {
		t.Fatal("team")
	}
}

func TestMaxEndpoints(t *testing.T) {
	cases := map[string]int{
		"free": 1, "personal": 2, "team": 10, "enterprise": -1,
	}
	for plan, want := range cases {
		if got := MaxEndpoints(plan); got != want {
			t.Fatalf("%s: got %d want %d", plan, got, want)
		}
	}
}

func TestMaxServersPerEndpoint(t *testing.T) {
	if MaxServersPerEndpoint("free") != 5 {
		t.Fatal("free")
	}
	if MaxServersPerEndpoint("team") != 10 {
		t.Fatal("team")
	}
	if MaxServersPerEndpoint("enterprise") != -1 {
		t.Fatal("enterprise")
	}
}

func TestToolsScope(t *testing.T) {
	if ToolsScope("free") != "account" {
		t.Fatal("free should be account-wide")
	}
	if ToolsScope("personal") != "tunnel" {
		t.Fatal("personal")
	}
}

func TestPayloadRetentionLabel(t *testing.T) {
	if PayloadRetentionLabel("free") != "48 hours" {
		t.Fatal(PayloadRetentionLabel("free"))
	}
	if PayloadRetentionLabel("personal") != "7 days" {
		t.Fatal(PayloadRetentionLabel("personal"))
	}
}
