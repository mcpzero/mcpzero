package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/mcpzero/mcpzero/cli/internal/cloud"
)

func TestPrintWhoamiLimits(t *testing.T) {
	var buf bytes.Buffer
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	me := &cloud.AccountMe{
		Plan: "free",
	}
	me.PersonalEndpoints.Used = 1
	me.PersonalEndpoints.Limit = 1
	printWhoamiLimits(me)

	_ = w.Close()
	os.Stdout = old
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("plan: free")) {
		t.Fatalf("output missing plan: %q", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("personal endpoints: 1/1")) {
		t.Fatalf("output missing endpoints: %q", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("max tools (account total): 50")) {
		t.Fatalf("output missing free account tools: %q", out)
	}
}
