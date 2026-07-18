package main

import (
	"os"
	"text/template"
	"strings"
)

const ebuildTemplate = `SRC_URI="
{{- range .Archs }}
  {{ .Keyword }}? ( {{ .URI }} -> {{ .File }} )
{{- end }}
"

src_install() {
{{- with .ExtraInstall }}
{{ . }}
{{- end }}
{{- with .Bindir }}
  exeinto {{ . }}
{{- end }}
}
`

func main() {
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"indent": func(spaces int, v string) string {
			pad := strings.Repeat(" ", spaces)
			return pad + strings.Replace(v, "\n", "\n"+pad, -1)
		},
	}).Parse(ebuildTemplate))

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

    // mimic what user passes, maybe multiline
    extraInstall := `dobin "foo"
doins "bar"`

	tmpl.Execute(os.Stdout, Data{
		ExtraInstall: extraInstall,
		Archs: []ArchData{
			{Keyword: "amd64", URI: "http", File: "file"},
		},
	})
}
