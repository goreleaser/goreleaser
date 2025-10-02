{{- define "linux_packages" }}
{{- range $element := .LinuxPackages }}
  {{- if eq $element.Arch "amd64" }}
  on_intel do
  {{- else if eq $element.Arch "arm64" }}
  on_arm do
  {{- end }}
    url "{{ $element.URL.Download }}"{{- include "additional_url_params" $element.URL }}
    sha256 "{{ $element.SHA256 }}"
    {{- if .Binary }}
    binary "{{ .Name }}", target: "{{ .Binary }}"
    {{- end }}
  end
{{- end }}
{{- end }}
