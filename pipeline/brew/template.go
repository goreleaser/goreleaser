package brew

import "github.com/goreleaser/goreleaser/config"

type templateData struct {
	Name         string
	Desc         string
	Homepage     string
	Repo         config.Repo // FIXME: will not work for anything but github right now.
	Tag          string
	Version      string
	Caveats      string
	File         string
	SHA256       string
	Plist        string
	Install      []string
	Dependencies []string
	Conflicts    []string
	Tests        []string
}

const formulaTemplate = `class {{ .Name }} < Formula
  desc "{{ .Desc }}"
  homepage "{{ .Homepage }}"
  url "https://github.com/{{ .Repo.Owner }}/{{ .Repo.Name }}/releases/download/{{ .Tag }}/{{ .File }}"
  version "{{ .Version }}"
  sha256 "{{ .SHA256 }}"

  {{- if .Dependencies }}

  {{ range $index, $element := .Dependencies -}}
  depends_on "{{ . }}"
  {{- end }}
  {{- end -}}

  {{- if .Conflicts }}
  {{ range $index, $element := .Conflicts -}}
  conflicts_with "{{ . }}"
  {{- end }}
  {{- end }}

  def install
    {{- range $index, $element := .Install }}
    {{ . -}}
    {{- end }}
  end

  {{- if .Caveats }}

  def caveats
    "{{ .Caveats }}"
  end
  {{- end -}}

  {{- if .Plist }}

  plist_options :startup => false

  def plist; <<-EOS.undent
    {{ .Plist }}
    EOS
  end
  {{- end -}}

  {{- if .Tests }}

  test do
    {{- range $index, $element := .Tests }}
    {{ . -}}
    {{- end }}
  end
  {{- end }}
end
`
