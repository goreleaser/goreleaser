// Package deprecate provides simple functions to standardize the output
// of deprecation notices on goreleaser
package deprecate

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const baseURL = "https://goreleaser.com/deprecations#"

// Notice warns the user about the deprecation of the given property.
func Notice(ctx *context.Context, property string) {
	NoticeCustom(ctx, property, "`{{ .Property }}` should not be used anymore, check {{ .URL }} for more info")
}

// NoticeCustom warns the user about the deprecation of the given property.
func NoticeCustom(ctx *context.Context, property, tmpl string) {
	ctx.Deprecated = true
	cli.Default.Padding += 3
	defer func() {
		cli.Default.Padding -= 3
	}()
	// replaces . and _ with -
	url := baseURL + strings.NewReplacer(
		".", "",
		"_", "",
	).Replace(property)
	var out bytes.Buffer
	if err := template.Must(template.New("deprecation").Parse("DEPRECATED: "+tmpl)).Execute(&out, templateData{
		URL:      url,
		Property: property,
	}); err != nil {
		panic(err) // this should never happen
	}
	log.Warn(color.New(color.Bold, color.FgHiYellow).Sprintf(out.String()))
}

type templateData struct {
	URL      string
	Property string
}
