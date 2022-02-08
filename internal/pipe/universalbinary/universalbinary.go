// Package universalbinary can join multiple darwin binaries into a single universal binary.
package universalbinary

import (
	"debug/macho"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/caarlos0/go-shellwords"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/shell"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/build"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for macos universal binaries.
type Pipe struct{}

func (Pipe) String() string                 { return "universal binaries" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.UniversalBinaries) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("universal_binaries")
	for i := range ctx.Config.UniversalBinaries {
		unibin := &ctx.Config.UniversalBinaries[i]
		if unibin.ID == "" {
			unibin.ID = ctx.Config.ProjectName
		}
		if len(unibin.IDs) == 0 {
			unibin.IDs = []string{unibin.ID}
		}
		if unibin.NameTemplate == "" {
			unibin.NameTemplate = "{{ .ProjectName }}"
		}
		ids.Inc(unibin.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, unibin := range ctx.Config.UniversalBinaries {
		unibin := unibin
		g.Go(func() error {
			opts := build.Options{
				Target: "darwin_all",
				Goos:   "darwin",
				Goarch: "all",
			}
			if err := runHook(ctx, &opts, unibin.Hooks.Pre); err != nil {
				return fmt.Errorf("pre hook failed: %w", err)
			}
			if err := makeUniversalBinary(ctx, &opts, unibin); err != nil {
				return err
			}
			if err := runHook(ctx, &opts, unibin.Hooks.Post); err != nil {
				return fmt.Errorf("post hook failed: %w", err)
			}
			if !unibin.Replace {
				return nil
			}
			return ctx.Artifacts.Remove(filterFor(unibin))
		})
	}
	return g.Wait()
}

func runHook(ctx *context.Context, opts *build.Options, hooks config.Hooks) error {
	if len(hooks) == 0 {
		return nil
	}

	for _, hook := range hooks {
		var envs []string
		envs = append(envs, ctx.Env.Strings()...)

		tpl := tmpl.New(ctx).WithBuildOptions(*opts)
		for _, rawEnv := range hook.Env {
			env, err := tpl.Apply(rawEnv)
			if err != nil {
				return err
			}

			envs = append(envs, env)
		}

		tpl = tpl.WithEnvS(envs)
		dir, err := tpl.Apply(hook.Dir)
		if err != nil {
			return err
		}

		sh, err := tpl.Apply(hook.Cmd)
		if err != nil {
			return err
		}

		log.WithField("hook", sh).Info("running hook")
		cmd, err := shellwords.Parse(sh)
		if err != nil {
			return err
		}

		if err := shell.Run(ctx, dir, cmd, envs, hook.Output); err != nil {
			return err
		}
	}
	return nil
}

type input struct {
	data   []byte
	cpu    uint32
	subcpu uint32
	offset int64
}

const (
	// Alignment wanted for each sub-file.
	// amd64 needs 12 bits, arm64 needs 14. We choose the max of all requirements here.
	alignBits = 14
	align     = 1 << alignBits
)

// heavily based on https://github.com/randall77/makefat
func makeUniversalBinary(ctx *context.Context, opts *build.Options, unibin config.UniversalBinary) error {
	name, err := tmpl.New(ctx).Apply(unibin.NameTemplate)
	if err != nil {
		return err
	}
	opts.Name = name

	path := filepath.Join(ctx.Config.Dist, name+"_darwin_all", name)
	opts.Path = path
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	binaries := ctx.Artifacts.Filter(filterFor(unibin)).List()
	if len(binaries) == 0 {
		return pipe.Skip(fmt.Sprintf("no darwin binaries found with id %q", unibin.ID))
	}

	log.WithField("binary", path).Infof("creating from %d binaries", len(binaries))

	var inputs []input
	offset := int64(align)
	for _, f := range binaries {
		data, err := os.ReadFile(f.Path)
		if err != nil {
			return fmt.Errorf("failed to read binary: %w", err)
		}
		inputs = append(inputs, input{
			data:   data,
			cpu:    binary.LittleEndian.Uint32(data[4:8]),
			subcpu: binary.LittleEndian.Uint32(data[8:12]),
			offset: offset,
		})
		offset += int64(len(data))
		offset = (offset + align - 1) / align * align
	}

	// Make output file.
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if err := out.Chmod(0o755); err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Build a fat_header.
	hdr := []uint32{macho.MagicFat, uint32(len(inputs))}

	// Build a fat_arch for each input file.
	for _, i := range inputs {
		hdr = append(hdr, i.cpu)
		hdr = append(hdr, i.subcpu)
		hdr = append(hdr, uint32(i.offset))
		hdr = append(hdr, uint32(len(i.data)))
		hdr = append(hdr, alignBits)
	}

	// Write header.
	// Note that the fat binary header is big-endian, regardless of the
	// endianness of the contained files.
	if err := binary.Write(out, binary.BigEndian, hdr); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	offset = int64(4 * len(hdr))

	// Write each contained file.
	for _, i := range inputs {
		if offset < i.offset {
			if _, err := out.Write(make([]byte, i.offset-offset)); err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
			offset = i.offset
		}
		if _, err := out.Write(i.data); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		offset += int64(len(i.data))
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	extra := map[string]interface{}{}
	for k, v := range binaries[0].Extra {
		extra[k] = v
	}
	extra[artifact.ExtraReplaces] = unibin.Replace

	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UniversalBinary,
		Name:   name,
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Extra:  extra,
	})

	return nil
}

func filterFor(unibin config.UniversalBinary) artifact.Filter {
	return artifact.And(
		artifact.ByType(artifact.Binary),
		artifact.ByGoos("darwin"),
		artifact.ByIDs(unibin.IDs...),
	)
}
