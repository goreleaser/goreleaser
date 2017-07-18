package git

import (
	"bytes"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/context"
)

type snapshotNameData struct {
	Commit    string
	Tag       string
	Timestamp int64
}

func getSnapshotName(ctx *context.Context) (string, error) {
	tmpl, err := template.New("snapshot").Parse(ctx.Config.Snapshot.NameTemplate)
	var out bytes.Buffer
	if err != nil {
		return "", err
	}
	var data = snapshotNameData{
		Commit:    ctx.Git.Commit,
		Tag:       ctx.Git.CurrentTag,
		Timestamp: time.Now().Unix(),
	}
	err = tmpl.Execute(&out, data)
	return out.String(), err
}
