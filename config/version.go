package config

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of Logstash Exporter.
	Version = "unknown"

	// GitCommit is the git commit hash of the current build.
	GitCommit = "unknown"

	// BuildDate is the date of the current build.
	BuildDate = "unknown"
)

// GetVersionInfo returns a VersionInfo struct with the current build information.
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		GoVersion: runtime.Version(),
		BuildArch: runtime.GOARCH,
		BuildOS:   runtime.GOOS,
		BuildDate: BuildDate,
	}
}

// VersionInfo contains the current build information.
type VersionInfo struct {
	Version   string
	GitCommit string
	GoVersion string
	BuildArch string
	BuildOS   string
	BuildDate string
}

// String returns a string representation of the VersionInfo struct.
func (v *VersionInfo) String() string {
	return fmt.Sprintf("Version: %s, GitCommit: %s, GoVersion: %s, BuildArch: %s, BuildOS: %s, BuildDate: %s", v.Version, v.GitCommit, v.GoVersion, v.BuildArch, v.BuildOS, v.BuildDate)
}
