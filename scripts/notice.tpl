# Third-Party Dependencies Licenses

The following is a listing of the GoReleaser open source components detailed in
this document. This list is provided for your convenience; please read further
if you wish to review the copyright notice(s) and the full text of the license
associated with each component.

{{ range . }}
## {{ .Name }}
{{- if eq .LicenseName "Unknown" }}

Unknown/unspecified license.
{{- else }}

```
{{ .LicenseText }}
```
{{ end }}
{{ end }}
