package nametemplate

import (
	"bytes"
	"regexp"
	"text/template"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

var deprecatedBinary = regexp.MustCompile("\\{\\{ ?\\.Binary ?\\}\\}")

// Apply applies the name template to the given artifact and name
// TODO: this should be refactored alongside with other name template related todos
func Apply(ctx *context.Context, a artifact.Artifact, name string) (string, error) {
	if deprecatedBinary.MatchString(ctx.Config.Archive.NameTemplate) {
		log.WithField("field", "{{.Binary}}").Warn("you are using a deprecated field on your template, please check the documentation")
	}
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
		ProjectName: name,
		Binary:      name, // TODO: deprecated, remove soon
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
