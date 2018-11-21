package brew

type templateData struct {
	Name             string
	Desc             string
	Homepage         string
	DownloadURL      string
	Version          string
	Caveats          []string
	SHA256           string
	Plist            string
	DownloadStrategy string
	Install          []string
	Dependencies     []string
	Conflicts        []string
	Tests            []string
	CustomRequire    string
}

const formulaTemplate = `{{ if .CustomRequire -}}
require_relative "{{ .CustomRequire }}"
{{ end -}}
class {{ .Name }} < Formula
  desc "{{ .Desc }}"
  homepage "{{ .Homepage }}"
  url "{{ .DownloadURL }}"
  {{- if .DownloadStrategy }}, :using => {{ .DownloadStrategy }}{{- end }}
  version "{{ .Version }}"
  sha256 "{{ .SHA256 }}"

  {{- with .Dependencies }}
  {{ range $index, $element := . }}
  depends_on "{{ . }}"
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
