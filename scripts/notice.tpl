# Third-Party Dependencies Licenses

{{ range . }}
## `{{ .Name }}`
{{- if eq .LicenseName "Unknown" }}
Unknown/unspecified license.
{{- else }}

```
{{ .LicenseText }}
```
{{ end }}
{{ end }}
