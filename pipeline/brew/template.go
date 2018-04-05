package brew

import "github.com/goreleaser/goreleaser/config"

type templateData struct {
	Name              string
	Desc              string
	Homepage          string
	DownloadURL       string
	Repo              config.Repo // FIXME: will not work for anything but github right now.
	Tag               string
	Version           string
	Caveats           []string
	File              string
	SHA256            string
	Plist             string
	DownloadStrategy  string
	Install           []string
	Dependencies      []string
	BuildDependencies []string
	Conflicts         []string
	Tests             []string
}

const formulaTemplate = `class {{ .Name }} < Formula
  desc "{{ .Desc }}"
  homepage "{{ .Homepage }}"
  {{ if .BuildDependencies -}}
  url "{{ .DownloadURL }}/{{ .Repo.Owner }}/{{ .Repo.Name }}/archive/{{ .Tag }}.tar.gz"
  head "https://github.com/{{ .Repo.Owner }}/{{ .Repo.Name }}.git"
  {{- else -}}
  url "{{ .DownloadURL }}/{{ .Repo.Owner }}/{{ .Repo.Name }}/releases/download/{{ .Tag }}/{{ .File }}"
  {{- if .DownloadStrategy }}, :using => {{ .DownloadStrategy }}{{- end }}
  {{- end }}
  version "{{ .Version }}"
  sha256 "{{ .SHA256 }}"

  {{- with .Dependencies }}
  {{ range $index, $element := . }}
  depends_on "{{ . }}"
  {{- end }}
  {{- end -}}
  {{- with .BuildDependencies -}}
  {{ range $index, $element := . }}
  depends_on "{{ . }}" => :build
  {{- end }}
  {{- end -}}

  {{- with .Conflicts }}
  {{ range $index, $element := . }}
  conflicts_with "{{ . }}"
  {{- end }}
  {{- end }}

  def install
    {{- range $index, $element := .Install }}
    {{ . -}}
    {{- end }}
  end

  {{- with .Caveats }}

  def caveats; <<~EOS
    {{- range $index, $element := . }}
    {{ . -}}
    {{- end }}
  EOS
  end
  {{- end -}}

  {{- with .Plist }}

  plist_options :startup => false

  def plist; <<~EOS
    {{ . }}
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
