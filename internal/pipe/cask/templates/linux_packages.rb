{{- define "linux_packages" }}
{{- range $element := .LinuxPackages }}
  {{- if eq $element.Arch "amd64" }}
  if Hardware::CPU.intel? and Hardware::CPU.is_64_bit?
  {{- else if eq $element.Arch "arm64" }}
  if Hardware::CPU.arm? and Hardware::CPU.is_64_bit?
  {{- else if eq $element.Arch "arm" }}
  if Hardware::CPU.arm? and !Hardware::CPU.is_64_bit?
  {{- end }}
    url "{{ $element.DownloadURL }}"
    sha256 "{{ $element.SHA256 }}"
  end
{{- end }}
{{- end }}
