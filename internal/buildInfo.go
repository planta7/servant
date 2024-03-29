// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package internal

type BuildInfo struct {
	Version string
	Commit  string
}

func (b *BuildInfo) GetShortCommit() string {
	if len(b.Commit) >= 7 {
		return b.Commit[0:7]
	}
	return b.Commit
}

var ServantInfo *BuildInfo

func SetBuildInfo(version string, commit string) {
	ServantInfo = &BuildInfo{
		Version: version,
		Commit:  commit,
	}
}
