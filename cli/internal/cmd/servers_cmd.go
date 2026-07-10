package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/config"
	"github.com/mcpzero/mcpzero/cli/internal/mcpconfig"
)

func runServers(args []string) error {
	fs := flag.NewFlagSet("servers", flag.ExitOnError)
	mcpConfig := fs.String("mcp-config", "", "Path to MCP config JSON (mcpServers / servers)")
	mcpAuto := fs.Bool("mcp-auto", false, "Discover MCP servers from installed agent configs")
	endpointID := fs.String("endpoint", "", "Endpoint ID for self-loop check (optional)")
	gwBase := fs.String("gw-base", "", "Gateway base for loop check (default: credentials or production)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var (
		specs []mcpconfig.ServerSpec
		err   error
		src   string
	)

	switch {
	case strings.TrimSpace(*mcpConfig) != "":
		specs, err = mcpconfig.Load(strings.TrimSpace(*mcpConfig))
		src = strings.TrimSpace(*mcpConfig)
	case *mcpAuto:
		work, _ := os.Getwd()
		var skipped int
		specs, skipped, err = mcpconfig.Discover(work)
		src = "agent configs (auto)"
		if skipped > 0 {
			fmt.Fprintf(os.Stderr, "note: skipped %d remote MCPZERO gateway URL(s)\n", skipped)
		}
	default:
		return fmt.Errorf("usage: mcpzero servers --mcp-config <file> | --mcp-auto [--endpoint <id>]")
	}
	if err != nil {
		return err
	}
	if len(specs) == 0 {
		fmt.Fprintln(os.Stdout, "No MCP servers found.")
		return nil
	}

	fmt.Fprintf(os.Stdout, "source: %s\n", src)
	fmt.Fprintf(os.Stdout, "servers: %d\n\n", len(specs))

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTRANSPORT\tDETAIL")
	for _, s := range specs {
		name := s.Name
		if s.RawName != "" && s.RawName != s.Name {
			name = fmt.Sprintf("%s (%s)", s.Name, s.RawName)
		}
		if s.IsHTTP() {
			fmt.Fprintf(w, "%s\thttp\t%s\n", name, s.URL)
		} else {
			fmt.Fprintf(w, "%s\tstdio\t%s\n", name, s.Command)
		}
	}
	_ = w.Flush()

	ep := strings.TrimSpace(*endpointID)
	if ep == "" {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "loop check: skipped (pass --endpoint <id> to validate)")
		return nil
	}

	gw := strings.TrimSpace(*gwBase)
	if gw == "" {
		if creds, cerr := auth.LoadCredentials(); cerr == nil {
			gw = firstNonEmpty(creds.GWBase, config.DefaultGWBase)
		} else {
			gw = config.DefaultGWBase
		}
	}

	if err := checkSelfReference(gw, ep, specs); err != nil {
		return fmt.Errorf("loop check failed: %w", err)
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "loop check: ok (no server points at %s/%s)\n", strings.TrimRight(gw, "/"), ep)
	return nil
}
