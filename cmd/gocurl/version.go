package main

import (
	"fmt"
	"runtime"
)

// Version information - these are set via ldflags during build
var (
	version = "dev"     // Version tag from git (e.g., "v1.1.0" or "dev")
	commit  = "unknown" // Git commit SHA (short)
	date    = "unknown" // Build date in ISO 8601 format
)

// GetVersion returns the version string
func GetVersion() string {
	return version
}

// GetVersionInfo returns formatted version information (one line)
func GetVersionInfo() string {
	return fmt.Sprintf("gocurl %s (commit: %s, built: %s)", version, commit, date)
}

// GetVersionDetails returns detailed version information including runtime
func GetVersionDetails() string {
	return fmt.Sprintf(
		"gocurl %s\nCommit: %s\nBuilt: %s\nGo version: %s\nOS/Arch: %s/%s",
		version, commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH,
	)
}
