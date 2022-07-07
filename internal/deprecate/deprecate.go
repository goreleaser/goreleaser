// Package deprecate provides simple functions to standardize the output
// of deprecation notices on goreleaser
package deprecate

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/caarlos0/log"
	"github.com/charmbracelet/lipgloss"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const baseURL = "https://goreleaser.com/deprecations#"

var warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)

// Notice warns the user about the deprecation of the given property.
func Notice(ctx *context.Context, property string) {
	NoticeCustom(ctx, property, "`{{ .Property }}` should not be used anymore, check {{ .URL }} for more info")
}

// NoticeCustom warns the user about the deprecation of the given property.
func NoticeCustom(ctx *context.Context, property, tmpl string) {
	log.IncreasePadding()
	defer log.DecreasePadding()

	// replaces . and _ with -
	url := baseURL + strings.NewReplacer(
		".", "",
		"_", "",
		":", "",
		" ", "-",
	).Replace(property)
	var out bytes.Buffer
	if err := template.Must(template.New("deprecation").Parse("DEPRECATED: "+tmpl)).Execute(&out, templateData{
		URL:      url,
		Property: property,
	}); err != nil {
		panic(err) // this should never happen
	}

	ctx.Deprecated = true
	log.Warn(warnStyle.Render(out.String()))
}

type templateData struct {
	URL      string
	Property string
}
