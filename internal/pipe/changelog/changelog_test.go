package changelog

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Sort: "desc",
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.NotEmpty(t, ctx.Config.Changelog.Format)
		require.NotContains(t, ctx.Config.Changelog.Format, "Author")
	})
	t.Run("github", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use:  useGitHub,
				Sort: "asc",
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.NotEmpty(t, ctx.Config.Changelog.Format)
		require.Contains(t, ctx.Config.Changelog.Format, "Author")
	})
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestChangelogProvidedViaFlag(t *testing.T) {
	ctx := testctx.New()
	ctx.ReleaseNotesFile = "testdata/changes.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "c0ff33 coffee\n", ctx.ReleaseNotes)
}

func TestChangelogProvidedViaFlagIsAWhitespaceOnlyFile(t *testing.T) {
	ctx := testctx.New()
	ctx.ReleaseNotesFile = "testdata/changes-empty.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "\n", ctx.ReleaseNotes)
}

func TestChangelogProvidedViaFlagIsReallyEmpty(t *testing.T) {
	ctx := testctx.New()
	ctx.ReleaseNotesFile = "testdata/changes-really-empty.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.ReleaseNotes)
}

func TestChangelogTmplProvidedViaFlagIsReallyEmpty(t *testing.T) {
	ctx := testctx.New()
	ctx.ReleaseNotesTmpl = "testdata/changes-really-empty.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.ReleaseNotes)
}

func TestTemplatedChangelogProvidedViaFlag(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	ctx.ReleaseNotesFile = "testdata/changes.md"
	ctx.ReleaseNotesTmpl = "testdata/changes-templated.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "c0ff33 coffee v0.0.1\n", ctx.ReleaseNotes)
}

func TestTemplatedChangelogProvidedViaFlagResultIsEmpty(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	ctx.ReleaseNotesTmpl = "testdata/changes-templated-empty.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "\n\n", ctx.ReleaseNotes)
}

func TestChangelogProvidedViaFlagDoesntExist(t *testing.T) {
	ctx := testctx.New()
	ctx.ReleaseNotesFile = "testdata/changes.nope"
	require.NoError(t, Pipe{}.Default(ctx))
	require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
}

func TestReleaseHeaderProvidedViaFlagDoesntExist(t *testing.T) {
	ctx := testctx.New()
	ctx.ReleaseHeaderFile = "testdata/header.nope"
	require.NoError(t, Pipe{}.Default(ctx))
	require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
}

func TestReleaseFooterProvidedViaFlagDoesntExist(t *testing.T) {
	ctx := testctx.New()
	ctx.ReleaseFooterFile = "testdata/footer.nope"
	require.NoError(t, Pipe{}.Default(ctx))
	require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
}

func TestChangelog(t *testing.T) {
	folder := testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitCommit(t, `a commit message "with quotes inside it'`)
	testlib.GitCommit(t, `a " quote ' fiesta`)
	testlib.GitCommit(t, `an unclosed <tag somewhere`)
	testlib.GitTag(t, "v0.0.2")
	ctx := testctx.NewWithCfg(config.Project{
		Dist: folder,
		Changelog: config.Changelog{
			Use: "git",
			Filters: config.Filters{
				Exclude: []string{
					"docs:",
					"ignored:",
					"(?i)cars",
					"^Merge pull request",
				},
			},
		},
	}, testctx.WithCurrentTag("v0.0.2"), testctx.WithPreviousTag("v0.0.1"))
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.NotContains(t, ctx.ReleaseNotes, "first")
	require.Contains(t, ctx.ReleaseNotes, "added feature 1")
	require.Contains(t, ctx.ReleaseNotes, "fixed bug 2")
	require.Contains(t, ctx.ReleaseNotes, `a commit message "with quotes inside it'`)
	require.Contains(t, ctx.ReleaseNotes, `a " quote ' fiesta`)
	require.Contains(t, ctx.ReleaseNotes, "an unclosed <tag somewhere")
	require.NotContains(t, ctx.ReleaseNotes, "docs")
	require.NotContains(t, ctx.ReleaseNotes, "ignored")
	require.NotContains(t, ctx.ReleaseNotes, "cArs")
	require.NotContains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")

	for _, line := range strings.Split(ctx.ReleaseNotes, "\n")[1:] {
		if line == "" {
			continue
		}
		require.Truef(t, strings.HasPrefix(line, "* "), "%q: changelog commit must be a list item", line)
	}

	bts, err := os.ReadFile(filepath.Join(folder, "CHANGELOG.md"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}

func TestChangelogInclude(t *testing.T) {
	folder := testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitTag(t, "v0.0.2")
	ctx := testctx.NewWithCfg(config.Project{
		Dist: folder,
		Changelog: config.Changelog{
			Use: "git",
			Filters: config.Filters{
				Include: []string{
					"docs:",
					"ignored:",
					"(?i)cars",
					"^Merge pull request",
				},
			},
		},
	}, testctx.WithCurrentTag("v0.0.2"), testctx.WithPreviousTag("v0.0.1"))
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.NotContains(t, ctx.ReleaseNotes, "first")
	require.NotContains(t, ctx.ReleaseNotes, "added feature 1")
	require.NotContains(t, ctx.ReleaseNotes, "fixed bug 2")
	require.Contains(t, ctx.ReleaseNotes, "docs")
	require.Contains(t, ctx.ReleaseNotes, "ignored")
	require.Contains(t, ctx.ReleaseNotes, "cArs")
	require.Contains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")

	for _, line := range strings.Split(ctx.ReleaseNotes, "\n")[1:] {
		if line == "" {
			continue
		}
		require.Truef(t, strings.HasPrefix(line, "* "), "%q: changelog commit must be a list item", line)
	}

	bts, err := os.ReadFile(filepath.Join(folder, "CHANGELOG.md"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}

func TestChangelogForGitlab(t *testing.T) {
	folder := testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitTag(t, "v0.0.2")
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: folder,
			Changelog: config.Changelog{
				Filters: config.Filters{
					Exclude: []string{
						"docs:",
						"ignored:",
						"(?i)cars",
						"^Merge pull request",
					},
				},
			},
		},
		testctx.GitLabTokenType,
		testctx.WithCurrentTag("v0.0.2"),
		testctx.WithPreviousTag("v0.0.1"),
	)
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.NotContains(t, ctx.ReleaseNotes, "first")
	require.Contains(t, ctx.ReleaseNotes, "added feature 1") // no whitespace because its the last entry of the changelog
	require.Contains(t, ctx.ReleaseNotes, "fixed bug 2   ")  // whitespaces are on purpose
	require.NotContains(t, ctx.ReleaseNotes, "docs")
	require.NotContains(t, ctx.ReleaseNotes, "ignored")
	require.NotContains(t, ctx.ReleaseNotes, "cArs")
	require.NotContains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")

	bts, err := os.ReadFile(filepath.Join(folder, "CHANGELOG.md"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}

func TestChangelogSort(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "whatever")
	testlib.GitTag(t, "v0.9.9")
	testlib.GitCommit(t, "c: commit")
	testlib.GitCommit(t, "a: commit")
	testlib.GitCommit(t, "b: commit")
	testlib.GitTag(t, "v1.0.0")
	ctx := testctx.NewWithCfg(
		config.Project{
			Changelog: config.Changelog{
				Format: "{{.Message}}",
			},
		},
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithPreviousTag("v0.9.9"),
	)

	for _, cfg := range []struct {
		Sort    string
		Entries []string
	}{
		{
			Sort: "",
			Entries: []string{
				"b: commit",
				"a: commit",
				"c: commit",
			},
		},
		{
			Sort: "asc",
			Entries: []string{
				"a: commit",
				"b: commit",
				"c: commit",
			},
		},
		{
			Sort: "desc",
			Entries: []string{
				"c: commit",
				"b: commit",
				"a: commit",
			},
		},
	} {
		t.Run("changelog sort='"+cfg.Sort+"'", func(t *testing.T) {
			ctx.Config.Changelog.Sort = cfg.Sort
			log, err := buildChangelog(ctx)
			require.NoError(t, err)
			entries := strings.Split(strings.TrimSpace(log), "\n")
			var changes []string
			for _, line := range entries {
				if line == "" || line[0] == '#' {
					continue
				}
				changes = append(changes, strings.TrimPrefix(line, li))
			}
			require.Len(t, changes, len(cfg.Entries))
			require.Equal(t, cfg.Entries, changes)
		})
	}
}

func Benchmark_sortEntries(b *testing.B) {
	ctx := testctx.New()
	entries := []Item{
		{SHA: "cafebabe", Message: "added feature 1"},
		{SHA: "cafebabe", Message: "fixed bug 2"},
		{SHA: "cafebabe", Message: "ignored: whatever"},
		{SHA: "cafebabe", Message: "docs: whatever"},
		{SHA: "cafebabe", Message: "something about cArs we dont need"},
		{SHA: "cafebabe", Message: "feat: added that thing"},
		{SHA: "cafebabe", Message: "Merge pull request #999 from goreleaser/some-branch"},
		{SHA: "cafebabe", Message: "this is not a Merge pull request"},
	}

	b.Run("asc", func(b *testing.B) {
		ctx.Config.Changelog.Sort = "asc"
		for b.Loop() {
			sortEntries(ctx, entries)
		}
	})
	b.Run("desc", func(b *testing.B) {
		ctx.Config.Changelog.Sort = "desc"
		for b.Loop() {
			sortEntries(ctx, entries)
		}
	})
}

func TestChangelogInvalidSort(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Changelog: config.Changelog{
			Sort: "dope",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), ErrInvalidSortDirection.Error())
}

func TestChangelogOfFirstRelease(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	msgs := []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		require.Contains(t, ctx.ReleaseNotes, msg)
	}
}

func TestChangelogFilterInvalidRegex(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "commitssss")
	testlib.GitTag(t, "v0.0.3")
	testlib.GitCommit(t, "commitzzz")
	testlib.GitTag(t, "v0.0.4")
	ctx := testctx.NewWithCfg(config.Project{
		Changelog: config.Changelog{
			Filters: config.Filters{
				Exclude: []string{
					"(?iasdr4qasd)not a valid regex i guess",
				},
			},
		},
	}, testctx.WithCurrentTag("v0.0.4"), testctx.WithPreviousTag("v0.0.3"))
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "error parsing regexp: invalid or unsupported Perl syntax: `(?ia`")
}

func TestChangelogFilterIncludeInvalidRegex(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "commitssss")
	testlib.GitTag(t, "v0.0.3")
	testlib.GitCommit(t, "commitzzz")
	testlib.GitTag(t, "v0.0.4")
	ctx := testctx.NewWithCfg(config.Project{
		Changelog: config.Changelog{
			Filters: config.Filters{
				Include: []string{
					"(?iasdr4qasd)not a valid regex i guess",
				},
			},
		},
	}, testctx.WithCurrentTag("v0.0.4"), testctx.WithPreviousTag("v0.0.3"))
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "error parsing regexp: invalid or unsupported Perl syntax: `(?ia`")
}

func TestChangelogNoTags(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	msgs := []string{"first", "second", "third"}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.NotEmpty(t, ctx.ReleaseNotes)
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		require.Contains(t, ctx.ReleaseNotes, msg)
	}
}

func TestChangelogOnBranchWithSameNameAsTag(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	msgs := []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		require.Contains(t, ctx.ReleaseNotes, msg)
	}
}

func TestChangeLogWithReleaseHeader(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir := testlib.Mktmp(t)
	require.NoError(t, os.Symlink(current+"/testdata", tmpdir+"/testdata"))
	testlib.GitInit(t)
	msgs := []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	ctx.ReleaseHeaderFile = "testdata/release-header.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Contains(t, ctx.ReleaseNotes, "test header")
}

func TestChangeLogWithTemplatedReleaseHeader(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir := testlib.Mktmp(t)
	require.NoError(t, os.Symlink(current+"/testdata", tmpdir+"/testdata"))
	testlib.GitInit(t)
	msgs := []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	ctx.ReleaseHeaderTmpl = "testdata/release-header-templated.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Contains(t, ctx.ReleaseNotes, "test header with tag v0.0.1")
}

func TestChangeLogWithReleaseFooter(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir := testlib.Mktmp(t)
	require.NoError(t, os.Symlink(current+"/testdata", tmpdir+"/testdata"))
	testlib.GitInit(t)
	msgs := []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	ctx.ReleaseFooterFile = "testdata/release-footer.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Contains(t, ctx.ReleaseNotes, "test footer")
	require.Equal(t, '\n', rune(ctx.ReleaseNotes[len(ctx.ReleaseNotes)-1]))
}

func TestChangeLogWithTemplatedReleaseFooter(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir := testlib.Mktmp(t)
	require.NoError(t, os.Symlink(current+"/testdata", tmpdir+"/testdata"))
	testlib.GitInit(t)
	msgs := []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	ctx.ReleaseFooterTmpl = "testdata/release-footer-templated.md"
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Contains(t, ctx.ReleaseNotes, "test footer with tag v0.0.1")
	require.Equal(t, '\n', rune(ctx.ReleaseNotes[len(ctx.ReleaseNotes)-1]))
}

func TestChangeLogWithoutReleaseFooter(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir := testlib.Mktmp(t)
	require.NoError(t, os.Symlink(current+"/testdata", tmpdir+"/testdata"))
	testlib.GitInit(t)
	msgs := []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	ctx := testctx.New(testctx.WithCurrentTag("v0.0.1"), withFirstCommit(t))
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Equal(t, '\n', rune(ctx.ReleaseNotes[len(ctx.ReleaseNotes)-1]))
}

func TestGetChangelogGitHubNative(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Changelog: config.Changelog{
			Use: useGitHubNative,
		},
	}, testctx.WithCurrentTag("v0.180.2"), testctx.WithPreviousTag("v0.180.1"))

	expected := `## What's changed

* Foo bar test

**Full Changelog**: https://github.com/gorelease/goreleaser/compare/v0.180.1...v0.180.2
`
	mock := client.NewMock()
	mock.ReleaseNotes = expected
	l := githubNativeChangeloger{
		client: mock,
		repo: client.Repo{
			Owner: "goreleaser",
			Name:  "goreleaser",
		},
	}
	log, err := l.Log(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, log)
	require.Equal(t, []string{"v0.180.1", "v0.180.2"}, mock.ReleaseNotesParams)
}

func TestGetChangelogGitHubNativeFirstRelease(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Changelog: config.Changelog{
			Use: useGitHubNative,
		},
	}, testctx.WithCurrentTag("v0.1.0"))

	expected := `## What's changed

* Foo bar test

**Full Changelog**: https://github.com/gorelease/goreleaser/commits/v0.1.0
`
	mock := client.NewMock()
	mock.ReleaseNotes = expected
	l := githubNativeChangeloger{
		client: mock,
		repo: client.Repo{
			Owner: "goreleaser",
			Name:  "goreleaser",
		},
	}
	log, err := l.Log(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, log)
	require.Equal(t, []string{"", "v0.1.0"}, mock.ReleaseNotesParams)
}

func TestGetChangeloger(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		c, err := getChangeloger(testctx.New())
		require.NoError(t, err)
		require.IsType(t, gitChangeloger{}, c)
	})

	t.Run(useGit, func(t *testing.T) {
		c, err := getChangeloger(testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: useGit,
			},
		}))
		require.NoError(t, err)
		require.IsType(t, gitChangeloger{}, c)
	})

	t.Run(useGitHub, func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: useGitHub,
			},
		}, testctx.GitHubTokenType, testctx.WithPreviousTag("v1.2.3"))
		c, err := getChangeloger(ctx)
		require.NoError(t, err)
		require.IsType(t, &scmChangeloger{}, c)
	})

	t.Run(useGitHub+" no previous", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: useGitHub,
			},
		}, testctx.GitHubTokenType)
		c, err := getChangeloger(ctx)
		require.NoError(t, err)
		require.IsType(t, gitChangeloger{}, c)
	})

	t.Run(useGitHubNative, func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: useGitHubNative,
			},
		}, testctx.GitHubTokenType, testctx.WithPreviousTag("v1.2.3"))
		c, err := newGithubChangeloger(ctx)
		require.NoError(t, err)
		require.IsType(t, &githubNativeChangeloger{}, c)
	})

	t.Run(useGitHubNative+"-invalid-repo", func(t *testing.T) {
		testlib.Mktmp(t)
		testlib.GitInit(t)
		testlib.GitRemoteAdd(t, "https://gist.github.com/")
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: useGitHubNative,
			},
		}, testctx.GitHubTokenType)
		c, err := newGithubChangeloger(ctx)
		require.EqualError(t, err, "unsupported repository URL: https://gist.github.com/")
		require.Nil(t, c)
	})

	t.Run(useGitLab, func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: useGitLab,
			},
		}, testctx.GitLabTokenType, testctx.WithPreviousTag("v1.2.3"))
		c, err := getChangeloger(ctx)
		require.NoError(t, err)
		require.IsType(t, &scmChangeloger{}, c)
	})

	t.Run(useGitHub+"-invalid-repo", func(t *testing.T) {
		testlib.Mktmp(t)
		testlib.GitInit(t)
		testlib.GitRemoteAdd(t, "https://gist.github.com/")
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: useGitHub,
			},
		}, testctx.GitHubTokenType, testctx.WithPreviousTag("v1.2.3"))
		c, err := getChangeloger(ctx)
		require.EqualError(t, err, "unsupported repository URL: https://gist.github.com/")
		require.Nil(t, c)
	})

	t.Run(useGitea, func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if strings.HasSuffix(r.URL.Path, "api/v1/version") {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "{\"version\":\"1.22.0\"}")
			}
		}))
		defer srv.Close()
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: useGitea,
			},
			GiteaURLs: config.GiteaURLs{
				API: srv.URL,
			},
		}, testctx.GiteaTokenType, testctx.WithPreviousTag("v1.2.3"))
		c, err := getChangeloger(ctx)
		require.NoError(t, err)
		require.IsType(t, &scmChangeloger{}, c)
	})

	t.Run("invalid", func(t *testing.T) {
		c, err := getChangeloger(testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Use: "nope",
			},
		}))
		require.EqualError(t, err, `invalid changelog.use: "nope"`)
		require.Nil(t, c)
	})
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.New(testctx.Snapshot)
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("skip/disable", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Disable: "{{gt .Patch 0}}",
			},
		}, testctx.WithSemver(0, 0, 1, ""))
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("disable on patches", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Disable: "{{gt .Patch 0}}",
			},
		}, testctx.WithSemver(0, 0, 1, ""))
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("invalid template", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Disable: "{{if eq .Patch 123}",
			},
		}, testctx.WithSemver(0, 0, 1, ""))
		_, err := Pipe{}.Skip(ctx)
		require.Error(t, err)
	})

	t.Run("dont skip", func(t *testing.T) {
		b, err := Pipe{}.Skip(testctx.New())
		require.NoError(t, err)
		require.False(t, b)
	})

	t.Run("dont skip based on template", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Changelog: config.Changelog{
				Disable: "{{gt .Patch 0}}",
			},
		})
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, b)
	})
}

func TestGroup(t *testing.T) {
	folder := testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "feat(deps): update foobar [bot]")
	testlib.GitCommit(t, "fix: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "chore: something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "bug: Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitTag(t, "v0.0.2")
	ctx := testctx.NewWithCfg(config.Project{
		Dist: folder,
		Changelog: config.Changelog{
			Groups: []config.ChangelogGroup{
				{
					Title:  "Bots",
					Regexp: ".*bot.*",
					Order:  900,
				},
				{
					Title:  "Features",
					Regexp: `^.*?feat(\([[:word:]]+\))??!?:.+$`,
					Order:  0,
				},
				{
					Title:  "Bug Fixes",
					Regexp: `^.*?bug(\([[:word:]]+\))??!?:.+$`,
					Order:  1,
				},
				{
					Title:  "Catch nothing",
					Regexp: "yada yada yada honk the planet",
					Order:  10,
				},
				{
					Title: "Others",
					Order: 999,
				},
			},
		},
	}, testctx.WithCurrentTag("v0.0.2"), withFirstCommit(t))
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Regexp(t, `## Changelog
### Features
\* \w+ feat: added that thing
### Bug Fixes
\* \w+ bug: Merge pull request #999 from goreleaser\/some-branch
### Bots
\* \w+ feat\(deps\): update foobar \[bot\]
### Others
\* \w+ this is not a Merge pull request
\* \w+ chore: something about cArs we dont need
\* \w+ docs: whatever
\* \w+ fix: whatever
\* \w+ ignored: whatever
\* \w+ fixed bug 2
\* \w+ added feature 1
\* \w+ first
`, ctx.ReleaseNotes)
}

func TestGroupBadRegex(t *testing.T) {
	folder := testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitTag(t, "v0.0.2")
	ctx := testctx.NewWithCfg(config.Project{
		Dist: folder,
		Changelog: config.Changelog{
			Groups: []config.ChangelogGroup{
				{
					Title:  "Something",
					Regexp: "^.*feat[a-z", // unterminated regex
				},
			},
		},
	}, testctx.WithCurrentTag("v0.0.2"), withFirstCommit(t))
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "failed to group into \"Something\": error parsing regexp: missing closing ]: `[a-z`")
}

func TestChangelogFormat(t *testing.T) {
	t.Run("without groups", func(t *testing.T) {
		makeConf := func(u string) config.Project {
			return config.Project{
				Changelog: config.Changelog{
					Use:    u,
					Format: "{{.SHA}} {{.Message}}",
				},
			}
		}

		for _, use := range []string{useGit, useGitHub, useGitLab, useGitea} {
			t.Run(use, func(t *testing.T) {
				out, err := formatChangelog(
					testctx.NewWithCfg(makeConf(use)),
					[]Item{
						{SHA: "aea123", Message: "foo"},
						{SHA: "aef653", Message: "bar"},
					},
				)
				require.NoError(t, err)
				require.Equal(t, `## Changelog
* aea123 foo
* aef653 bar`, out)
			})
		}
	})

	t.Run("with groups", func(t *testing.T) {
		makeConf := func(u string) config.Project {
			return config.Project{
				Changelog: config.Changelog{
					Use:    u,
					Format: "{{.SHA}} {{.Message}}",
					Groups: []config.ChangelogGroup{
						{Title: "catch-all"},
					},
				},
			}
		}

		for _, use := range []string{useGit, useGitHub, useGitLab, useGitea} {
			t.Run(use, func(t *testing.T) {
				out, err := formatChangelog(
					testctx.NewWithCfg(makeConf(use)),
					[]Item{
						{SHA: "aea123", Message: "foo"},
						{SHA: "aef653", Message: "bar"},
					},
				)
				require.NoError(t, err)
				require.Equal(t, `## Changelog
### catch-all
* aea123 foo
* aef653 bar`, out)
			})
		}
	})
}

func TestAbbrev(t *testing.T) {
	folder := testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "feat(deps): update foobar [bot]")
	testlib.GitCommit(t, "fix: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "chore: something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "bug: Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitTag(t, "v0.0.2")

	t.Run("no abbrev", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:      folder,
			Changelog: config.Changelog{},
		}, testctx.WithCurrentTag("v0.0.2"), withFirstCommit(t))

		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		ensureCommitHashLen(t, ctx.ReleaseNotes, 40)
	})

	t.Run("abbrev -1", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: folder,
			Changelog: config.Changelog{
				Abbrev: -1,
			},
		}, testctx.WithCurrentTag("v0.0.2"), withFirstCommit(t))
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("abbrev 3", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: folder,
			Changelog: config.Changelog{
				Abbrev: 3,
			},
		}, testctx.WithCurrentTag("v0.0.2"), withFirstCommit(t))
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		ensureCommitHashLen(t, ctx.ReleaseNotes, 3)
	})

	t.Run("abbrev 7", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: folder,
			Changelog: config.Changelog{
				Abbrev: 7,
			},
		}, testctx.WithCurrentTag("v0.0.2"), withFirstCommit(t))
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		ensureCommitHashLen(t, ctx.ReleaseNotes, 7)
	})

	t.Run("abbrev 50", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: folder,
			Changelog: config.Changelog{
				Abbrev: 50,
			},
		}, testctx.WithCurrentTag("v0.0.2"), withFirstCommit(t))
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		ensureCommitHashLen(t, ctx.ReleaseNotes, 40)
	})
}

func TestIssue5595(t *testing.T) {
	for name, format := range map[string]string{
		"abbrev-sha": "[{{.SHA}}]: {{.Message}} (@{{.AuthorName}})",
		"no-sha":     "{{.Message}} (@{{.AuthorName}})",
	} {
		t.Run(name, func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				Changelog: config.Changelog{
					Use:    useGitHub,
					Format: format,
					Abbrev: 3,
					Groups: []config.ChangelogGroup{
						{
							Title:  "Features",
							Regexp: `^.*?feat(\([[:word:]]+\))??!?:.+$`,
							Order:  0,
						},
						{
							Title:  "Fixes",
							Regexp: `^.*?fix(\([[:word:]]+\))??!?:.+$`,
							Order:  1,
						},
						{
							Title: "Others",
							Order: 999,
						},
					},
					Filters: config.Filters{
						Exclude: []string{
							"^docs:",
							"typo",
							"(?i)foo",
						},
						Include: []string{
							"^feat:",
							"^fix:",
						},
					},
				},
			}, testctx.WithCurrentTag("v0.0.2"), withFirstCommit(t))
			require.NoError(t, Pipe{}.Default(ctx))

			mock := client.NewMock()

			for i := range 20 {
				kind := "fix"
				if i%2 == 0 {
					kind = "feat"
				}
				if i%5 == 0 {
					kind = "chore"
				}
				if i%7 == 0 {
					kind = "docs"
				}
				msg := fmt.Sprintf("%s: commit #%d", kind, i)
				mock.Changes = append(mock.Changes, Item{
					SHA:            "cafebabe",
					Message:        msg,
					AuthorName:     "Carlos",
					AuthorEmail:    "nope@nope.com",
					AuthorUsername: "caarlos0",
				})
			}

			cl := wrappingChangeloger{
				changeloger: &scmChangeloger{
					client: mock,
					repo: client.Repo{
						Owner: "test",
						Name:  "test",
					},
				},
			}

			log, err := cl.Log(ctx)
			require.NoError(t, err)
			golden.RequireEqualExt(t, []byte(log), ".md")
		})
	}
}

func ensureCommitHashLen(tb testing.TB, log string, l int) {
	tb.Helper()
	for line := range strings.SplitSeq(log, "\n") {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		commit := strings.TrimPrefix(parts[1], "* ")
		commit = strings.TrimSuffix(commit, ":")
		require.Len(tb, commit, l)
	}
}

func withFirstCommit(tb testing.TB) testctx.Opt {
	tb.Helper()
	return func(ctx *context.Context) {
		s, err := git.Clean(git.Run(testctx.New(), "rev-list", "--max-parents=0", "HEAD"))
		require.NoError(tb, err)
		ctx.Git.FirstCommit = s
	}
}
