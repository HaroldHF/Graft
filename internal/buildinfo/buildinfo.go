// Package buildinfo exposes compile-time version metadata for Graft binaries.
package buildinfo

import "fmt"

// Version, Commit, and BuildTime can be overridden at link time via -ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

// Summary returns a single-line human-readable build identifier for component.
func Summary(component string) string {
	return fmt.Sprintf("%s version=%s commit=%s built=%s", component, Version, Commit, BuildTime)
}
