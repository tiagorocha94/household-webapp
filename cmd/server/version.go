package main

import "runtime/debug"

// version is set at build time with -ldflags "-X main.version=v0.3.0" (see
// the Makefile). Without it, buildVersion falls back to the Go build info.
var version = "dev"

// buildVersion returns the release version, or the short commit the binary
// was built from, or "dev" when neither is known.
func buildVersion() string {
	return versionFrom(version, debug.ReadBuildInfo)
}

func versionFrom(set string, readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if set != "" && set != "dev" {
		return set
	}
	info, ok := readBuildInfo()
	if !ok {
		return "dev"
	}
	var revision, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if revision == "" {
		return "dev"
	}
	revision = revision[:min(len(revision), 7)]
	if modified == "true" {
		revision += "-dirty"
	}
	return "dev-" + revision
}
