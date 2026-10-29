package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/nfpm"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestContextPackageCPUVariants(t *testing.T) {
	for _, arch := range []struct {
		name, baseline, other string
		setVariant            func(*artifact.Artifact, string)
	}{
		{"386", "sse2", "softfloat", func(a *artifact.Artifact, v string) { a.Go386 = v }},
		{"ppc64le", "power8", "power9", func(a *artifact.Artifact, v string) { a.Goppc64 = v }},
		{"riscv64", "rva20u64", "rva22u64", func(a *artifact.Artifact, v string) { a.Goriscv64 = v }},
	} {
		t.Run(arch.name, func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				ProjectName: "mybin",
				Dist:        t.TempDir(),
				NFPMs: []config.NFPM{{
					ID:               "packages",
					IDs:              []string{"cli"},
					Formats:          []string{"deb"},
					Maintainer:       "Example <example@example.com>",
					FileNameTemplate: "{{ .PackageName }}_{{ .Arch }}_{{ .I386 }}{{ .Ppc64 }}{{ .Riscv64 }}",
				}},
			}, testctx.WithVersion("1.0.0"))
			for _, variant := range []string{arch.baseline, arch.other} {
				src := filepath.Join(t.TempDir(), "mybin")
				require.NoError(t, os.WriteFile(src, []byte(variant), 0o755))
				a := &artifact.Artifact{
					Name: "mybin", Path: src, Goos: "linux", Goarch: arch.name,
					Type: artifact.Binary, Extra: artifact.Extras{artifact.ExtraID: "cli"},
				}
				arch.setVariant(a, variant)
				ctx.Artifacts.Add(a)
			}
			require.NoError(t, nfpm.Pipe{}.Default(ctx))
			require.NoError(t, nfpm.Pipe{}.Run(ctx))
			packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
			require.Len(t, packages, 2)

			bare := "linux/" + arch.name
			for _, tc := range []struct {
				platform, variant string
			}{
				{bare, arch.baseline},
				{bare + "/" + arch.baseline, arch.baseline},
				{bare + "/" + arch.other, arch.other},
			} {
				t.Run(tc.platform, func(t *testing.T) {
					d := config.DockerV2{IDs: []string{"packages"}, Platforms: []string{tc.platform}}
					selected := contextArtifacts(ctx, d)
					require.Len(t, selected, 1)
					require.Equal(t, "mybin_"+arch.name+"_"+tc.variant+".deb", selected[0].Name)

					dir, err := makeContext(d, selected, "./testdata/Dockerfile")
					require.NoError(t, err)
					t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
					for _, pkg := range packages {
						target := filepath.Join(dir, tc.platform, pkg.Name)
						if pkg != selected[0] {
							require.NoFileExists(t, target)
							continue
						}
						want, err := os.ReadFile(pkg.Path)
						require.NoError(t, err)
						got, err := os.ReadFile(target)
						require.NoError(t, err)
						require.Equal(t, want, got)
					}
				})
			}
		})
	}
}

func TestContextCPUVariants(t *testing.T) {
	for _, arch := range []struct {
		name, baseline, other string
		setVariant            func(*artifact.Artifact, string)
	}{
		{"386", "sse2", "softfloat", func(a *artifact.Artifact, v string) { a.Go386 = v }},
		{"ppc64le", "power8", "power9", func(a *artifact.Artifact, v string) { a.Goppc64 = v }},
		{"riscv64", "rva20u64", "rva22u64", func(a *artifact.Artifact, v string) { a.Goriscv64 = v }},
	} {
		t.Run(arch.name, func(t *testing.T) {
			bare := "linux/" + arch.name
			for _, tc := range []struct {
				name         string
				platforms    []string
				variants     []string
				wantSelected int
				want         map[string]string
			}{
				{"bare baseline first", []string{bare}, []string{arch.baseline, arch.other}, 1, map[string]string{bare: arch.baseline}},
				{"bare baseline last", []string{bare}, []string{arch.other, arch.baseline}, 1, map[string]string{bare: arch.baseline}},
				{"bare without baseline", []string{bare}, []string{arch.other}, 0, nil},
				{"bare missing metadata", []string{bare}, []string{""}, 1, map[string]string{bare: "unspecified"}},
				{"explicit other", []string{bare + "/" + arch.other}, []string{arch.baseline, arch.other}, 1, map[string]string{bare + "/" + arch.other: arch.other}},
				{"explicit baseline", []string{bare + "/" + arch.baseline}, []string{arch.baseline, arch.other}, 1, map[string]string{bare + "/" + arch.baseline: arch.baseline}},
				{"explicit baseline missing metadata", []string{bare + "/" + arch.baseline}, []string{""}, 1, map[string]string{bare + "/" + arch.baseline: "unspecified"}},
				{"bare and explicit baseline", []string{bare, bare + "/" + arch.baseline}, []string{arch.baseline, arch.other}, 1, map[string]string{bare: arch.baseline, bare + "/" + arch.baseline: arch.baseline}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					ctx := testctx.Wrap(t.Context())
					for _, variant := range tc.variants {
						src := filepath.Join(t.TempDir(), "mybin")
						content := variant
						if content == "" {
							content = "unspecified"
						}
						require.NoError(t, os.WriteFile(src, []byte(content), 0o755))
						a := &artifact.Artifact{
							Name: "mybin", Path: src, Goos: "linux", Goarch: arch.name,
							Type: artifact.Binary, Extra: artifact.Extras{artifact.ExtraID: "cli"},
						}
						arch.setVariant(a, variant)
						ctx.Artifacts.Add(a)
					}

					d := config.DockerV2{ID: "test", IDs: []string{"cli"}, Platforms: tc.platforms}
					selected := contextArtifacts(ctx, d)
					require.Len(t, selected, tc.wantSelected)

					dir, err := makeContext(d, selected, "./testdata/Dockerfile")
					require.NoError(t, err)
					t.Cleanup(func() { _ = os.RemoveAll(dir) })
					for platform, want := range tc.want {
						got, err := os.ReadFile(filepath.Join(dir, platform, "mybin"))
						require.NoError(t, err)
						require.Equal(t, want, string(got))
					}
					if len(tc.want) == 0 {
						require.NoFileExists(t, filepath.Join(dir, bare, "mybin"))
					}
				})
			}
		})
	}
}
