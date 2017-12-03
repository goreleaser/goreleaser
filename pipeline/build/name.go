package build

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/buildtarget"
)

func nameFor(ctx *context.Context, target buildtarget.Target, name string) (string, error) {
	var out bytes.Buffer
	t, err := template.New(target.String()).Parse(ctx.Config.Archive.NameTemplate)
	if err != nil {
		return "", err
	}
	data := struct {
		Os, Arch, Arm, Version, Tag, Binary, ProjectName string
	}{
		Os:          replace(ctx.Config.Archive.Replacements, target.OS),
		Arch:        replace(ctx.Config.Archive.Replacements, target.Arch),
		Arm:         replace(ctx.Config.Archive.Replacements, target.Arm),
		Version:     ctx.Version,
		Tag:         ctx.Git.CurrentTag,
		Binary:      name, // TODO: deprecated: remove this sometime
		ProjectName: name,
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
