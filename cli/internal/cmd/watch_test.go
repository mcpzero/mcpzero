package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mcpzero/mcpzero/cli/internal/cloud"
)

func ptr[T any](v T) *T { return &v }

func TestFormatActivityExtras(t *testing.T) {
	cases := []struct {
		name string
		code *string
		want string
	}{
		{name: "none", code: nil, want: ""},
		{name: "rate limit", code: ptr("rate_limited"), want: "429 rate_limited"},
		{name: "search mode", code: ptr("search_mode:keyword"), want: "search_mode=keyword"},
		{name: "generic", code: ptr("timeout"), want: "error_code=timeout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatActivityExtras(cloud.ActivityEntry{ErrorCode: tc.code})
			if tc.want == "" {
				if got != "" {
					t.Fatalf("got %q want empty", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("got %q want contain %q", got, tc.want)
			}
		})
	}
}

func TestFormatActivityLineJSONRoundTrip(t *testing.T) {
	e := cloud.ActivityEntry{
		ID:         "tr_1",
		EndpointID: "ep_a",
		ToolName:   ptr("meta_search"),
		Status:     "success",
		ErrorCode:  ptr("search_mode:llm"),
		CreatedAt:  "2026-07-13 10:00:00",
		TraceURL:   "https://mcpzero.io/app/activity/tr_1",
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var out cloud.ActivityEntry
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.ErrorCode == nil || *out.ErrorCode != "search_mode:llm" {
		t.Fatalf("error_code round-trip failed: %#v", out.ErrorCode)
	}
	line := formatActivityLine(e)
	if !strings.Contains(line, "search_mode=llm") {
		t.Fatalf("line missing search_mode: %q", line)
	}
}
