package brew

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"text/template"

	"github.com/google/go-github/github"
	"github.com/goreleaser/releaser/config"
	"golang.org/x/oauth2"
	"log"
)

const formulae = `class {{ .Name }} < Formula
  desc "{{ .Desc }}"
  homepage "{{ .Homepage }}"
  url "https://github.com/{{ .Repo }}/releases/download/{{ .Tag }}/{{ .BinaryName }}_#{%x(uname -s).gsub(/\n/, '')}_#{%x(uname -m).gsub(/\n/, '')}.tar.gz"
  head "https://github.com/{{ .Repo }}.git"
  version "{{ .Tag }}"

  def install
    bin.install "{{ .BinaryName }}"
  end

  {{ if .Caveats }}def caveats
    "{{ .Caveats }}"
  end{{ end }}
end
`

type templateData struct {
	Name, Desc, Homepage, Repo, Tag, BinaryName, Caveats string
}

type Pipe struct{}

func (Pipe) Name() string {
	return "Homebrew"
}

func (Pipe) Work(config config.ProjectConfig) error {
	if config.Brew.Repo == "" {
		return nil
	}
	log.Println("Updating brew formulae...")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	parts := strings.Split(config.Brew.Repo, "/")

	data, err := dataFor(config, client)
	if err != nil {
		return err
	}

	out, err := buildFormulae(data)
	if err != nil {
		return err
	}
	var sha *string
	file, _, _, err := client.Repositories.GetContents(
		parts[0], parts[1], config.BinaryName+".rb", &github.RepositoryContentGetOptions{},
	)
	if err == nil {
		sha = file.SHA
	} else {
		sha = github.String(fmt.Sprintf("%s", sha256.Sum256(out.Bytes())))
	}

	_, _, err = client.Repositories.UpdateFile(
		parts[0],
		parts[1],
		config.BinaryName+".rb",
		&github.RepositoryContentFileOptions{
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

func buildFormulae(data templateData) (bytes.Buffer, error) {
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
	parts := strings.Split(config.Repo, "/")
	rep, _, err := client.Repositories.Get(parts[0], parts[1])
	if err != nil {
		return result, err
	}
	if rep.Homepage == nil {
		homepage = *rep.HTMLURL
	} else {
		homepage = *rep.Homepage
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
	}, err
}

func formulaNameFor(name string) string {
	name = strings.Replace(name, "-", " ", -1)
	name = strings.Replace(name, "_", " ", -1)
	return strings.Replace(strings.Title(name), " ", "", -1)
}
