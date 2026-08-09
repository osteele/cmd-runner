package main

import (
	"runtime/debug"
	"strings"
)

// version is set for release builds with -ldflags "-X main.version=<version>".
var version string

func currentVersion() string {
	if version != "" {
		return strings.TrimPrefix(version, "v")
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}
	return versionFromBuildInfo(buildInfo)
}

func versionFromBuildInfo(buildInfo *debug.BuildInfo) string {
	if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		return strings.TrimPrefix(buildInfo.Main.Version, "v")
	}

	var revision string
	modified := false
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if revision == "" {
		return "devel"
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	result := "devel+" + revision
	if modified {
		result += ".dirty"
	}
	return result
}
