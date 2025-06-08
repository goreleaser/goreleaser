package cask

import (
	"cmp"
	"embed"
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
}

type releasePackage struct {
	SHA256 string
	OS     string
	Arch   string
	URL    downloadURL
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
	indent := strings.Repeat(" ", 2+len("depends_on: "))
	var sb strings.Builder
	sb.WriteString("depends_on: ")
	for i, cask := range casks {
		if i == 0 {
			sb.WriteString("cask: [\n")
		}
		sb.WriteString(indent + "  " + cask)
		if len(casks)-1 > i {
			sb.WriteByte(',')
			sb.WriteByte('\n')
		}
	}

	if len(casks) > 0 {
		sb.WriteString("\n" + indent + "]")
	}
	if len(casks) > 0 && len(formulas) > 0 {
		sb.WriteByte(',')
		sb.WriteByte('\n')
		sb.WriteString(indent)
	}
	for i, form := range formulas {
		if i == 0 {
			sb.WriteString("formula: [\n")
		}
		sb.WriteString(indent + "  " + form)
		if len(formulas)-1 > i {
			sb.WriteByte(',')
			sb.WriteByte('\n')
		}
	}
	if len(formulas) > 0 {
		sb.WriteString("\n" + indent + "]")
	}
	return sb.String()
}

func conflictsString(conflicts []config.HomebrewCaskConflict) string {
	var casks []string
	var formulas []string
	for _, conflict := range conflicts {
		if conflict.Cask != "" {
			casks = append(casks, conflict.Cask)
		}
		if conflict.Formula != "" {
			formulas = append(formulas, conflict.Formula)
		}
	}
	sort.Strings(casks)
	sort.Strings(formulas)
	indent := strings.Repeat(" ", 2+len("conflicts_with: "))
	var sb strings.Builder
	sb.WriteString("conflicts_with: ")
	for i, cask := range casks {
		if i == 0 {
			sb.WriteString("cask: [\n")
		}
		sb.WriteString(indent + "  " + cask)
		if len(casks)-1 > i {
			sb.WriteByte(',')
			sb.WriteByte('\n')
		}
	}

	if len(casks) > 0 {
		sb.WriteString("\n" + indent + "]")
	}
	if len(casks) > 0 && len(formulas) > 0 {
		sb.WriteByte(',')
		sb.WriteByte('\n')
		sb.WriteString(indent)
	}
	for i, form := range formulas {
		if i == 0 {
			sb.WriteString("formula: [\n")
		}
		sb.WriteString(indent + "  " + form)
		if len(casks)-1 > i {
			sb.WriteByte(',')
			sb.WriteByte('\n')
		}
	}
	if len(formulas) > 0 {
		sb.WriteString("\n" + indent + "]")
	}
	return sb.String()
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
