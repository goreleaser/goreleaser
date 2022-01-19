package aur

type templateData struct {
	Name            string
	Desc            string
	Homepage        string
	Version         string
	License         string
	ReleasePackages []releasePackage
	Maintainer      string
	Contributors    []string
	Provides        []string
	Conflicts       []string
	Depends         []string
	OptDepends      []string
	Arches          []string
	Rel             string
	Package         string
}

type releasePackage struct {
	DownloadURL string
	SHA256      string
	Arch        string
}

const pkgBuildTemplate = `# This file was generated by GoReleaser. DO NOT EDIT.

{{- with .Maintainer }}
# Maintainer: {{ . }}
{{- end }}
{{- range .Contributors }}
# Contributor: {{ . }}
{{- end }}

pkgname='{{ .Name }}'
pkgver={{ .Version }}
pkgrel={{ .Rel }}
pkgdesc='{{ .Desc }}'
url='{{ .Homepage }}'
arch=({{ pkgArray .Arches }})
license=('{{ .License }}')
{{- with .Provides }}
provides=({{ pkgArray . }})
{{- end }}
{{- with .Conflicts }}
conflicts=({{ pkgArray . }})
{{- end }}
{{- with .Depends }}
depends=({{ pkgArray . }})
{{- end }}
{{- with .OptDepends }}
optdepends=({{ pkgArray . }})
{{- end }}

{{ range .ReleasePackages -}}
source_{{ .Arch }}=('{{ .DownloadURL }}')
sha256sums_{{ .Arch }}=('{{ .SHA256 }}')
{{ printf "" }}
{{ end }}

{{-  with .Package -}}
package() {
{{ fixLines . }}
}
{{ end }}`

const srcInfoTemplate = `pkgbase = {{ .Name }}
	pkgdesc = {{ .Desc }}
	pkgver = {{ .Version }}
	pkgrel = {{ .Rel }}
	url = {{ .Homepage }}
	license = {{ .License }}
	{{ range .OptDepends -}}
	optdepends = {{ . }}
	{{ end }}
	{{ range .Depends -}}
	depends = {{ . }}
	{{ end }}
	{{ range .Conflicts -}}
	conflicts = {{ . }}
	{{ end }}
	{{ range .Provides -}}
	provides = {{ . }}
	{{ end }}
	{{ range .ReleasePackages -}}
	arch = {{ .Arch }}
	source_{{ .Arch }} = {{ .DownloadURL }}
	sha256sums_{{ .Arch }} = {{ .SHA256 }}
	{{ end}}

pkgname = {{ .Name }}
`
