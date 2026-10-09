package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

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
