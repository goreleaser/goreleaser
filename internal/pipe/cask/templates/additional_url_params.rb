{{- define "additional_url_params" }}
{{- if .URLAdditional.Verified }},
      verified: "{{ .URLAdditional.Verified }}"
{{- end }}
{{- if .URLAdditional.Using }},
      using: {{ .URLAdditional.Using }}
{{- end }}
{{- if .URLAdditional.Cookies }},
      cookies: {
        {{- range $key, $value := .URLAdditional.Cookies }}
        "{{ $key }}" => "{{ $value }}",
        {{- end }}
      }
{{- end }}
{{- if .URLAdditional.Referer }},
      referer: "{{ .URLAdditional.Referer }}"
{{- end }}
{{- if .URLAdditional.Headers }},
      header: [
        {{- range .URLAdditional.Headers }}
        "{{ . }}",
        {{- end }}
      ]
{{- end }}
{{- if .URLAdditional.UserAgent }},
      user_agent: "{{ .URLAdditional.UserAgent }}"
{{- end }}
{{- if .URLAdditional.Data }},
      data: {
        {{- range $key, $value := .URLAdditional.Data }}
        "{{ $key }}" => "{{ $value }}",
        {{- end }}
      }
{{- end }}
{{- end }}
