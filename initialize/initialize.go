package initialize

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/goreleaser/releaser/config/git"
)

var initTemplate = `repo: {{ .Repo }}
binary_name: {{ .BinaryName }}
`

type initData struct {
	Repo, BinaryName string
}

func Init() error {
	file, err := os.Create("goreleaser.yml")
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	data, err := generateFileContent()
	if err != nil {
		return err
	}
	fmt.Fprintf(file, data)
	return nil
}

func generateFileContent() (string, error) {
	repo, err := git.RemoteRepoName()
	if err != nil {
		return "", err
	}
	var data = initData{
		Repo:       repo,
		BinaryName: strings.Split(repo, "/")[1],
	}
	var out bytes.Buffer
	t, err := template.New(data.BinaryName).Parse(initTemplate)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}
