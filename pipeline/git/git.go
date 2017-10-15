// Package git implements the Pipe interface getting and validating the
// current git repository state
package git

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/pkg/errors"
)

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Getting and validating git state"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	tag, commit, err := getInfo()
	if err != nil {
		return
	}
	if tag == "" && !ctx.Snapshot {
		return ErrNoTag
	}
	ctx.Git = context.GitInfo{
		CurrentTag: tag,
		Commit:     commit,
	}
	log.Infof("releasing %s, commit %s", tag, commit)
	if err = setVersion(ctx, tag, commit); err != nil {
		return
	}
	if !ctx.Validate {
		return pipeline.Skip("--skip-validate is set")
	}
	return validate(ctx, commit, tag)
}

func setVersion(ctx *context.Context, tag, commit string) (err error) {
	if ctx.Snapshot {
		snapshotName, err := getSnapshotName(ctx, tag, commit)
		if err != nil {
			return errors.Wrap(err, "failed to generate snapshot name")
		}
		ctx.Version = snapshotName
		return nil
	}
	// removes usual `v` prefix
	ctx.Version = strings.TrimPrefix(tag, "v")
	return
}

type snapshotNameData struct {
	Commit    string
	Tag       string
	Timestamp int64
}

func getSnapshotName(ctx *context.Context, tag, commit string) (string, error) {
	tmpl, err := template.New("snapshot").Parse(ctx.Config.Snapshot.NameTemplate)
	var out bytes.Buffer
	if err != nil {
		return "", err
	}
	var data = snapshotNameData{
		Commit:    commit,
		Tag:       tag,
		Timestamp: time.Now().Unix(),
	}
	err = tmpl.Execute(&out, data)
	return out.String(), err
}

func validate(ctx *context.Context, commit, tag string) error {
	out, err := git.Run("status", "--porcelain")
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{out}
	}
	if ctx.Snapshot {
		return nil
	}
	if !regexp.MustCompile("^[0-9.]+").MatchString(ctx.Version) {
		return ErrInvalidVersionFormat{ctx.Version}
	}
	_, err = git.Clean(git.Run("describe", "--exact-match", "--tags", "--match", tag))
	if err != nil {
		return ErrWrongRef{commit, tag}
	}
	return nil
}

func getInfo() (tag, commit string, err error) {
	tag, err = git.Clean(git.Run("describe", "--tags", "--abbrev=0"))
	if err != nil {
		log.WithError(err).Info("failed to retrieve current tag")
	}
	commit, err = git.Clean(git.Run("show", "--format='%H'", "HEAD"))
	return
}
