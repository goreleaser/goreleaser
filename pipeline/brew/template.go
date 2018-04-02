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
	Special           []string
}

const formulaTemplate = `class {{ .Name }} < Formula
  desc "{{ .Desc }}"
  homepage "{{ .Homepage }}"
  url "{{ .DownloadURL }}/{{ .Repo.Owner }}/{{ .Repo.Name }}/releases/download/{{ .Tag }}/{{ .File }}"
  {{- if .DownloadStrategy }}, :using => {{ .DownloadStrategy }}{{- end }}
  version "{{ .Version }}"
  sha256 "{{ .SHA256 }}"
  head "https://github.com/{{ .Repo.Owner }}/{{ .Repo.Name }}.git"

  {{- if .BuildDependencies }}
  {{ range $index, $element := .BuildDependencies }}
  depends_on "{{ . }}" => :build
  {{- end }}
  {{- end -}}

  {{- if .Dependencies }}
  {{ range $index, $element := .Dependencies }}
  depends_on "{{ . }}"
  {{- end }}
  {{- end -}}

  {{- if .Conflicts }}
  {{ range $index, $element := .Conflicts }}
  conflicts_with "{{ . }}"
  {{- end }}
  {{- end }}

  {{- if .Special }}
  {{- range $index, $element := .Special }}
  {{ . -}}
  {{- end -}}
  {{- end }}

  def install
    {{- range $index, $element := .Install }}
    {{ . -}}
    {{- end }}
  end

  {{- if .Caveats }}

  def caveats; <<-EOS.undent
    {{- range $index, $element := .Caveats }}
    {{ . -}}
    {{- end }}
  EOS
  end
  {{- end -}}

  {{- if .Plist }}

  plist_options :startup => false

  def plist; <<~EOS
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
