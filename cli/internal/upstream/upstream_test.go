package upstream

import (
	"strings"
	"testing"
)

func TestServerRequest(t *testing.T) {
	cases := []struct {
		body       string
		wantMethod string
		wantID     string
		wantOK     bool
	}{
		{`{"jsonrpc":"2.0","id":"s1","method":"roots/list"}`, "roots/list", `"s1"`, true},
		{`{"jsonrpc":"2.0","id":3,"method":"sampling/createMessage"}`, "sampling/createMessage", "3", true},
		{`{"jsonrpc":"2.0","method":"notifications/cancelled"}`, "", "", false},
		{`{"jsonrpc":"2.0","id":null,"method":"roots/list"}`, "", "", false},
		{`{"jsonrpc":"2.0","id":1,"result":{}}`, "", "", false},
	}
	for _, c := range cases {
		method, id, ok := serverRequest([]byte(c.body))
		if ok != c.wantOK || method != c.wantMethod || string(id) != c.wantID {
			t.Errorf("serverRequest(%s) = %q,%s,%v; want %q,%s,%v",
				c.body, method, id, ok, c.wantMethod, c.wantID, c.wantOK)
		}
	}
}

func TestDeclineRootsList(t *testing.T) {
	got := string(declineRootsList([]byte(`"s1"`)))
	for _, want := range []string{`"id":"s1"`, `"code":-32601`} {
		if !strings.Contains(got, want) {
			t.Errorf("declineRootsList missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, `"method"`) {
		t.Errorf("reply must not carry a method: %s", got)
	}
}

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
