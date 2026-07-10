package cmd

import (
	"fmt"
	"os"

	"github.com/mcpzero/mcpzero/cli/internal/version"
)

func Execute(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "login":
		return login(args[1:])
	case "logout":
		return logout(args[1:])
	case "whoami":
		return whoami(args[1:])
	case "init":
		return runInit(args[1:])
	case "cursor":
		return runCursor(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "watch":
		return runWatch(args[1:])
	case "endpoint":
		return runEndpoint(args[1:])
	case "servers":
		return runServers(args[1:])
	case "tunnel":
		return runTunnel(args[1:])
	case "version", "-v", "--version":
		fmt.Printf("mcpzero %s\n", version.Version)
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (run mcpzero help)", args[0])
	}
}

func printUsage() {
	fmt.Fprintf(os.Stdout, `mcpzero — MCPZERO tunnel CLI

Usage:
  mcpzero login                  Browser login (opens dashboard)
  mcpzero logout                 Clear local credentials
  mcpzero whoami                 Show logged-in user
  mcpzero whoami --limits        Show plan quotas and limits
  mcpzero doctor                 Diagnose login, connectivity, and setup
  mcpzero watch                  Tail cloud Activity (poll; Ctrl-C to stop)
  mcpzero init                   Interactive setup (endpoint, API key, Cursor, tunnel)
  mcpzero init -y                Non-interactive setup (flags/defaults)
  mcpzero cursor add             Add MCPZERO endpoint to Cursor mcp.json
  mcpzero endpoint list          List endpoints
  mcpzero endpoint create        Create an endpoint (--name, optional --team)
  mcpzero endpoint rm <id>       Delete an endpoint you own (-y to skip confirm)
  mcpzero servers                List/validate MCP config (--mcp-config | --mcp-auto)
  mcpzero version                Print version

Tunnels:
  mcpzero tunnel start [-d]      Start a tunnel (-d runs it in the background)
                                  (--mcp-cmd | --mcp-url | --mcp-config <file> | --mcp-auto)
  mcpzero tunnel list            List background tunnels
  mcpzero tunnel logs <id> [-f]  Show tunnel logs (-f to follow)
  mcpzero tunnel attach <id>     Follow a tunnel's logs (Ctrl-C detaches)
  mcpzero tunnel stop <id>       Stop a tunnel and its child processes
  mcpzero tunnel rm [-f] <id>    Remove a tunnel's records (-f stops it first)

Run 'mcpzero tunnel start --help' for tunnel start options.
`)
}
