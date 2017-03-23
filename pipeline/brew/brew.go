package brew

import (
	"bytes"
	"errors"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/go-github/github"
	"github.com/goreleaser/goreleaser/clients"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/sha256sum"
)

// ErrNoDarwin64Build when there is no build for darwin_amd64 (goos doesn't
// contain darwin and/or goarch doesn't contain amd64)
var ErrNoDarwin64Build = errors.New("brew tap requires a darwin amd64 build")

const formula = `class {{ .Name }} < Formula
  desc "{{ .Desc }}"
  homepage "{{ .Homepage }}"
  url "https://github.com/{{ .Repo }}/releases/download/{{ .Tag }}/{{ .File }}.{{ .Format }}"
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
	Repo         string
	Tag          string
	Version      string
	BinaryName   string
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
	if ctx.Config.Brew.Repo == "" {
		return nil
	}
	client := clients.GitHub(ctx)
	path := filepath.Join(
		ctx.Config.Brew.Folder, ctx.Config.Build.BinaryName+".rb",
	)

	log.Println("Pushing", path, "to", ctx.Config.Brew.Repo)
	out, err := buildFormula(ctx, client)
	if err != nil {
		return err
	}

	options := &github.RepositoryContentFileOptions{
		Committer: &github.CommitAuthor{
			Name:  github.String("goreleaserbot"),
			Email: github.String("bot@goreleaser"),
		},
		Content: out.Bytes(),
		Message: github.String(
			ctx.Config.Build.BinaryName + " version " + ctx.Git.CurrentTag,
		),
	}

	owner := ctx.BrewRepo.Owner
	repo := ctx.BrewRepo.Name
	file, _, res, err := client.Repositories.GetContents(
		ctx, owner, repo, path, &github.RepositoryContentGetOptions{},
	)
	if err != nil && res.StatusCode == 404 {
		_, _, err = client.Repositories.CreateFile(
			ctx, owner, repo, path, options,
		)
		return err
	}
	options.SHA = file.SHA
	_, _, err = client.Repositories.UpdateFile(ctx, owner, repo, path, options)
	return err
}

func buildFormula(ctx *context.Context, client *github.Client) (bytes.Buffer, error) {
	data, err := dataFor(ctx, client)
	if err != nil {
		return bytes.Buffer{}, err
	}
	return doBuildFormula(data)
}

func doBuildFormula(data templateData) (bytes.Buffer, error) {
	var out bytes.Buffer
	tmpl, err := template.New(data.BinaryName).Parse(formula)
	if err != nil {
		return out, err
	}
	err = tmpl.Execute(&out, data)
	return out, err
}

func dataFor(
	ctx *context.Context, client *github.Client,
) (result templateData, err error) {
	var homepage string
	var description string
	rep, _, err := client.Repositories.Get(
		ctx, ctx.ReleaseRepo.Owner, ctx.ReleaseRepo.Name,
	)
	if err != nil {
		return
	}
	file := ctx.Archives["darwinamd64"]
	if file == "" {
		return result, ErrNoDarwin64Build
	}
	sum, err := sha256sum.For("dist/" + file + "." + ctx.Config.Archive.Format)
	if err != nil {
		return
	}
	if rep.Homepage != nil && *rep.Homepage != "" {
		homepage = *rep.Homepage
	} else {
		homepage = *rep.HTMLURL
	}
	if rep.Description == nil {
		description = "TODO"
	} else {
		description = *rep.Description
	}
	return templateData{
		Name:         formulaNameFor(ctx.Config.Build.BinaryName),
		Desc:         description,
		Homepage:     homepage,
		Repo:         ctx.Config.Release.Repo,
		Tag:          ctx.Git.CurrentTag,
		Version:      ctx.Version,
		BinaryName:   ctx.Config.Build.BinaryName,
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

func formulaNameFor(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	return strings.Replace(strings.Title(name), " ", "", -1)
}
