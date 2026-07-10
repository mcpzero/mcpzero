package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/cloud"
	"github.com/mcpzero/mcpzero/cli/internal/config"
)

func runWatch(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	endpointID := fs.String("endpoint", "", "Filter by endpoint ID")
	status := fs.String("status", "", "Filter by status (success, error, timeout, auth_denied)")
	search := fs.String("search", "", "Free-text search (tool, IP, server, status, trace id)")
	interval := fs.Duration("interval", 2*time.Second, "Poll interval")
	once := fs.Bool("once", false, "Fetch once and exit (no follow)")
	limit := fs.Int("limit", 20, "Max rows per poll")
	if err := fs.Parse(args); err != nil {
		return err
	}

	creds, err := auth.LoadCredentials()
	if err != nil {
		return err
	}
	web := firstNonEmpty(creds.WebBase, config.DefaultWebBase)
	client := cloud.NewClient(web, creds.RefreshToken)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts := cloud.ActivityListOptions{
		Limit:      *limit,
		EndpointID: strings.TrimSpace(*endpointID),
		Status:     strings.TrimSpace(*status),
		Search:     strings.TrimSpace(*search),
	}

	if *once {
		entries, err := client.ListActivity(ctx, opts)
		if err != nil {
			return err
		}
		// API returns newest first; print oldest→newest for readability.
		printActivityEntries(reverseEntries(entries))
		return nil
	}

	fmt.Fprintf(os.Stderr, "watching activity on %s (Ctrl-C to stop)\n", web)
	var after string
	first := true
	for {
		if ctx.Err() != nil {
			return nil
		}
		pollOpts := opts
		pollOpts.After = after
		if first && after == "" {
			// Initial snapshot: show recent history once.
			pollOpts.After = ""
		}
		entries, err := client.ListActivity(ctx, pollOpts)
		if err != nil {
			return err
		}
		if first {
			printActivityEntries(reverseEntries(entries))
			if len(entries) > 0 {
				after = entries[0].CreatedAt // newest
			}
			first = false
		} else if len(entries) > 0 {
			// entries are newest-first; print chronological.
			printActivityEntries(reverseEntries(entries))
			after = entries[0].CreatedAt
		}

		timer := time.NewTimer(*interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func reverseEntries(in []cloud.ActivityEntry) []cloud.ActivityEntry {
	out := make([]cloud.ActivityEntry, len(in))
	for i := range in {
		out[len(in)-1-i] = in[i]
	}
	return out
}

func printActivityEntries(entries []cloud.ActivityEntry) {
	for _, e := range entries {
		tool := "-"
		if e.ToolName != nil && *e.ToolName != "" {
			tool = *e.ToolName
		}
		server := ""
		if e.ServerName != nil && *e.ServerName != "" {
			server = " @" + *e.ServerName
		}
		lat := ""
		if e.LatencyMs != nil {
			lat = fmt.Sprintf(" %dms", *e.LatencyMs)
		}
		fmt.Fprintf(
			os.Stdout,
			"%s  %-12s  %-28s%s%s  %s\n  → %s\n",
			e.CreatedAt,
			e.Status,
			tool,
			server,
			lat,
			e.EndpointID,
			e.TraceURL,
		)
	}
}
