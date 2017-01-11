package brew

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/google/go-github/github"
	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/sha256sum"
	"github.com/goreleaser/releaser/split"
	"golang.org/x/oauth2"
)

const formulae = `class {{ .Name }} < Formula
  desc "{{ .Desc }}"
  homepage "{{ .Homepage }}"
  url "https://github.com/{{ .Repo }}/releases/download/{{ .Tag }}/{{ .File }}.{{ .Format }}"
  version "{{ .Tag }}"
  sha256 "{{ .SHA256 }}"

  def install
    bin.install "{{ .BinaryName }}"
  end

  {{- if .Caveats }}

  def caveats
    "{{ .Caveats }}"
  end
  {{- end }}
end
`

type templateData struct {
	Name, Desc, Homepage, Repo, Tag, BinaryName, Caveats, File, Format, SHA256 string
}

// Pipe for brew deployment
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Homebrew"
}

// Run the pipe
func (Pipe) Run(config config.ProjectConfig) error {
	if config.Brew.Repo == "" {
		return nil
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	owner, repo := split.OnSlash(config.Brew.Repo)
	name := config.BinaryName + ".rb"

	log.Println("Updating", name, "on", config.Brew.Repo, "...")
	out, err := buildFormulae(config, client)
	if err != nil {
		return err
	}
	sha, err := formulaeSHA(client, owner, repo, name, out)
	if err != nil {
		return err
	}
	_, _, err = client.Repositories.UpdateFile(
		owner, repo, name, &github.RepositoryContentFileOptions{
			Committer: &github.CommitAuthor{
				Name:  github.String("goreleaserbot"),
				Email: github.String("bot@goreleaser"),
			},
			Content: out.Bytes(),
			Message: github.String(config.BinaryName + " version " + config.Git.CurrentTag),
			SHA:     sha,
		},
	)
	return err
}

func formulaeSHA(client *github.Client, owner, repo, name string, out bytes.Buffer) (*string, error) {
	file, _, _, err := client.Repositories.GetContents(
		owner, repo, name, &github.RepositoryContentGetOptions{},
	)
	if err == nil {
		return file.SHA, err
	}
	return github.String(fmt.Sprintf("%s", sha256.Sum256(out.Bytes()))), nil
}

func buildFormulae(config config.ProjectConfig, client *github.Client) (bytes.Buffer, error) {
	data, err := dataFor(config, client)
	if err != nil {
		return bytes.Buffer{}, err
	}
	return doBuildFormulae(data)
}

func doBuildFormulae(data templateData) (bytes.Buffer, error) {
	var out bytes.Buffer
	tmpl, err := template.New(data.BinaryName).Parse(formulae)
	if err != nil {
		return out, err
	}
	err = tmpl.Execute(&out, data)
	return out, err
}

func dataFor(config config.ProjectConfig, client *github.Client) (result templateData, err error) {
	var homepage string
	var description string
	owner, repo := split.OnSlash(config.Repo)
	rep, _, err := client.Repositories.Get(owner, repo)
	if err != nil {
		return
	}
	file, err := config.ArchiveName("darwin", "amd64")
	if err != nil {
		return
	}
	sum, err := sha256sum.For("dist/" + file + "." + config.Archive.Format)
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
		Name:       formulaNameFor(config.BinaryName),
		Desc:       description,
		Homepage:   homepage,
		Repo:       config.Repo,
		Tag:        config.Git.CurrentTag,
		BinaryName: config.BinaryName,
		Caveats:    config.Brew.Caveats,
		File:       file,
		Format:     config.Archive.Format,
		SHA256:     sum,
	}, err
}

func formulaNameFor(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	return strings.Replace(strings.Title(name), " ", "", -1)
}
