{{- define "linux_packages" }}
{{- range $element := .LinuxPackages }}
  {{- if eq $element.Arch "amd64" }}
  on_intel do
  {{- else if eq $element.Arch "arm64" }}
  on_arm do
  {{- end }}
    url "{{ $element.DownloadURL }}"{{- include "additional_url_params" $element }}
    sha256 "{{ $element.SHA256 }}"
  end
{{- end }}
{{- end }}
