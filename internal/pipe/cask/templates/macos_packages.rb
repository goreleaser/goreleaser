{{- define "macos_packages" }}
{{- range $element := .MacOSPackages }}
  {{- if eq $element.Arch "all" }}
  url "{{ $element.URL.Download }}"{{- include "additional_url_params" $element.URL }}
  sha256 "{{ $element.SHA256 }}"
  {{- if .Binary }}
  binary "{{ .Name }}", target: "{{ .Binary }}"
  {{- end }}

  {{- else }}
  {{- if eq $element.Arch "amd64" }}
  on_intel do
  {{- end }}
  {{- if eq $element.Arch "arm64" }}
  on_arm do
  {{- end }}
    url "{{ $element.URL.Download }}"{{- include "additional_url_params" $element.URL }}
    sha256 "{{ $element.SHA256 }}"
    {{- if .Binary }}
    binary "{{ .Name }}", target: "{{ .Binary }}"
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
