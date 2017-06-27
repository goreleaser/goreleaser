// Package brew implements the Pipe, providing formula generation and
// uploading it to a configured repo.
package brew

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/checksum"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/client"
)

// ErrNoDarwin64Build when there is no build for darwin_amd64 (goos doesn't
// contain darwin and/or goarch doesn't contain amd64)
var ErrNoDarwin64Build = errors.New("brew tap requires a darwin amd64 build")

const formula = `class {{ .Name }} < Formula
  desc "{{ .Desc }}"
  homepage "{{ .Homepage }}"
  url "https://github.com/{{ .Repo.Owner }}/{{ .Repo.Name }}/releases/download/{{ .Tag }}/{{ .File }}.{{ .Format }}"
  version "{{ .Version }}"
  sha256 "{{ .SHA256 }}"

  {{- if .Dependencies }}
  {{ range $index, $element := .Dependencies }}
  depends_on "{{ . }}"
  {{- end }}
  {{- end }}

  {{- if .Conflicts }}
  {{ range $index, $element := .Conflicts }}
  conflicts_with "{{ . }}"
  {{- end }}
  {{- end }}

  def install
    {{- range $index, $element := .Install }}
    {{ . -}}
    {{- end }}
  end

  {{- if .Caveats }}

  def caveats
    "{{ .Caveats }}"
  end
  {{- end }}

  {{- if .Plist }}

  def plist; <<-EOS.undent
    {{ .Plist }}
	EOS
  end
  {{- end }}
end
`

type templateData struct {
	Name         string
	Desc         string
	Homepage     string
	Repo         config.Repo // FIXME: will not work for anything but github right now.
	Tag          string
	Version      string
	Caveats      string
	File         string
	Format       string
	SHA256       string
	Plist        string
	Install      []string
	Dependencies []string
	Conflicts    []string
}

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Creating homebrew formula"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	return doRun(ctx, client.NewGitHub(ctx))
}

func doRun(ctx *context.Context, client client.Client) error {
	if !ctx.Publish {
		log.Warn("skipped because --skip-publish is set")
		return nil
	}
	if ctx.Config.Brew.GitHub.Name == "" {
		log.Warn("skipped because brew section is not configured")
		return nil
	}
	if ctx.Config.Release.Draft {
		log.Warn("skipped because release is marked as draft")
		return nil
	}
	path := filepath.Join(ctx.Config.Brew.Folder, ctx.Config.Name+".rb")
	log.WithField("formula", path).
		WithField("repo", ctx.Config.Brew.GitHub.String()).
		Info("pushing")
	content, err := buildFormula(ctx, client)
	if err != nil {
		return err
	}
	return client.CreateFile(ctx, content, path)
}

func buildFormula(ctx *context.Context, client client.Client) (bytes.Buffer, error) {
	data, err := dataFor(ctx, client)
	if err != nil {
		return bytes.Buffer{}, err
	}
	return doBuildFormula(data)
}

func doBuildFormula(data templateData) (bytes.Buffer, error) {
	var out bytes.Buffer
	tmpl, err := template.New(data.Name).Parse(formula)
	if err != nil {
		return out, err
	}
	err = tmpl.Execute(&out, data)
	return out, err
}

func dataFor(ctx *context.Context, client client.Client) (result templateData, err error) {
	file := ctx.Archives["darwinamd64"]
	if file == "" {
		return result, ErrNoDarwin64Build
	}
	sum, err := checksum.SHA256(
		filepath.Join(
			ctx.Config.Dist,
			file+"."+ctx.Config.Archive.Format,
		),
	)
	if err != nil {
		return
	}
	return templateData{
		Name:         formulaNameFor(ctx.Config.Name),
		Desc:         ctx.Config.Brew.Description,
		Homepage:     ctx.Config.Brew.Homepage,
		Repo:         ctx.Config.Release.GitHub,
		Tag:          ctx.Git.CurrentTag,
		Version:      ctx.Version,
		Caveats:      ctx.Config.Brew.Caveats,
		File:         file,
		Format:       ctx.Config.Archive.Format, // TODO this can be broken by format_overrides
		SHA256:       sum,
		Dependencies: ctx.Config.Brew.Dependencies,
		Conflicts:    ctx.Config.Brew.Conflicts,
		Plist:        ctx.Config.Brew.Plist,
		Install:      strings.Split(ctx.Config.Brew.Install, "\n"),
	}, err
}

func formulaNameFor(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	return strings.Replace(strings.Title(name), " ", "", -1)
}
