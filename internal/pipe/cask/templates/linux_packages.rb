{{- define "linux_packages" }}
{{- range $element := .LinuxPackages }}
  {{- if eq $element.Arch "amd64" }}
  on_intel do
  {{- else if eq $element.Arch "arm64" }}
  on_arm do
  {{- end }}
    url "{{ $element.DownloadURL }}"
    {{- if $element.Using }},
      using: {{ $element.Using }}
    {{- end }}
    {{- if $element.Verified }},
      verified: "{{ $element.Verified }}"
    {{- end }}
    {{- if $element.Cookies }},
      cookies: {
        {{- range $key, $value := $element.Cookies }}
        "{{ $key }}" => "{{ $value }}",
        {{- end }}
      }
    {{- end }}
    {{- if $element.Referer }},
      referer: "{{ $element.Referer }}"
    {{- end }}
    {{- if $element.Header }},
      header: [
        {{- range $element.Header }}
        "{{ . }}",
        {{- end }}
      ]
    {{- end }}
    {{- if $element.UserAgent }},
      user_agent: "{{ $element.UserAgent }}"
    {{- end }}
    {{- if $element.Data }},
      data: {
        {{- range $key, $value := $element.Data }}
        "{{ $key }}" => "{{ $value }}",
        {{- end }}
      }
    {{- end }}
    sha256 "{{ $element.SHA256 }}"
  end
{{- end }}
{{- end }}
