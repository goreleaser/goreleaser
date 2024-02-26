package brew

import (
	"embed"

	"github.com/goreleaser/goreleaser/pkg/config"
)

type templateData struct {
	Name                 string
	Desc                 string
	Homepage             string
	Version              string
	License              string
	Caveats              []string
	Plist                string
	PostInstall          []string
	Dependencies         []config.HomebrewDependency
	Conflicts            []string
	Tests                []string
	CustomRequire        string
	CustomBlock          []string
	LinuxPackages        []releasePackage
	MacOSPackages        []releasePackage
	Service              []string
	HasOnlyAmd64MacOsPkg bool
}

type releasePackage struct {
	DownloadURL      string
	SHA256           string
	OS               string
	Arch             string
	DownloadStrategy string
	Install          []string
	Headers          []string
}

//go:embed templates
var formulaTemplate embed.FS
