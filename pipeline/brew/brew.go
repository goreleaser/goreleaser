package brew

import (
	"bytes"
	"errors"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/goreleaser/goreleaser/clients"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/sha256sum"
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
	Binary       string
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
	// TODO: remove this block in next release cycle
	if ctx.Config.Brew.Repo != "" {
		log.Println("The `brew.repo` syntax is deprecated and will soon be removed. Please check the README for more info.")
		var ss = strings.Split(ctx.Config.Brew.Repo, "/")
		ctx.Config.Brew.GitHub = config.Repo{
			Owner: ss[0],
			Name:  ss[1],
		}
	}
	if ctx.Config.Brew.GitHub.Name == "" {
		return nil
	}
	client := clients.NewGitHubClient(ctx)
	path := filepath.Join(ctx.Config.Brew.Folder, ctx.Config.Build.Binary+".rb")

	log.Println("Pushing", path, "to", ctx.Config.Brew.GitHub.String())
	content, err := buildFormula(ctx, client)
	if err != nil {
		return err
	}

	return client.CreateFile(ctx, content, path)
}

func buildFormula(ctx *context.Context, client clients.Client) (bytes.Buffer, error) {
	data, err := dataFor(ctx, client)
	if err != nil {
		return bytes.Buffer{}, err
	}
	return doBuildFormula(data)
}

func doBuildFormula(data templateData) (bytes.Buffer, error) {
	var out bytes.Buffer
	tmpl, err := template.New(data.Binary).Parse(formula)
	if err != nil {
		return out, err
	}
	err = tmpl.Execute(&out, data)
	return out, err
}

func dataFor(ctx *context.Context, client clients.Client) (result templateData, err error) {
	file := ctx.Archives["darwinamd64"]
	if file == "" {
		return result, ErrNoDarwin64Build
	}
	sum, err := sha256sum.For(
		filepath.Join(
			ctx.Config.Dist,
			file+"."+ctx.Config.Archive.Format,
		),
	)
	if err != nil {
		return
	}
	homepage, description, err := getInfo(ctx, client)
	if err != nil {
		return
	}
	return templateData{
		Name:         formulaNameFor(ctx.Config.Build.Binary),
		Desc:         description,
		Homepage:     homepage,
		Repo:         ctx.Config.Release.GitHub,
		Tag:          ctx.Git.CurrentTag,
		Version:      ctx.Version,
		Binary:       ctx.Config.Build.Binary,
		Caveats:      ctx.Config.Brew.Caveats,
		File:         file,
		Format:       ctx.Config.Archive.Format,
		SHA256:       sum,
		Dependencies: ctx.Config.Brew.Dependencies,
		Conflicts:    ctx.Config.Brew.Conflicts,
		Plist:        ctx.Config.Brew.Plist,
		Install:      strings.Split(ctx.Config.Brew.Install, "\n"),
	}, err
}

func getInfo(
	ctx *context.Context,
	client clients.Client,
) (homepage string, description string, err error) {
	info, err := client.GetInfo(ctx)
	if err != nil {
		return
	}
	if info.Homepage != "" {
		homepage = info.Homepage
	} else {
		homepage = info.URL
	}
	if info.Description == "" {
		description = "TODO"
	} else {
		description = info.Description
	}
	return
}

func formulaNameFor(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	return strings.Replace(strings.Title(name), " ", "", -1)
}
