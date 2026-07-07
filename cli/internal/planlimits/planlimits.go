package planlimits

import "strings"

// Display helpers mirroring saas/packages/shared/src/plan-limits.ts and plan-limit.md.

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
	case "team", "enterprise":
		return 10
	default:
		return 1
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

func MaxToolsPerTunnel(plan string) int {
	switch normalizePlan(plan) {
	case "team":
		return 200
	case "enterprise":
		return -1
	default:
		return 50
	}
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

func FormatToolsLimit(plan string) string {
	n := MaxToolsPerTunnel(plan)
	if n < 0 {
		return "unlimited"
	}
	return stringInt(n)
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
