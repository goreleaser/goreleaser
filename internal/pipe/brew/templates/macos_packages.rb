{{- define "macos_packages" }}
{{- range $element := .MacOSPackages }}
  {{- if eq $element.Arch "all" }}
  url "{{ $element.DownloadURL }}"
  {{- if .DownloadStrategy }}, using: {{ .DownloadStrategy }}{{- end }}
  {{- if .Headers }},
    headers: [{{ printf "\n" }}
      {{- join .Headers | indent 8 }}
    ]
  {{- end }}
  sha256 "{{ $element.SHA256 }}"

  def install
    {{- range $index, $element := .Install }}
    {{ . -}}
    {{- end }}
  end
  {{- else if $.HasOnlyAmd64MacOsPkg }}
  url "{{ $element.DownloadURL }}"
  {{- if .DownloadStrategy }}, using: {{ .DownloadStrategy }}{{- end }}
  {{- if .Headers }},
    headers: [{{ printf "\n" }}
      {{- join .Headers | indent 8 }}
    ]
  {{- end }}
  sha256 "{{ $element.SHA256 }}"

  def install
    {{- range $index, $element := .Install }}
    {{ . -}}
    {{- end }}
  end

  if Hardware::CPU.arm?
    def caveats
      <<~EOS
        The darwin_arm64 architecture is not supported for the {{ $.Name }}
        formula at this time. The darwin_amd64 binary may work in compatibility
        mode, but it might not be fully supported.
      EOS
    end
  end
  {{- else }}
  {{- if eq $element.Arch "amd64" }}
  if Hardware::CPU.intel?
  {{- end }}
  {{- if eq $element.Arch "arm64" }}
  if Hardware::CPU.arm?
  {{- end}}
    url "{{ $element.DownloadURL }}"
    {{- if .DownloadStrategy }}, using: {{ .DownloadStrategy }}{{- end }}
    {{- if .Headers }},
      headers: [{{ printf "\n" }}
        {{- join .Headers | indent 8 }}
      ]
    {{- end }}
    sha256 "{{ $element.SHA256 }}"

    def install
      {{- range $index, $element := .Install }}
      {{ . -}}
      {{- end }}
    end
  end
  {{- end }}
{{- end }}
{{- end }}
