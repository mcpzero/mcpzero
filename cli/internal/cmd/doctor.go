package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/cloud"
	"github.com/mcpzero/mcpzero/cli/internal/config"
	"github.com/mcpzero/mcpzero/cli/internal/daemon"
	"github.com/mcpzero/mcpzero/cli/internal/mcpconfig"
	"github.com/mcpzero/mcpzero/cli/internal/planlimits"
	"github.com/mcpzero/mcpzero/cli/internal/version"
)

type checkStatus string

const (
	checkOK   checkStatus = "ok"
	checkWarn checkStatus = "warn"
	checkFail checkStatus = "fail"
)

type checkResult struct {
	Name    string
	Status  checkStatus
	Detail  string
	Hint    string
}

func (c checkResult) printTo(w io.Writer) {
	icon := "✓"
	switch c.Status {
	case checkWarn:
		icon = "!"
	case checkFail:
		icon = "✗"
	}
	fmt.Fprintf(w, "  [%s] %s", icon, c.Name)
	if c.Detail != "" {
		fmt.Fprintf(w, ": %s", c.Detail)
	}
	fmt.Fprintln(w)
	if c.Hint != "" && c.Status != checkOK {
		fmt.Fprintf(w, "      → %s\n", c.Hint)
	}
}

func runDoctor(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: mcpzero doctor")
	}

	ctx := context.Background()
	results := runDoctorChecks(ctx)

	fmt.Fprintf(os.Stdout, "mcpzero doctor (v%s)\n\n", version.Version)
	failures := 0
	warnings := 0
	for _, r := range results {
		r.printTo(os.Stdout)
		switch r.Status {
		case checkFail:
			failures++
		case checkWarn:
			warnings++
		}
	}

	fmt.Fprintln(os.Stdout)
	if failures > 0 {
		fmt.Fprintf(os.Stdout, "%d check(s) failed, %d warning(s).\n", failures, warnings)
		return fmt.Errorf("doctor found %d issue(s) — fix the items marked ✗", failures)
	}
	if warnings > 0 {
		fmt.Fprintf(os.Stdout, "All required checks passed (%d warning(s)).\n", warnings)
	} else {
		fmt.Fprintln(os.Stdout, "All checks passed.")
	}
	return nil
}

func runDoctorChecks(ctx context.Context) []checkResult {
	var results []checkResult

	results = append(results, checkResult{
		Name:   "CLI version",
		Status: checkOK,
		Detail: version.Version,
	})

	creds, credErr := auth.LoadCredentials()
	if credErr != nil {
		results = append(results, checkResult{
			Name:   "Login",
			Status: checkFail,
			Detail: "not logged in",
			Hint:   "run: mcpzero login",
		})
		results = append(results, doctorUnreachableChecks(config.DefaultWebBase, config.DefaultGWBase)...)
		return results
	}

	results = append(results, checkResult{
		Name:   "Credentials file",
		Status: checkOK,
		Detail: creds.Email,
	})

	webBase := firstNonEmpty(creds.WebBase, config.DefaultWebBase)
	gwBase := firstNonEmpty(creds.GWBase, config.DefaultGWBase)

	if err := cloud.PingWeb(ctx, webBase); err != nil {
		results = append(results, checkResult{
			Name:   "Dashboard reachable",
			Status: checkFail,
			Detail: fmt.Sprintf("%s (%v)", webBase, err),
			Hint:   "check network, VPN, or --web-base",
		})
	} else {
		results = append(results, checkResult{
			Name:   "Dashboard reachable",
			Status: checkOK,
			Detail: webBase,
		})
	}

	if health, err := cloud.GatewayHealth(ctx, gwBase); err != nil {
		results = append(results, checkResult{
			Name:   "Gateway reachable",
			Status: checkFail,
			Detail: fmt.Sprintf("%s (%v)", gwBase, err),
			Hint:   "check network or --gw-base",
		})
	} else {
		detail := gwBase
		if health.Status != "" {
			detail = fmt.Sprintf("%s (%s)", gwBase, health.Status)
		}
		results = append(results, checkResult{
			Name:   "Gateway reachable",
			Status: checkOK,
			Detail: detail,
		})
	}

	client := cloud.NewClient(webBase, creds.RefreshToken)
	me, err := client.Me(ctx)
	if err != nil {
		hint := "run: mcpzero logout && mcpzero login"
		if apiErr, ok := cloud.AsAPIError(err); ok && apiErr.Status == 404 {
			hint = "CLI APIs are not on this server yet — deploy web, or login to dev: mcpzero login --web-base https://dev.mcpzero.io --gw-base https://gw-dev.mcpzero.io"
		}
		results = append(results, checkResult{
			Name:   "CLI token valid",
			Status: checkFail,
			Detail: err.Error(),
			Hint:   hint,
		})
		return results
	}
	results = append(results, checkResult{
		Name:   "CLI token valid",
		Status: checkOK,
		Detail: fmt.Sprintf("%s plan", me.Plan),
	})

	endpoints, err := client.ListEndpoints(ctx)
	if err != nil {
		results = append(results, checkResult{
			Name:   "Endpoints",
			Status: checkWarn,
			Detail: err.Error(),
		})
	} else if len(endpoints) == 0 {
		results = append(results, checkResult{
			Name:   "Endpoints",
			Status: checkWarn,
			Detail: "none yet",
			Hint:   "run: mcpzero init",
		})
	} else {
		online := 0
		for _, ep := range endpoints {
			if ep.Status == "online" {
				online++
			}
		}
		results = append(results, checkResult{
			Name:   "Endpoints",
			Status: checkOK,
			Detail: fmt.Sprintf("%d total, %d online", len(endpoints), online),
		})
	}

	keys, err := client.ListAPIKeys(ctx)
	if err != nil {
		results = append(results, checkResult{
			Name:   "API keys",
			Status: checkWarn,
			Detail: err.Error(),
		})
	} else {
		active := 0
		for _, k := range keys {
			if k.Status == "active" {
				active++
			}
		}
		status := checkOK
		hint := ""
		if active == 0 {
			status = checkWarn
			hint = "create a key in the dashboard or run: mcpzero init"
		}
		results = append(results, checkResult{
			Name:   "API keys",
			Status: status,
			Detail: fmt.Sprintf("%d active", active),
			Hint:   hint,
		})
	}

	cursorPath, err := mcpconfig.DefaultCursorConfigPath(false, "")
	if err == nil {
		if _, err := os.Stat(cursorPath); err == nil {
			results = append(results, checkResult{
				Name:   "Cursor config",
				Status: checkOK,
				Detail: cursorPath,
			})
		} else {
			results = append(results, checkResult{
				Name:   "Cursor config",
				Status: checkWarn,
				Detail: "not found",
				Hint:   "run: mcpzero init or mcpzero cursor add",
			})
		}
	}

	if daemon.Supported() {
		states, err := daemon.List()
		if err != nil {
			results = append(results, checkResult{
				Name:   "Background tunnels",
				Status: checkWarn,
				Detail: err.Error(),
			})
		} else {
			running := 0
			for _, s := range states {
				if daemon.IsAlive(s) {
					running++
				}
			}
			results = append(results, checkResult{
				Name:   "Background tunnels",
				Status: checkOK,
				Detail: fmt.Sprintf("%d registered, %d running", len(states), running),
			})
		}
	}

	return results
}

func doctorUnreachableChecks(webBase, gwBase string) []checkResult {
	return []checkResult{
		{
			Name:   "Dashboard reachable",
			Status: checkWarn,
			Detail: "skipped (not logged in)",
			Hint:   webBase,
		},
		{
			Name:   "Gateway reachable",
			Status: checkWarn,
			Detail: "skipped (not logged in)",
			Hint:   gwBase,
		},
	}
}

func printWhoamiLimits(me *cloud.AccountMe) {
	plan := me.Plan
	fmt.Fprintf(os.Stdout, "plan: %s\n", plan)
	fmt.Fprintf(os.Stdout, "personal endpoints: %d/%d\n", me.PersonalEndpoints.Used, me.PersonalEndpoints.Limit)
	fmt.Fprintf(os.Stdout, "rate limit: %d req/min per endpoint\n", planlimits.RateLimitRPM(plan))
	fmt.Fprintf(os.Stdout, "payload retention: %s\n", planlimits.PayloadRetentionLabel(plan))
	fmt.Fprintf(os.Stdout, "max tools per tunnel: %s\n", planlimits.FormatToolsLimit(plan))
	if planlimits.ClustersEnabled(plan) {
		fmt.Fprintln(os.Stdout, "endpoint clusters: yes (Team+)")
	} else {
		fmt.Fprintln(os.Stdout, "endpoint clusters: no (Team+ required)")
	}
}

func endpointOwned(endpoints []cloud.Endpoint, endpointID string) bool {
	for _, ep := range endpoints {
		if ep.ID == endpointID {
			return true
		}
	}
	return false
}

func validateEndpointID(endpointID string) error {
	id := strings.TrimSpace(endpointID)
	if id == "" {
		return fmt.Errorf("endpoint ID is required")
	}
	if !strings.HasPrefix(id, "ep_") && !strings.HasPrefix(id, "epc_") {
		return fmt.Errorf("endpoint ID %q should start with ep_ or epc_", id)
	}
	return nil
}
