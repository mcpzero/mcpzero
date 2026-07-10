package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/cloud"
	"github.com/mcpzero/mcpzero/cli/internal/config"
)

func runEndpoint(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mcpzero endpoint <list|create|rm> …")
	}
	switch args[0] {
	case "list", "ls":
		return endpointList(args[1:])
	case "create":
		return endpointCreate(args[1:])
	case "rm", "delete", "remove":
		return endpointRm(args[1:])
	case "help", "-h", "--help":
		printEndpointUsage()
		return nil
	default:
		return fmt.Errorf("unknown endpoint subcommand %q (run: mcpzero endpoint --help)", args[0])
	}
}

func printEndpointUsage() {
	fmt.Fprintf(os.Stdout, `mcpzero endpoint — manage cloud endpoints

Usage:
  mcpzero endpoint list
  mcpzero endpoint create --name <name> [--team <team_id>]
  mcpzero endpoint rm <endpoint_id> [-y]

`)
}

func endpointClient() (*cloud.Client, error) {
	creds, err := auth.LoadCredentials()
	if err != nil {
		return nil, err
	}
	web := firstNonEmpty(creds.WebBase, config.DefaultWebBase)
	return cloud.NewClient(web, creds.RefreshToken), nil
}

func endpointList(args []string) error {
	fs := flag.NewFlagSet("endpoint list", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	client, err := endpointClient()
	if err != nil {
		return err
	}
	endpoints, err := client.ListEndpoints(context.Background())
	if err != nil {
		return err
	}
	if len(endpoints) == 0 {
		fmt.Fprintln(os.Stdout, "No endpoints.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tID\tSTATUS\tSCOPE")
	for _, ep := range endpoints {
		scope := "personal"
		if ep.TeamID != nil && *ep.TeamID != "" {
			scope = "team"
			if ep.TeamName != nil && *ep.TeamName != "" {
				scope = *ep.TeamName
			}
		}
		status := ep.Status
		if status == "" {
			status = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ep.Name, ep.ID, status, scope)
	}
	return w.Flush()
}

func endpointCreate(args []string) error {
	fs := flag.NewFlagSet("endpoint create", flag.ExitOnError)
	name := fs.String("name", "", "Endpoint display name (required)")
	teamID := fs.String("team", "", "Team ID (optional; omit for personal)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	n := strings.TrimSpace(*name)
	if n == "" {
		return fmt.Errorf("--name is required")
	}
	client, err := endpointClient()
	if err != nil {
		return err
	}
	var team *string
	if t := strings.TrimSpace(*teamID); t != "" {
		team = &t
	}
	ep, err := client.CreateEndpoint(context.Background(), n, team)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "created %s (%s)\n", ep.Name, ep.ID)
	return nil
}

func endpointRm(args []string) error {
	fs := flag.NewFlagSet("endpoint rm", flag.ExitOnError)
	yes := fs.Bool("y", false, "Skip confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := strings.TrimSpace(fs.Arg(0))
	if id == "" {
		return fmt.Errorf("usage: mcpzero endpoint rm <endpoint_id> [-y]")
	}
	if !*yes {
		fmt.Fprintf(os.Stderr, "Delete endpoint %s? [y/N] ", id)
		var line string
		if _, err := fmt.Fscanln(os.Stdin, &line); err != nil || strings.ToLower(strings.TrimSpace(line)) != "y" {
			fmt.Fprintln(os.Stderr, "aborted")
			return nil
		}
	}
	client, err := endpointClient()
	if err != nil {
		return err
	}
	if err := client.DeleteEndpoint(context.Background(), id); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "deleted %s\n", id)
	return nil
}
