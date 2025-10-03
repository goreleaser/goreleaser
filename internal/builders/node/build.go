package node

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/common"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Default builder instance.
//
//nolint:gochecknoglobals
var Default = &Builder{}

var (
	_ api.Builder           = &Builder{}
	_ api.DependingBuilder  = &Builder{}
	_ api.ConcurrentBuilder = &Builder{}
)

//nolint:gochecknoinits
func init() {
	api.Register("node", Default)
}

// Builder is node builder.
type Builder struct{}

// AllowConcurrentBuilds implements build.ConcurrentBuilder.
func (b *Builder) AllowConcurrentBuilds() bool { return false }

// Dependencies implements build.DependingBuilder.
func (b *Builder) Dependencies() []string {
	return []string{"node", "nodejs-sea-creator"}
}

var once sync.Once

// WithDefaults implements build.Builder.
func (b *Builder) WithDefaults(build config.Build) (config.Build, error) {
	once.Do(func() {
		log.Warn("you are using the experimental Node.js builder")
	})

	if len(build.Targets) == 0 {
		build.Targets = defaultTargets()
	}

	if build.Tool == "" {
		build.Tool = "nodejs-sea-creator"
	}

	// the tool doesn't have any subcommands, it just takes path to sea-config.json
	// https://github.com/vedantmgoyal9/nodejs-sea-creator
	if build.Command == "" {
		build.Command = "sea-config.json"
	}

	if build.Dir == "" {
		build.Dir = "."
	}

	if len(build.Flags) != 0 {
		return build, errors.New("flags are not supported for node builder")
	}

	if build.Main != "" {
		return build, errors.New("main is not supported for node builder")
	}

	if err := common.ValidateNonGoConfig(build); err != nil {
		return build, err
	}

	for _, t := range build.Targets {
		if !isValid(t) {
			return build, fmt.Errorf("invalid target: %s", t)
		}
	}

	return build, nil
}

// Build implements build.Builder.
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	t := options.Target.(Target)
	a := &artifact.Artifact{
		Type:   artifact.Binary,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   t.Os,
		Goarch: convertToGoarch(t.Arch),
		Target: t.Target,
		Extra: map[string]interface{}{
			artifact.ExtraBinary:  strings.TrimSuffix(filepath.Base(options.Path), options.Ext),
			artifact.ExtraExt:     options.Ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "node",
		},
	}

	env := []string{}
	env = append(env, ctx.Env.Strings()...)

	tpl := tmpl.New(ctx).
		WithBuildOptions(options).
		WithEnvS(env).
		WithArtifact(a)

	nodejs_sea_creator, err := tpl.Apply(build.Tool)
	if err != nil {
		return err
	}

	command := []string{nodejs_sea_creator, build.Command}
	command = append(command, build.Flags...)
	command = append(
		command,
		"--target", t.Target,
	)

	tenv, err := common.TemplateEnv(build, tpl)
	if err != nil {
		return err
	}
	env = append(env, tenv...)

	flags, err := tpl.Slice(build.Flags, tmpl.NonEmpty())
	if err != nil {
		return err
	}
	command = append(command, flags...)

	if err := common.Exec(ctx, command, env, build.Dir); err != nil {
		return err
	}

	realPath := filepath.Join(build.Dir, options.Name + "-" + t.Target)
	if err := gio.Copy(realPath, options.Path); err != nil {
		return err
	}

	if err := common.ChTimes(build, tpl, a); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}
