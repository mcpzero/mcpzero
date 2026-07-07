package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/mcpzero/mcpzero/cli/internal/cloud"
)

type endpointMenuChoice struct {
	CreateNew  bool
	EndpointID string
	NewName    string
}

func buildEndpointMenu(endpoints []cloud.Endpoint, canCreate bool) []string {
	var lines []string
	for i, ep := range endpoints {
		scope := "personal"
		if ep.TeamID != nil && *ep.TeamID != "" {
			scope = "team"
			if ep.TeamName != nil && *ep.TeamName != "" {
				scope = *ep.TeamName
			}
		}
		status := ep.Status
		if status == "" {
			status = "unknown"
		}
		lines = append(lines, fmt.Sprintf("  [%d] %s — %s (%s, %s)", i+1, ep.Name, ep.ID, scope, status))
	}
	if canCreate {
		lines = append(lines, fmt.Sprintf("  [%d] Create a new endpoint", len(endpoints)+1))
	}
	return lines
}

func parseEndpointMenuChoice(line string, endpoints []cloud.Endpoint, canCreate bool) (endpointMenuChoice, error) {
	max := len(endpoints)
	if canCreate {
		max++
	}
	choice, err := parseMenuChoice(line, max)
	if err != nil {
		return endpointMenuChoice{}, err
	}
	if canCreate && choice == len(endpoints)+1 {
		return endpointMenuChoice{CreateNew: true}, nil
	}
	return endpointMenuChoice{EndpointID: endpoints[choice-1].ID}, nil
}

func keysForEndpoint(keys []cloud.APIKey, endpointID string) []cloud.APIKey {
	var out []cloud.APIKey
	for _, k := range keys {
		if k.Status != "active" {
			continue
		}
		if len(k.EndpointIDs) == 0 {
			out = append(out, k)
			continue
		}
		for _, id := range k.EndpointIDs {
			if id == endpointID {
				out = append(out, k)
				break
			}
		}
	}
	return out
}

type apiKeyMenuChoice struct {
	CreateNew bool
	Paste     bool
}

func parseAPIKeyMenuChoice(line string) (apiKeyMenuChoice, error) {
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "", "1", "c", "create", "new":
		return apiKeyMenuChoice{CreateNew: true}, nil
	case "2", "p", "paste":
		return apiKeyMenuChoice{Paste: true}, nil
	default:
		return apiKeyMenuChoice{}, fmt.Errorf("choose 1 (create) or 2 (paste)")
	}
}

func personalEndpointQuota(me *cloud.AccountMe) (used, limit int, canCreate bool) {
	if me == nil {
		return 0, 1, true
	}
	used = me.PersonalEndpoints.Used
	limit = me.PersonalEndpoints.Limit
	if limit <= 0 {
		limit = 1
	}
	return used, limit, used < limit
}

func (p *initPrompt) promptEndpoint(
	ctx context.Context,
	client *cloud.Client,
	endpoints []cloud.Endpoint,
	me *cloud.AccountMe,
	presetID string,
) (cloud.Endpoint, bool, error) {
	if presetID != "" {
		for _, ep := range endpoints {
			if ep.ID == presetID {
				return ep, false, nil
			}
		}
		return cloud.Endpoint{}, false, fmt.Errorf("endpoint %q not found", presetID)
	}

	_, limit, canCreate := personalEndpointQuota(me)
	if len(endpoints) == 0 {
		canCreate = true
	}

	p.println("")
	p.println("Step 1/4 — Endpoint")
	if len(endpoints) == 0 {
		p.println("No endpoints yet. We'll create one.")
		name, err := p.readLine("Endpoint name [default]: ")
		if err != nil {
			return cloud.Endpoint{}, false, err
		}
		if name == "" {
			name = "default"
		}
		created, err := client.CreateEndpoint(ctx, name, nil)
		if err != nil {
			return cloud.Endpoint{}, false, fmt.Errorf("create endpoint: %w", err)
		}
		return *created, true, nil
	}

	for _, line := range buildEndpointMenu(endpoints, canCreate) {
		p.println(line)
	}
	if !canCreate {
		p.printf("  (personal endpoint limit %d/%d — choose an existing endpoint)\n", limit, limit)
	}

	defaultHint := "1"
	line, err := p.readLine(fmt.Sprintf("Choose [%s]: ", defaultHint))
	if err != nil {
		return cloud.Endpoint{}, false, err
	}

	choice, err := parseEndpointMenuChoice(line, endpoints, canCreate)
	if err != nil {
		return cloud.Endpoint{}, false, err
	}

	if choice.CreateNew {
		name, err := p.readLine("New endpoint name [default]: ")
		if err != nil {
			return cloud.Endpoint{}, false, err
		}
		if name == "" {
			name = "default"
		}
		created, err := client.CreateEndpoint(ctx, name, nil)
		if err != nil {
			return cloud.Endpoint{}, false, fmt.Errorf("create endpoint: %w", formatEndpointLimitError(err, endpoints))
		}
		return *created, true, nil
	}

	for _, ep := range endpoints {
		if ep.ID == choice.EndpointID {
			return ep, false, nil
		}
	}
	return cloud.Endpoint{}, false, fmt.Errorf("endpoint not found")
}

func (p *initPrompt) promptAPIKey(
	ctx context.Context,
	client *cloud.Client,
	ep cloud.Endpoint,
	keys []cloud.APIKey,
	presetKey string,
) (rawKey, hint string, created bool, err error) {
	if k := strings.TrimSpace(presetKey); k != "" {
		return k, "(provided)", false, nil
	}

	p.println("")
	p.println("Step 2/4 — API key")
	scoped := keysForEndpoint(keys, ep.ID)
	if len(scoped) > 0 {
		p.println("Existing keys for this endpoint (raw values are only shown at creation):")
		for _, k := range scoped {
			p.printf("  • %s (%s)\n", k.Hint, k.Status)
		}
		p.println("")
	}

	p.println("  [1] Create a new API key (recommended)")
	p.println("  [2] Paste a saved key (mz_live_…)")
	choiceLine, err := p.readLine("Choose [1]: ")
	if err != nil {
		return "", "", false, err
	}
	choice, err := parseAPIKeyMenuChoice(choiceLine)
	if err != nil {
		return "", "", false, err
	}

	if choice.Paste {
		raw, err := p.readLine("Paste API key: ")
		if err != nil {
			return "", "", false, err
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return "", "", false, fmt.Errorf("API key is required")
		}
		return raw, "(provided)", false, nil
	}

	key, err := client.CreateAPIKey(ctx, []string{ep.ID})
	if err != nil {
		return "", "", false, err
	}
	return key.RawKey, key.Hint, true, nil
}

type cursorMenuChoice struct {
	Write   bool
	Project bool
}

func parseCursorMenuChoice(line string) (cursorMenuChoice, error) {
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "", "1", "global", "home":
		return cursorMenuChoice{Write: true, Project: false}, nil
	case "0", "skip", "none", "no", "n":
		return cursorMenuChoice{}, nil
	case "2", "project":
		return cursorMenuChoice{Write: true, Project: true}, nil
	default:
		return cursorMenuChoice{}, fmt.Errorf("choose 0 (skip), 1 (global ~/.cursor/mcp.json), or 2 (project .cursor/mcp.json)")
	}
}

func (p *initPrompt) promptCursor(skip bool, presetName string, presetProject bool) (serverName string, project bool, write bool, err error) {
	if skip {
		return "", false, false, nil
	}

	p.println("")
	p.println("Step 3/4 — Cursor")
	if presetName != "" && presetName != "mcpzero" {
		return presetName, presetProject, true, nil
	}

	p.println("  [0] Skip — don't write Cursor mcp.json")
	p.println("  [1] Global — ~/.cursor/mcp.json (recommended)")
	p.println("  [2] Project — .cursor/mcp.json in current directory")

	defaultHint := "1"
	if presetProject {
		defaultHint = "2"
	}
	line, err := p.readLine(fmt.Sprintf("Choose [%s]: ", defaultHint))
	if err != nil {
		return "", false, false, err
	}
	if strings.TrimSpace(line) == "" && presetProject {
		line = "2"
	}

	choice, err := parseCursorMenuChoice(line)
	if err != nil {
		return "", false, false, err
	}
	if !choice.Write {
		return "", false, false, nil
	}

	name, err := p.readLine("Server name in mcp.json [mcpzero]: ")
	if err != nil {
		return "", false, false, err
	}
	if name == "" {
		name = "mcpzero"
	}
	return name, choice.Project, true, nil
}

type tunnelPromptChoice struct {
	Start      bool
	Background bool
}

func (p *initPrompt) promptTunnel() (tunnelPromptChoice, error) {
	p.println("")
	p.println("Step 4/4 — Tunnel")
	start, err := p.askYesNo("Start tunnel now (auto-discover local MCP servers)?", true)
	if err != nil || !start {
		return tunnelPromptChoice{}, err
	}
	bg, err := p.askYesNo("Run tunnel in background (free this terminal)?", true)
	if err != nil {
		return tunnelPromptChoice{}, err
	}
	return tunnelPromptChoice{Start: true, Background: bg}, nil
}
