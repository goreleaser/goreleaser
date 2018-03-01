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
	"github.com/pkg/errors"
)

// Pipe for brew deployment
type Pipe struct{}

func (Pipe) String() string {
	return "getting and validating git state"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	info, err := getInfo(ctx)
	if err != nil {
		return err
	}
	ctx.Git = info
	log.Infof("releasing %s, commit %s", info.CurrentTag, info.Commit)
	if err := setVersion(ctx); err != nil {
		return err
	}
	return validate(ctx)
}

var fakeInfo = context.GitInfo{
	CurrentTag: "v0.0.0",
	Commit:     "none",
}

func getInfo(ctx *context.Context) (context.GitInfo, error) {
	if !git.IsRepo() && ctx.Snapshot {
		log.Warn("accepting to run without a git repo because this is a snapshot")
		return fakeInfo, nil
	}
	if !git.IsRepo() {
		return context.GitInfo{}, ErrNotRepository
	}
	info, err := getGitInfo(ctx)
	if err != nil && ctx.Snapshot {
		log.WithError(err).Warn("ignoring errors because this is a snapshot")
		if info.Commit == "" {
			info = fakeInfo
		}
		return info, nil
	}
	return info, err
}

func getGitInfo(ctx *context.Context) (context.GitInfo, error) {
	commit, err := getCommit()
	if err != nil {
		return context.GitInfo{}, errors.Wrap(err, "couldn't get current commit")
	}
	tag, err := getTag()
	if err != nil {
		return context.GitInfo{
			Commit:     commit,
			CurrentTag: "v0.0.0",
		}, ErrNoTag
	}
	return context.GitInfo{
		CurrentTag: tag,
		Commit:     commit,
	}, nil
}

func setVersion(ctx *context.Context) error {
	if ctx.Snapshot {
		snapshotName, err := getSnapshotName(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to generate snapshot name")
		}
		ctx.Version = snapshotName
		return nil
	}
	// removes usual `v` prefix
	ctx.Version = strings.TrimPrefix(ctx.Git.CurrentTag, "v")
	return nil
}

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

func validate(ctx *context.Context) error {
	if ctx.Snapshot {
		return nil
	}
	out, err := git.Run("status", "--porcelain")
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{out}
	}
	if !regexp.MustCompile("^[0-9.]+").MatchString(ctx.Version) {
		return ErrInvalidVersionFormat{ctx.Version}
	}
	_, err = git.Clean(git.Run("describe", "--exact-match", "--tags", "--match", ctx.Git.CurrentTag))
	if err != nil {
		return ErrWrongRef{ctx.Git.Commit, ctx.Git.CurrentTag}
	}
	return nil
}

func getCommit() (string, error) {
	return git.Clean(git.Run("show", "--format='%H'", "HEAD"))
}

func getTag() (string, error) {
	return git.Clean(git.Run("describe", "--tags", "--abbrev=0"))
}
