package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/cloud"
	"github.com/mcpzero/mcpzero/cli/internal/mcpconfig"
)

type tunnelPreflightInput struct {
	EndpointID string
	GWBase     string
	MgmtKey    string
	Servers    []mcpconfig.ServerSpec
}

// runTunnelPreflight validates connectivity and ownership before starting a tunnel.
func runTunnelPreflight(ctx context.Context, in tunnelPreflightInput) error {
	if err := validateEndpointID(in.EndpointID); err != nil {
		return err
	}

	if err := cloud.PingGateway(ctx, in.GWBase); err != nil {
		return fmt.Errorf("gateway %s unreachable: %w (run: mcpzero doctor)", in.GWBase, err)
	}

	mgmtKey := strings.TrimSpace(in.MgmtKey)
	if mgmtKey != "" {
		return nil
	}

	if isLocalGateway(in.GWBase) {
		return fmt.Errorf(
			"local gateway %s requires --mgmt-key or %s (run: mcpzero doctor)",
			in.GWBase, mgmtKeyEnvVar,
		)
	}

	return runTunnelPreflightCloud(ctx, in.EndpointID)
}

func runTunnelPreflightCloud(ctx context.Context, endpointID string) error {
	creds, err := auth.LoadCredentials()
	if err != nil {
		return fmt.Errorf("not logged in: %w (run: mcpzero login)", err)
	}

	client := cloud.NewClient(creds.WebBase, creds.RefreshToken)
	if _, err := client.Me(ctx); err != nil {
		return fmt.Errorf("CLI token invalid: %w (run: mcpzero logout && mcpzero login)", err)
	}

	if strings.HasPrefix(endpointID, "ep_") {
		endpoints, err := client.ListEndpoints(ctx)
		if err != nil {
			return fmt.Errorf("list endpoints: %w", err)
		}
		if !endpointOwned(endpoints, endpointID) {
			return fmt.Errorf(
				"endpoint %q not found in your account (run: mcpzero whoami --limits)",
				endpointID,
			)
		}
	}

	return nil
}
