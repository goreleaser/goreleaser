package fatbinary

import (
	"debug/macho"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for macos fat binaries.
type Pipe struct{}

func (Pipe) String() string                 { return "macos fat binaries" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.MacOSFatBinaries) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("macos_fatbins")
	for i := range ctx.Config.MacOSFatBinaries {
		fatbin := &ctx.Config.MacOSFatBinaries[i]
		if fatbin.ID == "" {
			fatbin.ID = ctx.Config.ProjectName
		}
		if fatbin.NameTemplate == "" {
			fatbin.NameTemplate = "{{ .ProjectName }}"
		}
		ids.Inc(fatbin.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, fatbin := range ctx.Config.MacOSFatBinaries {
		fatbin := fatbin
		g.Go(func() error {
			if err := makeFatBinary(ctx, fatbin); err != nil {
				return err
			}
			if !fatbin.Replace {
				return nil
			}
			return ctx.Artifacts.Remove(filterFor(fatbin))
		})
	}
	return g.Wait()
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
func makeFatBinary(ctx *context.Context, fatbin config.FatBinary) error {
	name, err := tmpl.New(ctx).Apply(fatbin.NameTemplate)
	if err != nil {
		return err
	}

	path := filepath.Join(ctx.Config.Dist, name+"_darwinall", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	binaries := ctx.Artifacts.Filter(filterFor(fatbin)).List()
	if len(binaries) == 0 {
		return pipe.Skip(fmt.Sprintf("no darwin binaries found with id %q", fatbin.ID))
	}

	log.WithField("fatbinary", path).Infof("creating from %d binaries", len(binaries))

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

	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.FatBinary,
		Name:   name,
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Extra:  binaries[0].Extra,
	})

	return nil
}

func filterFor(fatbin config.FatBinary) artifact.Filter {
	return artifact.And(
		artifact.ByType(artifact.Binary),
		artifact.ByGoos("darwin"),
		artifact.ByIDs(fatbin.ID),
	)
}
