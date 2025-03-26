package nix

import _ "embed"

//go:embed tmpl.nix
var pkgTmpl []byte

type archive struct {
	URL, Sha string
}

type templateData struct {
	Name         string
	Version      string
	Install      []string
	PostInstall  []string
	SourceRoot   string
	SourceRoots  map[string]string
	Archives     map[string]archive
	Description  string
	Homepage     string
	License      string
	Platforms    []string
	Inputs       []string
	Dependencies []string
}
