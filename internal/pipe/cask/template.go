package cask

import (
	"embed"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

type templateData struct {
	Name                 string
	Desc                 string
	Homepage             string
	Version              string
	License              string
	Caveats              []string
	PostFlight           []string
	Dependencies         []config.HomebrewCaskDependency
	Conflicts            []config.HomebrewCaskConflict
	CustomRequire        string
	CustomBlock          []string
	LinuxPackages        []releasePackage
	MacOSPackages        []releasePackage
	Service              string
	HasOnlyAmd64MacOsPkg bool
	Binary               string
	Zap                  []string
	Manpage              string
	BashCompletions      string
	ZshCompletions       string
	FishCompletions      string
}

type releasePackage struct {
	DownloadURL      string
	SHA256           string
	OS               string
	Arch             string
	DownloadStrategy string
	Headers          []string
}

//go:embed templates
var templates embed.FS
