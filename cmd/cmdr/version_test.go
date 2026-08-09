package main

import (
	"runtime/debug"
	"testing"
)

func TestCurrentVersionUsesReleaseOverride(t *testing.T) {
	previousVersion := version
	version = "v1.2.3"
	t.Cleanup(func() { version = previousVersion })

	if got := currentVersion(); got != "1.2.3" {
		t.Fatalf("currentVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestVersionFromBuildInfo(t *testing.T) {
	tests := []struct {
		name      string
		buildInfo debug.BuildInfo
		want      string
	}{
		{
			name:      "module version",
			buildInfo: debug.BuildInfo{Main: debug.Module{Version: "v2.3.4"}},
			want:      "2.3.4",
		},
		{
			name: "revision",
			buildInfo: debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "0123456789abcdef"},
			}},
			want: "devel+0123456789ab",
		},
		{
			name: "dirty revision",
			buildInfo: debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "0123456789abcdef"},
				{Key: "vcs.modified", Value: "true"},
			}},
			want: "devel+0123456789ab.dirty",
		},
		{
			name: "no metadata",
			want: "devel",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := versionFromBuildInfo(&test.buildInfo); got != test.want {
				t.Fatalf("versionFromBuildInfo() = %q, want %q", got, test.want)
			}
		})
	}
}
