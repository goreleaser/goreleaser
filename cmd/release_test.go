package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestRelease(t *testing.T) {
	setup(t)
	cmd := newReleaseCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestReleaseGoModProxyReplaceNotCaughtBySnapshotAlone(t *testing.T) {
	t.Run("snapshot_accepts_replace", func(t *testing.T) {
		setup(t)
		createGoModProxyGoreleaser(t)
		modEditReplace(t, "foo", "../bar")
		cmd := newReleaseCmd()
		cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated", "--clean"})
		require.NoError(t, cmd.cmd.Execute())
	})

	t.Run("non_snapshot_rejects_replace", func(t *testing.T) {
		setup(t)
		createGoModProxyGoreleaser(t)
		modEditReplace(t, "foo", "../bar")
		cmd := newReleaseCmd()
		cmd.cmd.SetArgs([]string{
			"--skip=publish,announce,validate",
			"--timeout=1m",
			"--parallelism=2",
			"--deprecated",
			"--clean",
		})
		err := cmd.cmd.Execute()
		require.Error(t, err)
		require.ErrorContains(t, err, "replace")
	})
}

func createGoModProxyGoreleaser(tb testing.TB) {
	tb.Helper()
	createFile(tb, "goreleaser.yml", `gomod:
  proxy: true
builds:
- id: foo
  binary: 'fake{{if .IsSnapshot}}_snapshot{{end}}'
  main: .
  dir: .
  goos:
    - linux
  goarch:
    - amd64
release:
  github:
    owner: goreleaser
    name: fake
`)
}

func modEditReplace(tb testing.TB, modulePath, replacement string) {
	tb.Helper()
	cmd := exec.CommandContext(tb.Context(), "go", "mod", "edit", "-replace", modulePath+"="+replacement)
	require.NoError(tb, cmd.Run())
}

func TestReleaseAutoSnapshot(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		setup(t)
		cmd := newReleaseCmd()
		cmd.cmd.SetArgs([]string{"--auto-snapshot", "--skip=publish"})
		require.NoError(t, cmd.cmd.Execute())
		require.FileExists(t, "dist/fake_0.0.2_checksums.txt", "should have created checksums when run with --snapshot")
	})

	t.Run("dirty", func(t *testing.T) {
		setup(t)
		createFile(t, "foo", "force dirty tree")
		cmd := newReleaseCmd()
		cmd.cmd.SetArgs([]string{"--auto-snapshot", "--skip=publish"})
		require.NoError(t, cmd.cmd.Execute())
		matches, err := filepath.Glob("./dist/fake_0.0.2-SNAPSHOT-*_checksums.txt")
		require.NoError(t, err)
		require.Len(t, matches, 1, "should have implied --snapshot")
	})
}

func TestReleaseInvalidConfig(t *testing.T) {
	setup(t)
	createFile(t, "goreleaser.yml", "foo: bar\nversion: 2")
	cmd := newReleaseCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.EqualError(t, cmd.cmd.Execute(), "yaml: unmarshal errors:\n  line 1: field foo not found in type config.Project")
}

func TestReleaseBrokenProject(t *testing.T) {
	setup(t)
	createFile(t, "main.go", "not a valid go file")
	cmd := newReleaseCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2"})
	require.ErrorContains(t, cmd.cmd.Execute(), "failed to parse dir: .: main.go:1:1: expected 'package', found not")
}

func TestReleaseFlags(t *testing.T) {
	setup := func(tb testing.TB, opts releaseOpts) *context.Context {
		tb.Helper()
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, setupReleaseContext(ctx, opts))
		return ctx
	}

	t.Run("draft", func(t *testing.T) {
		t.Run("not set", func(t *testing.T) {
			ctx := setup(t, releaseOpts{})
			require.False(t, ctx.Config.Release.Draft)
		})

		t.Run("set via flag", func(t *testing.T) {
			ctx := setup(t, releaseOpts{
				draft: true,
			})
			require.True(t, ctx.Config.Release.Draft)
		})

		t.Run("set in config", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Release: config.Release{
					Draft: true,
				},
			})

			require.NoError(t, setupReleaseContext(ctx, releaseOpts{}))
			require.True(t, ctx.Config.Release.Draft)
		})
	})

	t.Run("action", func(t *testing.T) {
		ctx := setup(t, releaseOpts{})
		require.Equal(t, context.ActionRelease, ctx.Action)
	})

	t.Run("snapshot", func(t *testing.T) {
		ctx := setup(t, releaseOpts{
			snapshot: true,
		})
		require.True(t, ctx.Snapshot)
		requireAll(t, ctx, skips.Publish, skips.Validate, skips.Announce)
	})

	t.Run("skips", func(t *testing.T) {
		ctx := setup(t, releaseOpts{
			skips: []string{
				string(skips.Publish),
				string(skips.Sign),
				string(skips.Validate),
			},
		})

		requireAll(t, ctx, skips.Sign, skips.Publish, skips.Validate, skips.Announce)
	})

	t.Run("parallelism", func(t *testing.T) {
		require.Equal(t, 1, setup(t, releaseOpts{
			parallelism: 1,
		}).Parallelism)
	})

	t.Run("notes", func(t *testing.T) {
		notes := "foo.md"
		header := "header.md"
		footer := "footer.md"
		ctx := setup(t, releaseOpts{
			releaseNotesFile:  notes,
			releaseHeaderFile: header,
			releaseFooterFile: footer,
		})
		require.Equal(t, notes, ctx.ReleaseNotesFile)
		require.Equal(t, header, ctx.ReleaseHeaderFile)
		require.Equal(t, footer, ctx.ReleaseFooterFile)
	})

	t.Run("templated notes", func(t *testing.T) {
		notes := "foo.md"
		header := "header.md"
		footer := "footer.md"
		ctx := setup(t, releaseOpts{
			releaseNotesTmpl:  notes,
			releaseHeaderTmpl: header,
			releaseFooterTmpl: footer,
		})
		require.Equal(t, notes, ctx.ReleaseNotesTmpl)
		require.Equal(t, header, ctx.ReleaseHeaderTmpl)
		require.Equal(t, footer, ctx.ReleaseFooterTmpl)
	})

	t.Run("rm dist", func(t *testing.T) {
		require.True(t, setup(t, releaseOpts{
			clean: true,
		}).Clean)
	})
}
