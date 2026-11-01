package main

import (
	_ "embed"
	"strings"
)

// The version is X.Y.Z-BUILD. X.Y.Z comes from the VERSION file, and CI sets
// build to the pipeline number at link time.
//
//go:embed VERSION
var versionFile string

var build = "dev"

var version = strings.TrimSpace(versionFile) + "-" + build
