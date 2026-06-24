// Package version holds the CLI build version.
package version

// Version is the CLI build version. It is overridden at build time via
// -ldflags "-X github.com/mcpzero/mcpzero/cli/internal/version.Version=<v>".
// It must be a var (not const): the linker's -X flag cannot rewrite constants.
var Version = "0.0.0-dev"
