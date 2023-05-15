package nix

import _ "embed"

//go:embed tmpl.nix
var pkgTmpl []byte

type Archive struct {
	URL, Sha string
}

type TemplateData struct {
	Name       string
	Version    string
	Install    string
	SourceRoot string
	Archives   map[string]Archive
}
