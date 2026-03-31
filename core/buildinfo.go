package core

import "runtime/debug"

var Version = "v0.0.0-devel"
var CommitHash = ""
var BuildTime = "0000-00-00T00:00:00Z"

type BuildInfo struct {
	Version    string
	CommitHash string
	BuildTime  string
	Module     string
	Package    string
	Compiler   string
}

func ReadBuildInfo(version, commitHash, buildTime string) BuildInfo {
	info := BuildInfo{
		Version:    version,
		CommitHash: commitHash,
		BuildTime:  buildTime,
	}

	if build, ok := debug.ReadBuildInfo(); ok {
		info.Module = build.Main.Path
		info.Package = build.Path
		info.Compiler = build.GoVersion
	}

	return info
}

func CurrentBuildInfo() BuildInfo {
	return ReadBuildInfo(Version, CommitHash, BuildTime)
}
