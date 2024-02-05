package config

import (
	"runtime"
	"testing"
)

func TestGetBuildInfo(t *testing.T) {
	versionInfo := GetVersionInfo()

	if versionInfo.Version == "" {
		t.Error("expected Version to be set")
	}

	if versionInfo.SemanticVersion == "" {
		t.Error("expected SemanticVersion to be set")
	}

	if versionInfo.GitCommit == "" {
		t.Error("expected GitCommit to be set")
	}

	if versionInfo.GoVersion != runtime.Version() {
		t.Errorf("expected GoVersion: %s, but got: %s", runtime.Version(), versionInfo.GoVersion)
	}

	if versionInfo.BuildArch != runtime.GOARCH {
		t.Errorf("expected BuildArch: %s, but got: %s", runtime.GOARCH, versionInfo.BuildArch)
	}

	if versionInfo.BuildOS != runtime.GOOS {
		t.Errorf("expected BuildOS: %s, but got: %s", runtime.GOOS, versionInfo.BuildOS)
	}

	if versionInfo.BuildDate == "" {
		t.Error("expected BuildDate to be set")
	}
}

func TestVersionInfoString(t *testing.T) {
	versionInfo := &VersionInfo{
		Version:         "test-version",
		SemanticVersion: "v0.0.1-0548a52",
		GitCommit:       "test-commit",
		GoVersion:       "test-go-version",
		BuildArch:       "test-arch",
		BuildOS:         "test-os",
		BuildDate:       "test-date",
	}

	expectedString := "Version: test-version, SemanticVersion: v0.0.1-0548a52, GitCommit: test-commit, GoVersion: test-go-version, BuildArch: test-arch, BuildOS: test-os, BuildDate: test-date"
	versionInfoString := versionInfo.String()

	if versionInfoString != expectedString {
		t.Errorf("expected string: %s, but got: %s", expectedString, versionInfoString)
	}
}
