{{- define "macos_packages" }}
{{- if eq $.SingleMacOSArch "arm64" }}
  depends_on arch: :arm64
{{- end }}
{{- range $element := .MacOSPackages }}
{{- if or (eq $element.Arch "all") (eq $.SingleMacOSArch $element.Arch) }}
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

{{- else }}
  {{- if eq $element.Arch "amd64" }}
  on_intel do
  {{- end }}
  {{- if eq $element.Arch "arm64" }}
  on_arm do
  {{- end }}
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

  {{- if $.HasOnlyAmd64MacOsPkg }}
  on_arm do
    def caveats
      <<~EOS
        The darwin_arm64 architecture is not supported for the {{ $.Name }}
        formula at this time. The darwin_amd64 binary may work in compatibility
        mode, but it might not be fully supported.
      EOS
    end
  end
  {{- end }}
{{- end }}
{{- end }}
