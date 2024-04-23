// Package deprecate provides simple functions to standardize the output
// of deprecation notices on goreleaser
package deprecate

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const baseURL = "https://goreleaser.com/deprecations#"

// Notice warns the user about the deprecation of the given property.
func Notice(ctx *context.Context, property string) {
	NoticeCustom(ctx, property, "{{ .Property }} should not be used anymore, check {{ .URL }} for more info")
}

var urlPropertyReplacer = strings.NewReplacer(
	".", "",
	"_", "",
	":", "",
	" ", "-",
)

// NoticeCustom warns the user about the deprecation of the given property.
func NoticeCustom(ctx *context.Context, property, tmpl string) {
	// replaces . and _ with -
	url := baseURL + urlPropertyReplacer.Replace(property)
	var out bytes.Buffer
	if err := template.
		Must(template.New("deprecation").Parse("DEPRECATED: "+tmpl)).
		Execute(&out, templateData{
			URL:      logext.URL(url),
			Property: logext.Keyword(property),
		}); err != nil {
		panic(err) // this should never happen
	}

	ctx.Deprecated = true
	log.Warn(logext.Warning(out.String()))
}

type templateData struct {
	URL      string
	Property string
}
