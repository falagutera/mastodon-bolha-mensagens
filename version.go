package main

import (
	_ "embed"
	"strings"

	"gitlab.gutocarvalho.net/open/bolha/justmytoots/internal/relnotes"
)

// The version is X.Y.Z-BUILD. X.Y.Z comes from the VERSION file, and CI sets
// build to the pipeline number at link time.
//
//go:embed VERSION
var versionFile string

var build = "dev"

var version = strings.TrimSpace(versionFile) + "-" + build

// The release notes page and /release-notes.json come from CHANGELOG.md.
//
//go:embed CHANGELOG.md
var changelog string

var releases = relnotes.MustParse(changelog)
