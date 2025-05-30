{{- define "additional_url_params" }}
{{- if .Verified }},
      verified: "{{ .Verified }}"
{{- end }}
{{- if .Using }},
      using: {{ .Using }}
{{- end }}
{{- if .Cookies }},
      cookies: {
        {{- range $key, $value := .Cookies }}
        "{{ $key }}" => "{{ $value }}",
        {{- end }}
      }
{{- end }}
{{- if .Referer }},
      referer: "{{ .Referer }}"
{{- end }}
{{- if .Header }},
      header: [
        {{- range .Header }}
        "{{ . }}",
        {{- end }}
      ]
{{- end }}
{{- if .UserAgent }},
      user_agent: "{{ .UserAgent }}"
{{- end }}
{{- if .Data }},
      data: {
        {{- range $key, $value := .Data }}
        "{{ $key }}" => "{{ $value }}",
        {{- end }}
      }
{{- end }}
{{- end }}
