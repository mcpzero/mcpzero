package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mcpzero/mcpzero/cli/internal/config"
	"github.com/mcpzero/mcpzero/cli/internal/mcpconfig"
)

func runCursor(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mcpzero cursor add ...")
	}

	switch args[0] {
	case "add":
		return cursorAdd(args[1:])
	default:
		return fmt.Errorf("unknown cursor subcommand %q (run mcpzero cursor add)", args[0])
	}
}

func cursorAdd(args []string) error {
	fs := flag.NewFlagSet("cursor add", flag.ExitOnError)
	endpointID := fs.String("endpoint", "", "Endpoint ID (required)")
	apiKey := fs.String("api-key", "", "API key (mz_live_…, required)")
	server := fs.String("server", "", "Direct server route name (optional)")
	serverName := fs.String("name", "mcpzero", "MCP server name in mcp.json")
	project := fs.Bool("project", false, "Write project .cursor/mcp.json instead of global")
	gwBase := fs.String("gw-base", config.DefaultGWBase, "MCPZERO gateway base URL")
	configPath := fs.String("config", "", "Override Cursor mcp.json path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*endpointID) == "" {
		return fmt.Errorf("--endpoint is required")
	}
	if strings.TrimSpace(*apiKey) == "" {
		return fmt.Errorf("--api-key is required")
	}

	mcpURL := mcpconfig.GatewayEndpointURL(*gwBase, *endpointID, *server)
	path, err := mcpconfig.AddCursorHTTPServer(mcpconfig.CursorAddOptions{
		ConfigPath: *configPath,
		Project:    *project,
		ServerName: *serverName,
		URL:        mcpURL,
		Headers: map[string]string{
			"Authorization": "Bearer " + strings.TrimSpace(*apiKey),
		},
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Added %q to %s\n", *serverName, path)
	fmt.Fprintf(os.Stdout, "URL: %s\n", mcpURL)
	return nil
}
