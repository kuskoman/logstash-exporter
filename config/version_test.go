package config

import (
	"runtime"
	"testing"
)

func TestGetBuildInfo(t *testing.T) {
	versionInfo := GetVersionInfo()

	if versionInfo.Version == "" {
		t.Error("Expected Version to be set")
	}

	if versionInfo.SemanticVersion == "" {
		t.Error("Expected SemanticVersion to be set")
	}

	if versionInfo.GitCommit == "" {
		t.Error("Expected GitCommit to be set")
	}

	if versionInfo.GoVersion != runtime.Version() {
		t.Errorf("Expected GoVersion: %s, but got: %s", runtime.Version(), versionInfo.GoVersion)
	}

	if versionInfo.BuildArch != runtime.GOARCH {
		t.Errorf("Expected BuildArch: %s, but got: %s", runtime.GOARCH, versionInfo.BuildArch)
	}

	if versionInfo.BuildOS != runtime.GOOS {
		t.Errorf("Expected BuildOS: %s, but got: %s", runtime.GOOS, versionInfo.BuildOS)
	}

	if versionInfo.BuildDate == "" {
		t.Error("Expected BuildDate to be set")
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
		t.Errorf("Expected string: %s, but got: %s", expectedString, versionInfoString)
	}
}
