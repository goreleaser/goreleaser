package cask

import (
	"cmp"
	"embed"
	"strings"

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
	DownloadURL   string
	SHA256        string
	OS            string
	Arch          string
	URLAdditional config.HomebrewCaskURLAdditionalParameters
}

//go:embed templates
var templates embed.FS

func split(s string) []string {
	strings := strings.Split(strings.TrimSpace(s), "\n")
	if len(strings) == 1 && strings[0] == "" {
		return []string{}
	}
	return strings
}

func zapString(u config.HomebrewCaskUninstall) string {
	return cmp.Or(makeUninstallLikeBlock("zap", u), "# No zap stanza required")
}

func uninstallString(u config.HomebrewCaskUninstall) string {
	return makeUninstallLikeBlock("uninstall", u)
}

func makeUninstallLikeBlock(stanza string, u config.HomebrewCaskUninstall) string {
	groups := []string{}
	if len(u.Launchctl) > 0 {
		groups = append(groups, groupToS("launchctl", u.Launchctl))
	}
	if len(u.Quit) > 0 {
		groups = append(groups, groupToS("quit", u.Quit))
	}
	if len(u.LoginItem) > 0 {
		groups = append(groups, groupToS("login_item", u.LoginItem))
	}
	if len(u.Delete) > 0 {
		groups = append(groups, groupToS("delete", u.Delete))
	}
	if len(u.Trash) > 0 {
		groups = append(groups, groupToS("trash", u.Trash))
	}
	if len(groups) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(stanza + " ")
	for i, group := range groups {
		if i > 0 {
			sb.WriteString("    ")
		}
		sb.WriteString(group)
		if len(groups)-1 > i {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func groupToS(name string, lines []string) string {
	var sb strings.Builder
	sb.WriteString(name + ": [\n")
	for _, l := range lines {
		sb.WriteString("      " + l + ",\n")
	}
	sb.WriteString("    ]")
	return sb.String()
}
