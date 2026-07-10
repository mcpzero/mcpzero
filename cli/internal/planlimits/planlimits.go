package planlimits

import "strings"

// Display helpers mirroring saas/packages/shared/src/plan-limits.ts and plan-limit.md.
// -1 means unlimited.

func normalizePlan(plan string) string {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case "personal", "team", "enterprise":
		return strings.ToLower(strings.TrimSpace(plan))
	default:
		return "free"
	}
}

func MaxEndpoints(plan string) int {
	switch normalizePlan(plan) {
	case "personal":
		return 2
	case "team":
		return 10
	case "enterprise":
		return -1
	default:
		return 1
	}
}

func MaxServersPerEndpoint(plan string) int {
	switch normalizePlan(plan) {
	case "team":
		return 10
	case "enterprise":
		return -1
	default:
		return 5
	}
}

func RateLimitRPM(plan string) int {
	switch normalizePlan(plan) {
	case "team", "enterprise":
		return 60
	default:
		return 30
	}
}

func MaxTools(plan string) int {
	switch normalizePlan(plan) {
	case "team":
		return 200
	case "enterprise":
		return -1
	default:
		return 50
	}
}

// MaxToolsPerTunnel is kept for callers; Free is account-wide (see ToolsScope).
func MaxToolsPerTunnel(plan string) int {
	return MaxTools(plan)
}

func ToolsScope(plan string) string {
	if normalizePlan(plan) == "free" {
		return "account"
	}
	return "tunnel"
}

func ClustersEnabled(plan string) bool {
	p := normalizePlan(plan)
	return p == "team" || p == "enterprise"
}

func PayloadRetentionLabel(plan string) string {
	switch normalizePlan(plan) {
	case "personal":
		return "7 days"
	case "team", "enterprise":
		return "30 days"
	default:
		return "48 hours"
	}
}

func FormatLimit(n int) string {
	if n < 0 {
		return "unlimited"
	}
	return stringInt(n)
}

func FormatToolsLimit(plan string) string {
	return FormatLimit(MaxTools(plan))
}

func stringInt(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
