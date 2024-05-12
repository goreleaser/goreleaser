{{- define "linux_packages" }}
{{- range $element := .LinuxPackages }}
  {{- if eq $element.Arch "amd64" }}
  on_intel do
    if Hardware::CPU.is_64_bit?
  {{- end }}
  {{- if or (eq $element.Arch "arm") (eq $element.Arch "arm64") }}
  on_arm do
  {{- if eq $element.Arch "arm64" }}
    if Hardware::CPU.is_64_bit?
  {{- else }}
    if !Hardware::CPU.is_64_bit?
  {{- end }}
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
  end
{{- end }}
{{- end }}
