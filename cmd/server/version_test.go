package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionFrom(t *testing.T) {
	t.Parallel()

	info := func(settings ...debug.BuildSetting) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Settings: settings}, true }
	}
	none := func() (*debug.BuildInfo, bool) { return nil, false }

	tests := []struct {
		name string
		set  string
		read func() (*debug.BuildInfo, bool)
		want string
	}{
		{"release set by ldflags", "v0.3.0", none, "v0.3.0"},
		{"no build info", "dev", none, "dev"},
		{"no revision", "dev", info(), "dev"},
		{"revision", "dev", info(debug.BuildSetting{Key: "vcs.revision", Value: "0123456789abcdef"}), "dev-0123456"},
		{"modified tree", "", info(
			debug.BuildSetting{Key: "vcs.revision", Value: "0123456789abcdef"},
			debug.BuildSetting{Key: "vcs.modified", Value: "true"},
		), "dev-0123456-dirty"},
		{"short revision", "dev", info(debug.BuildSetting{Key: "vcs.revision", Value: "abc"}), "dev-abc"},
	}
	for _, tt := range tests {
		if got := versionFrom(tt.set, tt.read); got != tt.want {
			t.Errorf("%s: versionFrom() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
