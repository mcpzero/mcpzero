package cmd

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/config"
	"github.com/mcpzero/mcpzero/cli/internal/daemon"
	"github.com/mcpzero/mcpzero/cli/internal/mcpconfig"
	tunnelpkg "github.com/mcpzero/mcpzero/cli/internal/tunnel"
	"github.com/mcpzero/mcpzero/cli/internal/upstream"
)

// stringSlice collects a repeatable string flag (e.g. --mcp-header).
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }

func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runTunnel(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mcpzero tunnel <start|list|logs|attach|stop> ...")
	}

	switch args[0] {
	case "start":
		return tunnelStart(args[1:])
	case "list", "ls":
		return tunnelList(args[1:])
	case "logs":
		return tunnelLogs(args[1:])
	case "attach":
		return tunnelAttach(args[1:])
	case "stop":
		return tunnelStop(args[1:])
	case "remove", "rm":
		return tunnelRemove(args[1:])
	case daemon.InternalRunCommand:
		return tunnelDaemonRun(args[1:])
	default:
		return fmt.Errorf("unknown tunnel subcommand %q (run mcpzero tunnel)", args[0])
	}
}

type startParams struct {
	endpointID   string
	mcpCmd       string
	mcpWorkDir   string
	mcpURL       string
	mcpTransport string
	mcpHeaders   []upstream.Header
	// servers holds the MCP servers selected from --mcp-config. When set, the
	// tunnel multiplexes them and the single mcpCmd/mcpURL fields are unused.
	servers []mcpconfig.ServerSpec
	gwBase  string
	mgmtKey string
}

// mgmtKeyEnvVar is the user-facing env var for supplying a management key,
// matching the convention used by the dashboard and the tunnel protocol docs.
const mgmtKeyEnvVar = "MCPZERO_MGMT_KEY"

func tunnelStart(args []string) error {
	fs := flag.NewFlagSet("tunnel start", flag.ExitOnError)
	endpointID := fs.String("endpoint", "", "endpoint ID from dashboard")
	mgmtKey := fs.String("mgmt-key", "", "management key for headless/CI auth (or set "+mgmtKeyEnvVar+"; optional if logged in)")
	mcpCmd := fs.String("mcp-cmd", "", "command to start local MCP server (stdio)")
	mcpWorkDir := fs.String("mcp-workdir", "", "working directory for --mcp-cmd")
	mcpURL := fs.String("mcp-url", "", "URL of an HTTP MCP server to proxy (alternative to --mcp-cmd)")
	mcpTransport := fs.String("mcp-transport", "auto", "HTTP MCP transport for --mcp-url: auto|streamable-http|sse")
	var mcpHeaders stringSlice
	fs.Var(&mcpHeaders, "mcp-header", "HTTP header for --mcp-url (repeatable), e.g. \"Authorization: Bearer ${TOKEN}\"")
	mcpConfig := fs.String("mcp-config", "", "path to a standard MCP server config (mcpServers JSON); multiplexes selected servers over one tunnel")
	mcpAuto := fs.Bool("mcp-auto", false, "auto-discover MCP servers from installed AI agent configs (home + current dir) and choose interactively")
	var mcpServerSel stringSlice
	fs.Var(&mcpServerSel, "mcp-server", "server name from --mcp-config/--mcp-auto to start (repeatable; skips the interactive prompt)")
	gwBase := fs.String("gw-base", "", "MCPZERO gateway base URL (default from login or config)")
	detach := fs.Bool("detach", false, "run the tunnel in the background")
	detachShort := fs.Bool("d", false, "run the tunnel in the background (shorthand)")
	force := fs.Bool("force", false, "start even if another tunnel is already running for this endpoint")
	forceShort := fs.Bool("f", false, "shorthand for --force")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *endpointID == "" {
		return fmt.Errorf("--endpoint is required")
	}
	if *mcpCmd == "" && *mcpURL == "" && *mcpConfig == "" && !*mcpAuto {
		return fmt.Errorf("one of --mcp-cmd, --mcp-url, --mcp-config, or --mcp-auto is required")
	}
	if *mcpCmd != "" && *mcpURL != "" {
		return fmt.Errorf("--mcp-cmd and --mcp-url are mutually exclusive")
	}
	if *mcpConfig != "" && *mcpAuto {
		return fmt.Errorf("--mcp-config and --mcp-auto are mutually exclusive")
	}
	multiSource := *mcpConfig != "" || *mcpAuto
	if multiSource && (*mcpCmd != "" || *mcpURL != "") {
		return fmt.Errorf("--mcp-config/--mcp-auto cannot be combined with --mcp-cmd or --mcp-url")
	}
	if len(mcpHeaders) > 0 && *mcpURL == "" {
		return fmt.Errorf("--mcp-header requires --mcp-url")
	}
	if len(mcpServerSel) > 0 && !multiSource {
		return fmt.Errorf("--mcp-server requires --mcp-config or --mcp-auto")
	}

	headers, err := parseHeaders(mcpHeaders)
	if err != nil {
		return err
	}

	var selectedServers []mcpconfig.ServerSpec
	switch {
	case *mcpConfig != "":
		selectedServers, err = loadAndSelectServers(*mcpConfig, mcpServerSel)
		if err != nil {
			return err
		}
	case *mcpAuto:
		selectedServers, err = discoverAndSelectServers(mcpServerSel)
		if err != nil {
			return err
		}
	}

	if err := checkEndpointConflict(*endpointID, *force || *forceShort); err != nil {
		return err
	}

	mgmtKeyValue := *mgmtKey
	if mgmtKeyValue == "" {
		mgmtKeyValue = os.Getenv(mgmtKeyEnvVar)
	}

	resolvedGWBase, err := resolveGWBase(*gwBase, mgmtKeyValue)
	if err != nil {
		return err
	}

	if err := checkSelfReference(resolvedGWBase, *endpointID, selectedServers); err != nil {
		return err
	}

	p := startParams{
		endpointID:   *endpointID,
		mcpCmd:       *mcpCmd,
		mcpWorkDir:   *mcpWorkDir,
		mcpURL:       *mcpURL,
		mcpTransport: *mcpTransport,
		mcpHeaders:   headers,
		servers:      selectedServers,
		gwBase:       resolvedGWBase,
		mgmtKey:      mgmtKeyValue,
	}

	if *detach || *detachShort {
		return startDetached(p)
	}
	return startForeground(p)
}

func parseHeaders(raw []string) ([]upstream.Header, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	headers := make([]upstream.Header, 0, len(raw))
	for _, r := range raw {
		h, err := upstream.ParseHeader(r)
		if err != nil {
			return nil, err
		}
		headers = append(headers, h)
	}
	return headers, nil
}

// checkEndpointConflict guards against pointing multiple local tunnels at the
// same endpoint. The gateway keeps a single active tunnel per endpoint and
// evicts the previous one on each register; since daemons auto-reconnect, two
// tunnels on one endpoint would repeatedly kick each other ("replaced by new
// connection") and route requests to whichever upstream currently holds the
// slot. Without --force this is a hard error.
func checkEndpointConflict(endpointID string, force bool) error {
	states, err := daemon.List()
	if err != nil {
		// Listing is best-effort: never block a start because state is unreadable.
		return nil
	}

	var running []*daemon.State
	for _, s := range states {
		if s.EndpointID == endpointID && daemon.IsAlive(s) {
			running = append(running, s)
		}
	}
	if len(running) == 0 {
		return nil
	}

	var b strings.Builder
	for _, s := range running {
		fmt.Fprintf(&b, "    %s  ->  %s\n", s.Hash, truncate(s.Target(), 50))
	}

	if !force {
		return fmt.Errorf(
			"endpoint %s already has a running tunnel:\n%s\n"+
				"The gateway allows only one active tunnel per endpoint; starting another\n"+
				"would make them repeatedly evict each other. Stop the existing tunnel first\n"+
				"(mcpzero tunnel rm -f <id>), use a different --endpoint, or pass --force to override.",
			endpointID, b.String(),
		)
	}

	fmt.Fprintf(os.Stderr,
		"warning: endpoint %s already has a running tunnel; --force will let them compete for the single active slot:\n%s",
		endpointID, b.String(),
	)
	return nil
}

// resolveGWBase validates auth and resolves the gateway base URL.
func resolveGWBase(gwBaseFlag, mgmtKey string) (string, error) {
	gwBase := gwBaseFlag
	if mgmtKey == "" {
		creds, err := auth.LoadCredentials()
		if err != nil {
			return "", fmt.Errorf("no --mgmt-key provided and not logged in: %w", err)
		}
		if gwBase == "" {
			gwBase = creds.GWBase
		}
	}
	if gwBase == "" {
		gwBase = config.DefaultGWBase
	}
	return gwBase, nil
}

func startForeground(p startParams) error {
	refreshToken := ""
	if p.mgmtKey == "" {
		creds, err := auth.LoadCredentials()
		if err != nil {
			return err
		}
		refreshToken = creds.RefreshToken
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := tunnelpkg.Client{
		GWBase:       p.gwBase,
		EndpointID:   p.endpointID,
		MgmtKey:      p.mgmtKey,
		RefreshToken: refreshToken,
	}

	if len(p.servers) > 0 {
		ups, err := namedUpstreamsFromSpecs(p.servers, nil)
		if err != nil {
			return err
		}
		client.Upstreams = ups
		printEndpointURLs(p.gwBase, p.endpointID, specNames(p.servers))
	} else {
		up, err := buildUpstream(p.mcpCmd, p.mcpWorkDir, p.mcpURL, p.mcpTransport, p.mcpHeaders, nil)
		if err != nil {
			return err
		}
		client.Upstream = up
		printEndpointURLs(p.gwBase, p.endpointID, nil)
	}

	fmt.Fprintf(os.Stderr, "connecting to %s/tunnel/%s …\n", p.gwBase, p.endpointID)
	return client.Run(ctx)
}

// loadAndSelectServers parses an --mcp-config file and resolves which servers
// to start (interactively, or via --mcp-server). It runs in the foreground
// parent so the prompt has access to the terminal even for detached tunnels.
func loadAndSelectServers(path string, preselected []string) ([]mcpconfig.ServerSpec, error) {
	specs, err := mcpconfig.Load(path)
	if err != nil {
		return nil, err
	}
	selected, err := mcpconfig.SelectServers(specs, preselected, os.Stdin, os.Stderr)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "selected %d server(s): %s\n", len(selected), strings.Join(specNames(selected), ", "))
	return selected, nil
}

// discoverAndSelectServers scans known AI-agent config locations (home and the
// current directory) for MCP servers and resolves which to start. Like
// --mcp-config, selection runs in the foreground parent.
func discoverAndSelectServers(preselected []string) ([]mcpconfig.ServerSpec, error) {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = ""
	}
	specs, err := mcpconfig.Discover(workDir)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "discovered %d MCP server(s) from agent configs\n", len(specs))
	selected, err := mcpconfig.SelectServers(specs, preselected, os.Stdin, os.Stderr)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "selected %d server(s): %s\n", len(selected), strings.Join(specNames(selected), ", "))
	return selected, nil
}

// specNames returns the normalized names of the given specs.
func specNames(specs []mcpconfig.ServerSpec) []string {
	names := make([]string, 0, len(specs))
	for _, s := range specs {
		names = append(names, s.Name)
	}
	return names
}

// printEndpointURLs writes the remote URL(s) clients should use. With named
// servers each gets a /<name> sub-path; otherwise the plain endpoint URL.
func printEndpointURLs(gwBase, endpointID string, serverNames []string) {
	base := strings.TrimRight(gwBase, "/")
	if len(serverNames) == 0 {
		fmt.Fprintf(os.Stderr, "remote MCP URL: %s/v1/%s\n", base, endpointID)
		return
	}
	fmt.Fprintln(os.Stderr, "remote MCP URLs:")
	for _, name := range serverNames {
		fmt.Fprintf(os.Stderr, "  %-20s %s/v1/%s/%s\n", name, base, endpointID, name)
	}
}

// buildUpstream constructs an stdio or HTTP upstream from resolved parameters.
func buildUpstream(
	mcpCmd, mcpWorkDir, mcpURL, mcpTransport string,
	headers []upstream.Header,
	onStart func(pid int),
) (upstream.Upstream, error) {
	if mcpURL != "" {
		return upstream.NewHTTP(mcpURL, headers, mcpTransport)
	}
	return upstream.NewStdio(mcpCmd, mcpWorkDir, nil, onStart), nil
}

// upstreamFromState reconstructs the single upstream for a background tunnel,
// decrypting any persisted HTTP headers and recording the stdio process group.
func upstreamFromState(s *daemon.State) (upstream.Upstream, error) {
	headers, err := s.Headers()
	if err != nil {
		return nil, err
	}
	onStart := func(pid int) {
		s.MCPGroupPID = pid
		_ = daemon.Save(s)
	}
	return buildUpstream(s.MCPCmd, s.MCPWorkDir, s.MCPURL, s.MCPTransport, headers, onStart)
}

func startDetached(p startParams) error {
	if !daemon.Supported() {
		return fmt.Errorf("background tunnels are not supported on this platform; run without -d")
	}

	var daemonServers []daemon.ServerSpec
	if len(p.servers) > 0 {
		var err error
		daemonServers, err = daemonServersFromSpecs(p.servers)
		if err != nil {
			return err
		}
	}

	s, err := daemon.Start(daemon.SpawnConfig{
		EndpointID:   p.endpointID,
		MCPCmd:       p.mcpCmd,
		MCPWorkDir:   p.mcpWorkDir,
		MCPURL:       p.mcpURL,
		MCPTransport: p.mcpTransport,
		MCPHeaders:   p.mcpHeaders,
		Servers:      daemonServers,
		GWBase:       p.gwBase,
		MgmtKey:      p.mgmtKey,
	})
	if err != nil {
		return err
	}

	// Give the child a moment to connect and surface early failures. The
	// child records its own status on exit, which we use instead of process
	// liveness (a freshly exited child is a zombie until the parent leaves,
	// so it would otherwise still look "alive").
	time.Sleep(800 * time.Millisecond)
	current, err := daemon.Load(s.Hash)
	if err != nil {
		return err
	}
	if current.Status == daemon.StatusError || current.Status == daemon.StatusStopped || !daemon.IsAlive(current) {
		fmt.Fprintf(os.Stderr, "tunnel %s exited during startup. Recent logs:\n", s.Hash)
		if logPath, lerr := daemon.LogPath(s.Hash); lerr == nil {
			_ = tailFile(logPath, false)
		}
		_ = daemon.Remove(s.Hash)
		return fmt.Errorf("tunnel failed to start")
	}

	fmt.Printf("Tunnel started in background.\n")
	fmt.Printf("  id:       %s\n", s.Hash)
	fmt.Printf("  endpoint: %s\n", s.EndpointID)
	fmt.Printf("\n")
	printEndpointURLs(p.gwBase, p.endpointID, specNames(p.servers))
	fmt.Printf("\n")
	fmt.Printf("  mcpzero tunnel logs %s -f   # follow logs\n", s.Hash)
	fmt.Printf("  mcpzero tunnel stop %s      # stop tunnel\n", s.Hash)
	return nil
}

func tunnelDaemonRun(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: internal daemon-run <hash>")
	}
	hash := args[0]

	s, err := daemon.Load(hash)
	if err != nil {
		return err
	}

	mgmtKey := os.Getenv(daemon.MgmtKeyEnvVar)
	refreshToken := ""
	gwBase := s.GWBase
	if mgmtKey == "" {
		creds, err := auth.LoadCredentials()
		if err != nil {
			_ = daemon.MarkStatus(hash, daemon.StatusError)
			return err
		}
		refreshToken = creds.RefreshToken
		if gwBase == "" {
			gwBase = creds.GWBase
		}
	}
	if gwBase == "" {
		gwBase = config.DefaultGWBase
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := tunnelpkg.Client{
		GWBase:       gwBase,
		EndpointID:   s.EndpointID,
		MgmtKey:      mgmtKey,
		RefreshToken: refreshToken,
	}

	if len(s.Servers) > 0 {
		onStartFor := func(name string) func(pid int) {
			return func(pid int) {
				for i := range s.Servers {
					if s.Servers[i].Name == name {
						s.Servers[i].MCPGroupPID = pid
						_ = daemon.Save(s)
						return
					}
				}
			}
		}
		ups, err := namedUpstreamsFromState(s.Servers, onStartFor)
		if err != nil {
			_ = daemon.MarkStatus(hash, daemon.StatusError)
			return err
		}
		client.Upstreams = ups
	} else {
		up, err := upstreamFromState(s)
		if err != nil {
			_ = daemon.MarkStatus(hash, daemon.StatusError)
			return err
		}
		client.Upstream = up
	}

	runErr := client.Run(ctx)

	// A gateway-initiated close (e.g. replaced by a newer tunnel) is a normal
	// stop; anything else is an error. In all cases client.Run has already
	// torn down the MCP subprocess group before returning.
	var de *tunnelpkg.DisconnectError
	switch {
	case runErr == nil:
		_ = daemon.MarkStatus(hash, daemon.StatusStopped)
		fmt.Fprintf(os.Stderr, "tunnel stopped\n")
		return nil
	case errors.As(runErr, &de) && de.Terminal:
		_ = daemon.MarkStatus(hash, daemon.StatusStopped)
		fmt.Fprintf(os.Stderr, "tunnel stopped: %v\n", runErr)
		return nil
	default:
		_ = daemon.MarkStatus(hash, daemon.StatusError)
		fmt.Fprintf(os.Stderr, "tunnel exited with error: %v\n", runErr)
		return runErr
	}
}

func tunnelList(_ []string) error {
	states, err := daemon.List()
	if err != nil {
		return err
	}
	if len(states) == 0 {
		fmt.Println("No tunnels.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tENDPOINT\tSTATUS\tPID\tSTARTED\tMCP-TARGET")
	for _, s := range states {
		pid := "-"
		if s.PID > 0 {
			pid = fmt.Sprintf("%d", s.PID)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			s.Hash,
			s.EndpointID,
			daemon.EffectiveStatus(s),
			pid,
			s.StartedAt.Format("2006-01-02 15:04"),
			truncate(s.Target(), 40),
		)
	}
	return w.Flush()
}

func tunnelLogs(args []string) error {
	fs := flag.NewFlagSet("tunnel logs", flag.ExitOnError)
	follow := fs.Bool("f", false, "follow log output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: mcpzero tunnel logs <id> [-f]")
	}

	s, err := daemon.Resolve(fs.Arg(0))
	if err != nil {
		return err
	}
	logPath, err := daemon.LogPath(s.Hash)
	if err != nil {
		return err
	}
	return tailFile(logPath, *follow)
}

func tunnelAttach(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mcpzero tunnel attach <id>")
	}
	s, err := daemon.Resolve(args[0])
	if err != nil {
		return err
	}
	logPath, err := daemon.LogPath(s.Hash)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Attached to tunnel %s (Ctrl-C to detach; tunnel keeps running)\n", s.Hash)
	return tailFile(logPath, true)
}

func tunnelStop(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mcpzero tunnel stop <id>")
	}
	s, err := daemon.Resolve(args[0])
	if err != nil {
		return err
	}
	if err := daemon.Stop(s); err != nil {
		return err
	}
	fmt.Printf("Stopped tunnel %s (run 'mcpzero tunnel rm %s' to clear it)\n", s.Hash, s.Hash)
	return nil
}

func tunnelRemove(args []string) error {
	fs := flag.NewFlagSet("tunnel remove", flag.ExitOnError)
	force := fs.Bool("force", false, "stop the tunnel and its child processes before removing")
	forceShort := fs.Bool("f", false, "shorthand for --force")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: mcpzero tunnel remove [-f] <id>")
	}

	s, err := daemon.Resolve(fs.Arg(0))
	if err != nil {
		return err
	}

	if daemon.IsAlive(s) {
		if !*force && !*forceShort {
			return fmt.Errorf("tunnel %s is still running; use -f/--force to stop and remove it", s.Hash)
		}
		// Stop terminates the daemon and its MCP subprocess group.
		if err := daemon.Stop(s); err != nil {
			return err
		}
	}

	if err := daemon.Remove(s.Hash); err != nil {
		return err
	}
	fmt.Printf("Removed tunnel %s\n", s.Hash)
	return nil
}

// tailFile prints a file, optionally following appended output until Ctrl-C.
func tailFile(path string, follow bool) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no logs available for this tunnel yet")
		}
		return err
	}
	defer f.Close()

	if _, err := io.Copy(os.Stdout, f); err != nil {
		return err
	}
	if !follow {
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reader := bufio.NewReader(f)
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr)
			return nil
		default:
		}

		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			fmt.Print(line)
		}
		if err == io.EOF {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if err != nil {
			return err
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
