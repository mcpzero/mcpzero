package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/cloud"
	"github.com/mcpzero/mcpzero/cli/internal/config"
	"github.com/mcpzero/mcpzero/cli/internal/mcpconfig"
)

type initOptions struct {
	endpointName  string
	endpointID    string
	forceNew      bool
	apiKey        string
	serverName    string
	noCursor      bool
	projectCursor bool
	teamID        string
	webBase       string
	gwBase        string
	yes           bool
	noTunnel      bool
}

func runInit(args []string) error {
	opts, err := parseInitOptions(args)
	if err != nil {
		return err
	}

	creds, err := auth.LoadCredentials()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Not logged in — starting browser login...")
		creds, err = auth.Login(context.Background(), auth.LoginOptions{
			WebBase: firstNonEmpty(opts.webBase, config.DefaultWebBase),
			GWBase:  firstNonEmpty(opts.gwBase, config.DefaultGWBase),
		})
		if err != nil {
			return err
		}
	}

	web := firstNonEmpty(opts.webBase, creds.WebBase, config.DefaultWebBase)
	gw := firstNonEmpty(opts.gwBase, creds.GWBase, config.DefaultGWBase)
	client := cloud.NewClient(web, creds.RefreshToken)

	if !opts.yes && isInteractiveTerminal(os.Stdin) {
		return runInitWizard(client, gw, opts)
	}
	return runInitAutomatic(client, gw, opts)
}

func parseInitOptions(args []string) (initOptions, error) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	var opts initOptions
	fs.StringVar(&opts.endpointName, "endpoint-name", "default", "Name for a new endpoint (non-interactive)")
	fs.StringVar(&opts.endpointID, "endpoint", "", "Reuse an existing endpoint ID (non-interactive)")
	fs.BoolVar(&opts.forceNew, "new", false, "Always create a new endpoint (non-interactive)")
	fs.StringVar(&opts.apiKey, "api-key", "", "Use an existing API key for Cursor (non-interactive)")
	fs.StringVar(&opts.serverName, "name", "mcpzero", "Cursor MCP server name in mcp.json")
	fs.BoolVar(&opts.noCursor, "no-cursor", false, "Skip writing Cursor mcp.json")
	fs.BoolVar(&opts.projectCursor, "project", false, "Write project .cursor/mcp.json instead of global")
	fs.StringVar(&opts.teamID, "team-id", "", "Team ID for team-scoped endpoint (optional)")
	fs.StringVar(&opts.webBase, "web-base", "", "MCPZERO web base URL")
	fs.StringVar(&opts.gwBase, "gw-base", "", "MCPZERO gateway base URL")
	fs.BoolVar(&opts.yes, "yes", false, "Non-interactive mode (use flags/defaults)")
	fs.BoolVar(&opts.yes, "y", false, "Shorthand for --yes")
	fs.BoolVar(&opts.noTunnel, "no-tunnel", false, "Skip starting a tunnel")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	return opts, nil
}

func runInitWizard(client *cloud.Client, gw string, opts initOptions) error {
	ctx := context.Background()
	prompt := newInitPrompt(os.Stdin, os.Stdout)

	prompt.println("MCPZERO setup")
	prompt.println("We'll connect Cursor to your cloud endpoint through a local tunnel.")
	prompt.println("")

	me, err := client.Me(ctx)
	if err != nil {
		return fmt.Errorf("account info: %w", err)
	}
	prompt.printf("Signed in as %s (%s plan)\n", me.Email, me.Plan)

	endpoints, err := client.ListEndpoints(ctx)
	if err != nil {
		return fmt.Errorf("list endpoints: %w", err)
	}

	ep, createdEndpoint, err := prompt.promptEndpoint(ctx, client, endpoints, me, opts.endpointID)
	if err != nil {
		return err
	}

	keys, err := client.ListAPIKeys(ctx)
	if err != nil {
		return fmt.Errorf("list api keys: %w", err)
	}

	rawKey, keyHint, createdKey, err := prompt.promptAPIKey(ctx, client, ep, keys, opts.apiKey)
	if err != nil {
		return err
	}

	serverName, project, writeCursor, err := prompt.promptCursor(opts.noCursor, opts.serverName, opts.projectCursor)
	if err != nil {
		return err
	}

	mcpURL := mcpconfig.GatewayEndpointURL(gw, ep.ID, "")
	var cursorPath string
	if writeCursor {
		cursorPath, err = mcpconfig.AddCursorHTTPServer(mcpconfig.CursorAddOptions{
			Project:    project,
			ServerName: serverName,
			URL:        mcpURL,
			Headers: map[string]string{
				"Authorization": "Bearer " + rawKey,
			},
		})
		if err != nil {
			return fmt.Errorf("configure cursor: %w", err)
		}
	}

	printInitSummary(printInitSummaryInput{
		ep:              ep,
		createdEndpoint: createdEndpoint,
		keyHint:         keyHint,
		createdKey:      createdKey,
		rawKey:          rawKey,
		mcpURL:          mcpURL,
		cursorPath:      cursorPath,
		serverName:      serverName,
	})

	if opts.noTunnel {
		printInitTunnelHint(ep.ID)
		return nil
	}

	tunnelChoice, err := prompt.promptTunnel()
	if err != nil {
		return err
	}
	if !tunnelChoice.Start {
		printInitTunnelHint(ep.ID)
		return nil
	}

	return startTunnelFromInit(ep.ID, gw, tunnelChoice.Background)
}

func runInitAutomatic(client *cloud.Client, gw string, opts initOptions) error {
	ctx := context.Background()

	existing, err := client.ListEndpoints(ctx)
	if err != nil {
		return fmt.Errorf("list endpoints: %w", err)
	}

	resolved, err := resolveInitEndpoint(initResolveInput{
		Endpoints:  existing,
		EndpointID: opts.endpointID,
		ForceNew:   opts.forceNew,
		TeamID:     opts.teamID,
	})
	if err != nil {
		return err
	}

	var ep cloud.Endpoint
	createdEndpoint := false
	if resolved.CreateNew {
		var teamPtr *string
		if tid := strings.TrimSpace(opts.teamID); tid != "" {
			teamPtr = &tid
		}
		created, err := client.CreateEndpoint(ctx, strings.TrimSpace(opts.endpointName), teamPtr)
		if err != nil {
			return fmt.Errorf("create endpoint: %w", formatEndpointLimitError(err, existing))
		}
		ep = *created
		createdEndpoint = true
	} else {
		ep = *resolved.Endpoint
		fmt.Fprintln(os.Stdout, resolved.ReuseNote)
	}

	var rawKey string
	var keyHint string
	createdKey := false
	if k := strings.TrimSpace(opts.apiKey); k != "" {
		rawKey = k
		keyHint = "(provided)"
	} else {
		key, err := client.CreateAPIKey(ctx, []string{ep.ID})
		if err != nil {
			return fmt.Errorf("create api key: %w", err)
		}
		rawKey = key.RawKey
		keyHint = key.Hint
		createdKey = true
	}

	mcpURL := mcpconfig.GatewayEndpointURL(gw, ep.ID, "")
	var cursorPath string
	if !opts.noCursor {
		cursorPath, err = mcpconfig.AddCursorHTTPServer(mcpconfig.CursorAddOptions{
			Project:    opts.projectCursor,
			ServerName: opts.serverName,
			URL:        mcpURL,
			Headers: map[string]string{
				"Authorization": "Bearer " + rawKey,
			},
		})
		if err != nil {
			return fmt.Errorf("configure cursor: %w", err)
		}
	}

	printInitSummary(printInitSummaryInput{
		ep:              ep,
		createdEndpoint: createdEndpoint,
		keyHint:         keyHint,
		createdKey:      createdKey,
		rawKey:          rawKey,
		mcpURL:          mcpURL,
		cursorPath:      cursorPath,
		serverName:      opts.serverName,
	})

	if !opts.noTunnel {
		printInitTunnelHint(ep.ID)
	}
	return nil
}

type printInitSummaryInput struct {
	ep              cloud.Endpoint
	createdEndpoint bool
	keyHint         string
	createdKey      bool
	rawKey          string
	mcpURL          string
	cursorPath      string
	serverName      string
}

func printInitSummary(in printInitSummaryInput) {
	if in.createdEndpoint {
		fmt.Fprintf(os.Stdout, "Endpoint created: %s (%s)\n", in.ep.ID, in.ep.Name)
	} else {
		fmt.Fprintf(os.Stdout, "Endpoint: %s (%s)\n", in.ep.ID, in.ep.Name)
	}
	if in.createdKey {
		fmt.Fprintf(os.Stdout, "API key created: %s\n", in.keyHint)
	} else {
		fmt.Fprintf(os.Stdout, "API key: %s\n", in.keyHint)
	}
	fmt.Fprintf(os.Stdout, "Gateway URL: %s\n", in.mcpURL)
	if in.cursorPath != "" {
		fmt.Fprintf(os.Stdout, "Cursor config updated: %s (server %q)\n", in.cursorPath, in.serverName)
	}
	if in.createdKey {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "Store the API key now — it won't be shown again:")
		fmt.Fprintf(os.Stdout, "  %s\n", in.rawKey)
	}
}

func printInitTunnelHint(endpointID string) {
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "Start your tunnel:")
	fmt.Fprintf(os.Stdout, "  mcpzero tunnel start --endpoint %s --mcp-auto\n", endpointID)
}

func startTunnelFromInit(endpointID, gw string, background bool) error {
	args := []string{"--endpoint", endpointID, "--mcp-auto", "--gw-base", gw}
	if background {
		args = append(args, "-d")
	}
	fmt.Fprintln(os.Stdout, "")
	if background {
		fmt.Fprintln(os.Stdout, "Starting tunnel in background...")
	} else {
		fmt.Fprintln(os.Stdout, "Starting tunnel in foreground (Ctrl-C to stop)...")
	}
	return tunnelStart(args)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
