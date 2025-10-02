package cask

import (
	"cmp"
	"embed"
	"fmt"
	"sort"
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
	HasOnlyBinaryPkgs    bool
}

type releasePackage struct {
	SHA256 string
	OS     string
	Arch   string
	URL    downloadURL
	Name   string
	Binary string
}

type downloadURL struct {
	Download  string
	Verified  string
	Using     string
	Cookies   map[string]string
	Referer   string
	Headers   []string
	UserAgent string
	Data      map[string]string
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

func dependsString(dependencies []config.HomebrewCaskDependency) string {
	var casks []string
	var formulas []string
	for _, dependency := range dependencies {
		if dependency.Cask != "" {
			casks = append(casks, dependency.Cask)
		}
		if dependency.Formula != "" {
			formulas = append(formulas, dependency.Formula)
		}
	}
	sort.Strings(casks)
	sort.Strings(formulas)

	var groups []string
	if len(casks) > 0 {
		groups = append(groups, groupToS("cask", casks))
	}
	if len(formulas) > 0 {
		groups = append(groups, groupToS("formula", formulas))
	}
	return joinGroups("depends_on", groups)
}

func conflictsString(conflicts []config.HomebrewCaskConflict) string {
	var casks []string
	for _, conflict := range conflicts {
		if conflict.Cask != "" {
			casks = append(casks, conflict.Cask)
		}
	}
	sort.Strings(casks)
	var groups []string
	if len(casks) > 0 {
		groups = append(groups, groupToS("cask", casks))
	}
	return joinGroups("conflicts_with", groups)
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
	return joinGroups(stanza, groups)
}

func joinGroups(stanza string, groups []string) string {
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
		sb.WriteString("      " + fmt.Sprintf("%q", l) + ",\n")
	}
	sb.WriteString("    ]")
	return sb.String()
}
