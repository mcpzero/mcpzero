package upstream

import "testing"

func TestParseHeader(t *testing.T) {
	t.Setenv("UPSTREAM_TOKEN", "secret-xyz")

	cases := []struct {
		in        string
		wantName  string
		wantValue string
		wantErr   bool
	}{
		{in: "Authorization: Bearer abc", wantName: "Authorization", wantValue: "Bearer abc"},
		{in: "X-Org:acme", wantName: "X-Org", wantValue: "acme"},
		{in: "Authorization: Bearer ${UPSTREAM_TOKEN}", wantName: "Authorization", wantValue: "Bearer secret-xyz"},
		{in: "X-Missing: ${NOPE_NOT_SET}", wantName: "X-Missing", wantValue: "${NOPE_NOT_SET}"},
		{in: "no-colon", wantErr: true},
		{in: ": empty-name", wantErr: true},
	}
	for _, c := range cases {
		h, err := ParseHeader(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseHeader(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseHeader(%q): %v", c.in, err)
			continue
		}
		if h.Name != c.wantName || h.Value != c.wantValue {
			t.Errorf("ParseHeader(%q) = %q:%q, want %q:%q", c.in, h.Name, h.Value, c.wantName, c.wantValue)
		}
	}
}

func TestExpectsResponse(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, true},
		{`{"jsonrpc":"2.0","id":"abc","method":"tools/call"}`, true},
		{`{"jsonrpc":"2.0","method":"notifications/initialized"}`, false},
		{`{"jsonrpc":"2.0","id":null,"method":"x"}`, false},
		{`{"jsonrpc":"2.0","id":1}`, false},
	}
	for _, c := range cases {
		if got := expectsResponse([]byte(c.body)); got != c.want {
			t.Errorf("expectsResponse(%s) = %v, want %v", c.body, got, c.want)
		}
	}
}

func TestJSONRPCID(t *testing.T) {
	if id := jsonRPCID([]byte(`{"id":42,"method":"x"}`)); id != "42" {
		t.Errorf("got %q, want 42", id)
	}
	if id := jsonRPCID([]byte(`{"id":"a","method":"x"}`)); id != `"a"` {
		t.Errorf("got %q, want \"a\"", id)
	}
	if id := jsonRPCID([]byte(`{"method":"x"}`)); id != "" {
		t.Errorf("got %q, want empty", id)
	}
}
