package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mcpzero/mcpzero/cli/internal/cloud"
)

type initResolveInput struct {
	Endpoints  []cloud.Endpoint
	EndpointID string // --endpoint: reuse this id
	ForceNew   bool   // --new
	TeamID     string // --team-id: prefer endpoints in this team when reusing
}

type initResolveResult struct {
	Endpoint   *cloud.Endpoint
	CreateNew  bool
	ReuseNote  string
}

// resolveInitEndpoint decides whether init should create a new endpoint or reuse an existing one.
func resolveInitEndpoint(in initResolveInput) (initResolveResult, error) {
	endpointID := strings.TrimSpace(in.EndpointID)
	teamID := strings.TrimSpace(in.TeamID)

	if endpointID != "" {
		for i := range in.Endpoints {
			if in.Endpoints[i].ID == endpointID {
				ep := in.Endpoints[i]
				return initResolveResult{
					Endpoint:  &ep,
					CreateNew: false,
					ReuseNote: fmt.Sprintf("Using existing endpoint %s (%s)", ep.ID, ep.Name),
				}, nil
			}
		}
		return initResolveResult{}, fmt.Errorf("endpoint %q not found in your account", endpointID)
	}

	if in.ForceNew {
		return initResolveResult{CreateNew: true}, nil
	}

	candidates := filterEndpointsForInit(in.Endpoints, teamID)
	if len(candidates) > 0 {
		ep := newestEndpoint(candidates)
		note := fmt.Sprintf("Using existing endpoint %s (%s)", ep.ID, ep.Name)
		if len(in.Endpoints) > 1 {
			note += " — pass --endpoint to choose another, or --new to create one"
		}
		return initResolveResult{
			Endpoint:  &ep,
			CreateNew: false,
			ReuseNote: note,
		}, nil
	}

	return initResolveResult{CreateNew: true}, nil
}

// filterEndpointsForInit narrows endpoints for reuse. Personal endpoints are preferred
// when no --team-id is set (typical free/personal init flow).
func filterEndpointsForInit(endpoints []cloud.Endpoint, teamID string) []cloud.Endpoint {
	if teamID != "" {
		var out []cloud.Endpoint
		for _, ep := range endpoints {
			if ep.TeamID != nil && *ep.TeamID == teamID {
				out = append(out, ep)
			}
		}
		return out
	}

	var personal []cloud.Endpoint
	for _, ep := range endpoints {
		if ep.TeamID == nil || *ep.TeamID == "" {
			personal = append(personal, ep)
		}
	}
	if len(personal) > 0 {
		return personal
	}
	return endpoints
}

func formatEndpointLimitError(err error, endpoints []cloud.Endpoint) error {
	if !cloud.IsAPIError(err, "endpoint_limit_reached") && !cloud.IsAPIError(err, "team_endpoint_limit_reached") {
		return err
	}
	msg := "endpoint limit reached for your plan"
	if len(endpoints) > 0 {
		ep := newestEndpoint(endpoints)
		msg = fmt.Sprintf(
			"%s — you already have %s (%s). Re-run without --new to reuse it, or pass --endpoint %s",
			msg, ep.ID, ep.Name, ep.ID,
		)
	} else {
		msg += " — re-run without --new if you already created one in the dashboard"
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func newestEndpoint(endpoints []cloud.Endpoint) cloud.Endpoint {
	if len(endpoints) == 1 {
		return endpoints[0]
	}
	sorted := append([]cloud.Endpoint(nil), endpoints...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt > sorted[j].CreatedAt
	})
	return sorted[0]
}
