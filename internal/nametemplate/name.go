package nametemplate

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

func Apply(ctx *context.Context, a artifact.Artifact) (string, error) {
	var out bytes.Buffer
	t, err := template.New("archive_name").Parse(ctx.Config.Archive.NameTemplate)
	if err != nil {
		return "", err
	}
	data := struct {
		Os, Arch, Arm, Version, Tag, Binary, ProjectName string
		Env                                              map[string]string
	}{
		Os:          replace(ctx.Config.Archive.Replacements, a.Goos),
		Arch:        replace(ctx.Config.Archive.Replacements, a.Goarch),
		Arm:         replace(ctx.Config.Archive.Replacements, a.Goarm),
		Version:     ctx.Version,
		Tag:         ctx.Git.CurrentTag,
		ProjectName: ctx.Config.ProjectName,
		Env:         ctx.Env,
	}
	err = t.Execute(&out, data)
	return out.String(), err
}

func replace(replacements map[string]string, original string) string {
	result := replacements[original]
	if result == "" {
		return original
	}
	return result
}
