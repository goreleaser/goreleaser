package main

import (
	"os"
	"text/template"
)

const ebuildTemplate = `SRC_URI="
{{- range .Archs }}
  {{ .Keyword }}? ( {{ .URI }} -> {{ .File }} )
{{- end }}
"

src_install() {
{{- if .ExtraInstall }}
  {{ .ExtraInstall }}
{{- end }}
{{- if .Bindir }}
  exeinto {{ .Bindir }}
{{- end }}
}
`

func main() {
	tmpl := template.Must(template.New("").Parse(ebuildTemplate))

	type ArchData struct {
		Keyword string
		URI string
		File string
	}
	type Data struct {
		ExtraInstall string
		Bindir       string
		Archs []ArchData
	}

	tmpl.Execute(os.Stdout, Data{
		Bindir: "/usr/bin",
		Archs: []ArchData{
			{Keyword: "amd64", URI: "http", File: "file"},
		},
	})
}
