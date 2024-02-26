{{- define "linux_packages" }}
{{- range $element := .LinuxPackages }}
  {{- if eq $element.Arch "amd64" }}
  if Hardware::CPU.intel?
  {{- end }}
  {{- if eq $element.Arch "arm" }}
  if Hardware::CPU.arm? && !Hardware::CPU.is_64_bit?
  {{- end }}
  {{- if eq $element.Arch "arm64" }}
  if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
  {{- end }}
    url "{{ $element.DownloadURL }}"
    {{- if .DownloadStrategy }}, using: {{ .DownloadStrategy }}{{- end }}
    {{- if .Headers }},
      headers: [{{ printf "\n" }}
        {{- join .Headers | indent 10 }}
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
