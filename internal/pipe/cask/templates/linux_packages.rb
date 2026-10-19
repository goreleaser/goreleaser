{{- define "linux_packages" }}
{{- if eq $.SingleLinuxArch "amd64" }}
  depends_on arch: :x86_64
{{- else if eq $.SingleLinuxArch "arm64" }}
  depends_on arch: :arm64
{{- end }}
{{- range $element := .LinuxPackages }}
{{- if or (eq $element.Arch "all") (eq $.SingleLinuxArch $element.Arch) }}
  sha256 "{{ $element.SHA256 }}"
  url "{{ $element.URL.Download }}"{{- include "additional_url_params" $element.URL }}
  {{- if .Binary }}
  binary "{{ .Name }}", target: "{{ .Binary }}"
  {{- end }}
  {{- if .WrappedIn }}
  {{- $wrap := .WrappedIn }}
  {{- range .Wrapped }}
  rename "{{ $wrap }}/{{ . }}", "{{ . }}"
  {{- end }}
  {{- end }}
  {{- if $.HasMixedPackageTypes }}
  {{- range .CaskBins }}
  binary "{{ . }}"
  {{- end }}
  {{- end }}
{{- else if eq $element.Arch "amd64" }}
  on_intel do
    sha256 "{{ $element.SHA256 }}"
    url "{{ $element.URL.Download }}"{{- include "additional_url_params" $element.URL }}
    {{- if .Binary }}
    binary "{{ .Name }}", target: "{{ .Binary }}"
    {{- end }}
    {{- if .WrappedIn }}
    {{- $wrap := .WrappedIn }}
    {{- range .Wrapped }}
    rename "{{ $wrap }}/{{ . }}", "{{ . }}"
    {{- end }}
    {{- end }}
    {{- if $.HasMixedPackageTypes }}
    {{- range .CaskBins }}
    binary "{{ . }}"
    {{- end }}
    {{- end }}
  end
{{- else if eq $element.Arch "arm64" }}
  on_arm do
    sha256 "{{ $element.SHA256 }}"
    url "{{ $element.URL.Download }}"{{- include "additional_url_params" $element.URL }}
    {{- if .Binary }}
    binary "{{ .Name }}", target: "{{ .Binary }}"
    {{- end }}
    {{- if .WrappedIn }}
    {{- $wrap := .WrappedIn }}
    {{- range .Wrapped }}
    rename "{{ $wrap }}/{{ . }}", "{{ . }}"
    {{- end }}
    {{- end }}
    {{- if $.HasMixedPackageTypes }}
    {{- range .CaskBins }}
    binary "{{ . }}"
    {{- end }}
    {{- end }}
  end
{{- end }}
{{- end }}
{{- end }}
