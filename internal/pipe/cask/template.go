package cask

import (
	"embed"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

type templateData struct {
	config.HomebrewCask
	Name                 string
	Version              string
	LinuxPackages        []releasePackage
	MacOSPackages        []releasePackage
	HasOnlyAmd64MacOsPkg bool
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
