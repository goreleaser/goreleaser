package winget

import (
	"html/template"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("should", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.WrapWithCfg(t.Context(), config.Project{
			Winget: []config.Winget{{}},
		}, testctx.Skip(skips.Winget))))
	})
	t.Run("should not", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.WrapWithCfg(t.Context(), config.Project{
			Winget: []config.Winget{{}},
		})))
	})
}

func TestRunPipe(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		expectRunErrorIs     error
		expectPublishErrorIs error
		expectPath           string
		winget               config.Winget
	}{
		{
			name:       "minimal",
			expectPath: "manifests/f/Foo/min/1.2.1/Foo.min.",
			winget: config.Winget{
				Name:             "min",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "mixed-formats",
			expectRunErrorIs: errMixedFormats,
			winget: config.Winget{
				Name:             "mixed",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"zaz", "bar"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:       "package id and name",
			expectPath: "manifests/b/beckersoft/foo/1.2.1",
			winget: config.Winget{
				Name:                "foo",
				Publisher:           "Beckersoft LTDA",
				PackageIdentifier:   "beckersoft.foo",
				PackageName:         "Foo Bar Pkg",
				PublisherURL:        "https://carlosbecker.com",
				PublisherSupportURL: "https://carlosbecker.com/support",
				PrivacyURL:          "https://carlosbecker.com/privacy",
				Copyright:           "bla bla bla",
				CopyrightURL:        "https://goreleaser.com/copyright",
				Author:              "Carlos Becker",
				Repository:          config.RepoRef{Owner: "foo", Name: "bar"},
				CommitAuthor:        config.CommitAuthor{},
				IDs:                 []string{"foo"},
				Goamd64:             "v1",
				SkipUpload:          "false",
				ShortDescription:    "foo",
				Description: `long foo bar

				yadaa yada yada loooaaasssss

				sss`,
				Homepage:          "https://goreleaser.com",
				License:           "MIT",
				LicenseURL:        "https://goreleaser.com/eula/",
				ReleaseNotesURL:   "https://github.com/goreleaser/goreleaser/tags/{{.Tag}}",
				ReleaseNotes:      "{{.Changelog}}",
				InstallationNotes: "https://goreleaser.com/install/",
				Tags:              []string{"foo", "bar", "foo bar baz"},
			},
		},
		{
			name:       "full",
			expectPath: "manifests/b/Beckersoft LTDA/foo/1.2.1",
			winget: config.Winget{
				Name:                "foo",
				Publisher:           "Beckersoft",
				PublisherURL:        "https://carlosbecker.com",
				PublisherSupportURL: "https://carlosbecker.com/support",
				PrivacyURL:          "https://carlosbecker.com/privacy",
				Copyright:           "bla bla bla",
				CopyrightURL:        "https://goreleaser.com/copyright",
				Author:              "Carlos Becker",
				Path:                "manifests/b/Beckersoft LTDA/foo/{{.Version}}",
				Repository:          config.RepoRef{Owner: "foo", Name: "bar"},
				CommitAuthor:        config.CommitAuthor{},
				IDs:                 []string{"foo"},
				Goamd64:             "v1",
				SkipUpload:          "false",
				ShortDescription:    "foo",
				Description: `long foo bar

				yadaa yada yada loooaaasssss

				sss`,
				Homepage:          "https://goreleaser.com",
				License:           "MIT",
				LicenseURL:        "https://goreleaser.com/eula/",
				ReleaseNotesURL:   "https://github.com/goreleaser/goreleaser/tags/{{.Tag}}",
				ReleaseNotes:      "{{.Changelog}}",
				InstallationNotes: "https://goreleaser.com/install/",
				Tags:              []string{"Foo", "bar", "FoO BaSiMdrR LAmd"},
			},
		},
		{
			name:       "default-locale",
			expectPath: "manifests/f/Foo/loc/1.2.1/Foo.loc.",
			winget: config.Winget{
				Name:             "loc",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				DefaultLocale:    "en-GB",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "open-pr",
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				IDs:              []string{"foo"},
				Description:      "my test",
				Homepage:         "https://goreleaser.com",
				License:          "mit",
				Path:             "pkgs/foo.winget",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner:  "foo",
					Name:   "bar",
					Branch: "update-{{.Version}}",
					PullRequest: config.PullRequest{
						Enabled: true,
						Base: config.PullRequestBase{
							Owner: "ms",
							Name:  "winget",
						},
					},
				},
			},
		},
		{
			name: "wrapped-in-dir",
			winget: config.Winget{
				Name:             "wrapped-in-dir",
				Publisher:        "Beckersoft",
				IDs:              []string{"wrapped-in-dir"},
				Description:      "my test",
				Homepage:         "https://goreleaser.com",
				License:          "mit",
				LicenseURL:       "https://goreleaser.com/license",
				ReleaseNotesURL:  "https://github.com/goreleaser/goreleaser/tags/{{.Tag}}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "no-archives",
			expectRunErrorIs: errNoArchivesFound{
				goamd64: "v2",
				ids:     []string{"nopenopenope"},
			},
			winget: config.Winget{
				Name:             "no-archives",
				Publisher:        "Beckersoft",
				IDs:              []string{"nopenopenope"},
				Goamd64:          "v2",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "too-many-archives",
			expectRunErrorIs: errMultipleArchives,
			winget: config.Winget{
				Name:             "min",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "partial",
			winget: config.Winget{
				Name:             "partial",
				Publisher:        "Beckersoft",
				IDs:              []string{"partial"},
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "no-repo-name",
			expectRunErrorIs: errNoRepoName,
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
				},
			},
		},
		{
			name:             "no-license",
			expectRunErrorIs: errNoLicense,
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				ShortDescription: "aa",
				Repository: config.RepoRef{
					Name:  "foo",
					Owner: "foo",
				},
			},
		},
		{
			name:             "no-short-description",
			expectRunErrorIs: errNoShortDescription,
			winget: config.Winget{
				Name:      "doesnotmatter",
				Publisher: "Beckersoft",
				License:   "MIT",
				Repository: config.RepoRef{
					Name:  "foo",
					Owner: "foo",
				},
			},
		},
		{
			name:             "invalid-package-identifier",
			expectRunErrorIs: errInvalidPackageIdentifier,
			winget: config.Winget{
				Name:              "min",
				PackageIdentifier: "foobar",
				Publisher:         "Foo",
				License:           "MIT",
				ShortDescription:  "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "no-publisher",
			expectRunErrorIs: errNoPublisher,
			winget: config.Winget{
				Name:             "min",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-name-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "{{ .Nope }}",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-releasenotes-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				ReleaseNotes:     "{{ .Nope }}",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-publisher-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-publisher-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				PublisherURL:     "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-author-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foobar",
				Publisher:        "Beckersoft",
				Author:           "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-homepage-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foobar",
				Publisher:        "Beckersoft",
				Homepage:         "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-description-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foobar",
				Publisher:        "Beckersoft",
				Description:      "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-short-description-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foobar",
				Publisher:        "Beckersoft",
				ShortDescription: "{{ .Nope }}",
				License:          "MIT",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-repo-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "{{ .Nope }}",
				},
			},
		},
		{
			name:             "bad-skip-upload-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				SkipUpload:       "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-release-notes-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				ReleaseNotesURL:  `https://goo/bar/asdfsd/{{.nope}}`,
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-release-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				URLTemplate:      "{{.BadURL}}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-path-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "MIT",
				Path:             "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "bad-commit-msg-tmpl",
			expectPublishErrorIs: &template.Error{},
			winget: config.Winget{
				Name:                  "foo",
				Publisher:             "Beckersoft",
				License:               "MIT",
				ShortDescription:      "foo bar zaz",
				CommitMessageTemplate: "{{.Foo}}",
				IDs:                   []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-publisher-support-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:                "foo",
				Publisher:           "Beckersoft",
				PublisherSupportURL: "{{.Nope}}",
				License:             "MIT",
				ShortDescription:    "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-copyright-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "MIT",
				Copyright:        "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-copyright-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "{{ .Nope }}",
				Publisher:        "Beckersoft",
				License:          "MIT",
				CopyrightURL:     "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-license-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-license-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "MIT",
				LicenseURL:       "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-default-locale-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "MIT",
				DefaultLocale:    "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "skip-upload",
			expectPublishErrorIs: errSkipUpload,
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				SkipUpload:       "true",
				License:          "MIT",
				IDs:              []string{"foo"},
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "skip-upload-auto",
			expectPublishErrorIs: errSkipUploadAuto,
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				SkipUpload:       "auto",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:       "with-deps",
			expectPath: "manifests/f/Foo/deps/1.2.1/Foo.deps.",
			winget: config.Winget{
				Name:             "deps",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				Dependencies: []config.WingetDependency{
					{
						PackageIdentifier: "foo.bar",
						MinimumVersion:    "1.2.3",
					},
					{
						PackageIdentifier: "zaz.bar",
					},
				},
			},
		},
		{
			name:             "bad-dependency-template",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "MIT",
				LicenseURL:       "https://foo.bar",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				Dependencies: []config.WingetDependency{
					{PackageIdentifier: "{{.Nope}}"},
				},
			},
		},
		{
			name:       "with-additional-locales",
			expectPath: "manifests/f/Foo/additional/1.2.1/Foo.additional.",
			winget: config.Winget{
				Name:             "additional",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{
						Locale:           "pt-BR",
						ShortDescription: "foo bar zaz pt-BR",
						Tags:             []string{"foo", "bar"},
					},
				},
			},
		},
		{
			name:       "with-multiple-additional-locales",
			expectPath: "manifests/f/Foo/multi-locales/1.2.1/Foo.multi-locales.",
			winget: config.Winget{
				Name:             "multi-locales",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{
						Locale:           "pt-BR",
						ShortDescription: "portuguese description",
					},
					{
						Locale:           "fr-FR",
						ShortDescription: "french description",
						Tags:             []string{"cli", "french"},
					},
				},
			},
		},
		{
			name:       "full-additional-locale",
			expectPath: "manifests/f/Foo/full-locales/1.2.1/Foo.full-locales.",
			winget: config.Winget{
				Name:                "full-locales",
				Publisher:           "Foo",
				PublisherURL:        "https://foo.com",
				PublisherSupportURL: "https://foo.com/support",
				PrivacyURL:          "https://foo.com/privacy",
				Author:              "Foo Author",
				Homepage:            "https://foo.com",
				License:             "MIT",
				LicenseURL:          "https://foo.com/license",
				Copyright:           "Copyright Foo",
				CopyrightURL:        "https://foo.com/copyright",
				ShortDescription:    "foo bar zaz",
				Description:         "default description",
				ReleaseNotesURL:     "https://foo.com/releases",
				InstallationNotes:   "https://foo.com/install",
				Tags:                []string{"default"},
				IDs:                 []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{
						Locale:              "pt-BR",
						Publisher:           "Foo Brasil",
						PublisherURL:        "https://foo.com.br",
						PublisherSupportURL: "https://foo.com.br/support",
						PrivacyURL:          "https://foo.com.br/privacy",
						Author:              "Author Foo",
						PackageName:         "foo-br",
						Homepage:            "https://foo.com.br",
						License:             "MIT BR",
						LicenseURL:          "https://foo.com.br/license",
						Copyright:           "Copyright BR",
						CopyrightURL:        "https://foo.com.br/copyright",
						ShortDescription:    "descricao curta",
						Description:         "descricao completa",
						Tags:                []string{"br", "cli"},
						ReleaseNotes:        "notas de lancamento",
						ReleaseNotesURL:     "https://foo.com.br/releases",
						InstallationNotes:   "https://foo.com.br/install",
					},
				},
			},
		},
		{
			name:       "additional-locale-description-fallback",
			expectPath: "manifests/f/Foo/desc-fallback/1.2.1/Foo.desc-fallback.",
			winget: config.Winget{
				Name:             "desc-fallback",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Description:      "default description",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{
						Locale:           "pt-BR",
						ShortDescription: "descricao curta",
					},
				},
			},
		},
		{
			name:             "bad-additional-locale-field-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "bad-field-locale",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{Locale: "pt-BR", Publisher: "{{ .Nope }}"},
				},
			},
		},
		{
			name:             "bad-additional-locale-release-notes-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "bad-release-notes-locale",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{Locale: "pt-BR", ReleaseNotes: "{{ .Nope }}"},
				},
			},
		},
		{
			name:             "additional-locale-empty",
			expectRunErrorIs: errAdditionalLocaleEmpty,
			winget: config.Winget{
				Name:             "empty-locale",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{Locale: ""},
				},
			},
		},
		{
			name:             "additional-locale-duplicate",
			expectRunErrorIs: errAdditionalLocaleDuplicate,
			winget: config.Winget{
				Name:             "dup-locale",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{Locale: "pt-BR"},
					{Locale: "pt-BR"},
				},
			},
		},
		{
			name:             "additional-locale-is-default",
			expectRunErrorIs: errAdditionalLocaleIsDefault,
			winget: config.Winget{
				Name:             "default-clash",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				DefaultLocale:    "en-GB",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{Locale: "en-GB"},
				},
			},
		},
		{
			name:             "bad-additional-locale-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "bad-tmpl-locale",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{Locale: "{{.Nope}}"},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.WrapWithCfg(t.Context(),
				config.Project{
					Dist:        folder,
					ProjectName: "foo",
					Winget:      []config.Winget{tt.winget},
				},
				testctx.WithVersion("1.2.1"),
				testctx.WithCurrentTag("v1.2.1"),
				testctx.WithSemver(1, 2, 1, "rc1"),
				testctx.WithDate(time.Date(2023, 6, 12, 20, 32, 10, 12, time.Local)))

			ctx.ReleaseNotes = "the changelog for this release..."
			createFakeArtifact := func(id, goos, goarch, goamd64, goarm string, extra map[string]any) {
				path := filepath.Join(folder, "dist/foo_"+goos+goarch+goamd64+goarm+".zip")
				art := artifact.Artifact{
					Name:    "foo_" + goos + "_" + goarch + goamd64 + goarm + ".zip",
					Path:    path,
					Goos:    goos,
					Goarch:  goarch,
					Goarm:   goarm,
					Goamd64: goamd64,
					Type:    artifact.UploadableArchive,
					Extra: map[string]any{
						artifact.ExtraID:        id,
						artifact.ExtraFormat:    "zip",
						artifact.ExtraBinaries:  []string{"foo.exe"},
						artifact.ExtraWrappedIn: "",
					},
				}
				maps.Copy(art.Extra, extra)
				ctx.Artifacts.Add(&art)

				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				f, err := os.Create(path)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			goos := "windows"
			goarch := "amd64"
			createFakeArtifact("partial", goos, goarch, "v1", "", nil)
			createFakeArtifact("foo", goos, goarch, "v1", "", nil)
			createFakeArtifact("zaz", goos, goarch, "v1", "", nil)
			createFakeArtifact("wrapped-in-dir", goos, goarch, "v1", "", map[string]any{
				artifact.ExtraWrappedIn: "foo",
				artifact.ExtraBinaries:  []string{"bin/foo.exe"},
			})

			goarch = "386"
			createFakeArtifact("foo", goos, goarch, "", "", nil)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bar.exe",
				Path:   "doesnt-matter",
				Goos:   goos,
				Goarch: goarch,
				Type:   artifact.UploadableBinary,
				Extra: map[string]any{
					artifact.ExtraID: "bar",
				},
			})
			createFakeArtifact("bar", goos, goarch, "v1", "", nil)
			createFakeArtifact("wrapped-in-dir", goos, goarch, "", "", map[string]any{
				artifact.ExtraWrappedIn: "foo",
				artifact.ExtraBinaries:  []string{"bin/foo.exe"},
			})

			goarch = "arm64"
			createFakeArtifact("foo", goos, goarch, "", "", nil)
			createFakeArtifact("wrapped-in-dir", goos, goarch, "", "", map[string]any{
				artifact.ExtraWrappedIn: "foo",
				artifact.ExtraBinaries:  []string{"bin/foo.exe"},
			})

			client := client.NewMock()
			pipe := Pipe{}

			// default
			require.NoError(t, pipe.Default(ctx))

			// run
			if tt.expectRunErrorIs != nil {
				err := pipe.runAll(ctx, client)
				require.Error(t, err)
				require.ErrorAs(t, err, &tt.expectPublishErrorIs)
				return
			}

			require.NoError(t, pipe.runAll(ctx, client))
			for _, winget := range ctx.Artifacts.Filter(artifact.ByTypes(
				artifact.WingetInstaller,
				artifact.WingetVersion,
				artifact.WingetDefaultLocale,
				artifact.WingetLocale,
			)).List() {
				bts, err := os.ReadFile(winget.Path)
				require.NoError(t, err)
				locale := artifact.ExtraOr(*winget, wingetLocaleExtra, "")
				if locale == "" {
					cfg := artifact.MustExtra[config.Winget](*winget, wingetConfigExtra)
					locale = cfg.DefaultLocale
				}
				golden.RequireEqualExtSubfolder(t, bts, extFor(winget.Type, locale))
			}

			// publish
			if tt.expectPublishErrorIs != nil {
				err := pipe.publishAll(ctx, client)
				require.Error(t, err)
				require.ErrorAs(t, err, &tt.expectPublishErrorIs)
				return
			}
			require.NoError(t, pipe.publishAll(ctx, client))
			require.True(t, client.CreatedFile)

			expected := map[string]int{
				"locale":    1 + len(tt.winget.AdditionalLocales),
				"version":   1,
				"installer": 1,
			}
			require.Len(t, client.Messages, expected["version"]+expected["installer"]+expected["locale"])
			for _, msg := range client.Messages {
				require.Regexp(t, "New version: \\w+\\.[\\w-]+ 1.2.1", msg)
				for k, v := range expected {
					if strings.Contains(msg, ": add "+k) {
						expected[k] = v - 1
					}
				}
			}
			for k, v := range expected {
				require.Equalf(t, 0, v, "missing commit for %s", k)
			}

			require.NotEmpty(t, client.Path)
			if tt.expectPath != "" {
				require.Truef(t, strings.HasPrefix(client.Path, tt.expectPath), "expected %q to begin with %q", client.Path, tt.expectPath)
			}

			if tt.winget.Repository.PullRequest.Enabled {
				require.True(t, client.SyncedFork)
				require.True(t, client.OpenedPullRequest)
			}
		})
	}
}

func TestRunNoArtifactsOnInvalidAdditionalLocale(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Winget: []config.Winget{{
				Name:             "foo",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				AdditionalLocales: []config.WingetLocale{
					{Locale: "pt-BR"},
					{Locale: "pt-BR"},
				},
			}},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
		testctx.WithSemver(1, 2, 1, "rc1"),
		testctx.WithDate(time.Date(2023, 6, 12, 20, 32, 10, 12, time.Local)))

	createFakeArtifact := func(id, goos, goarch, goamd64 string) {
		path := filepath.Join(folder, "dist/foo_"+goos+goarch+goamd64+".zip")
		art := artifact.Artifact{
			Name:    "foo_" + goos + "_" + goarch + goamd64 + ".zip",
			Path:    path,
			Goos:    goos,
			Goarch:  goarch,
			Goamd64: goamd64,
			Type:    artifact.UploadableArchive,
			Extra: map[string]any{
				artifact.ExtraID:        id,
				artifact.ExtraFormat:    "zip",
				artifact.ExtraBinaries:  []string{"foo.exe"},
				artifact.ExtraWrappedIn: "",
			},
		}
		ctx.Artifacts.Add(&art)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}
	createFakeArtifact("foo", "windows", "amd64", "v1")

	pipe := Pipe{}
	require.NoError(t, pipe.Default(ctx))

	err := pipe.runAll(ctx, client.NewMock())
	require.ErrorIs(t, err, errAdditionalLocaleDuplicate)

	// When a later additional locale is invalid, no winget artifact (version,
	// installer, default locale, or any additional locale) may be registered.
	require.Empty(t, ctx.Artifacts.Filter(artifact.ByTypes(
		artifact.WingetInstaller,
		artifact.WingetVersion,
		artifact.WingetDefaultLocale,
		artifact.WingetLocale,
	)).List())
}

func TestErrNoArchivesFound(t *testing.T) {
	require.EqualError(t, errNoArchivesFound{
		goamd64: "v1",
		ids:     []string{"foo", "bar"},
	}, "no zip archives found matching goos=[windows] goarch=[amd64 386] goamd64=v1 ids=[foo bar]")
}

func TestMakeInstallerMultipleArchivesSameArch(t *testing.T) {
	for _, goarch := range []string{"amd64", "386", "arm64"} {
		t.Run(goarch, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dist:        folder,
				ProjectName: "foo",
			}, testctx.WithVersion("1.2.1"))

			var archives []*artifact.Artifact
			for _, id := range []string{"a", "b"} {
				path := filepath.Join(folder, "foo_"+id+".zip")
				require.NoError(t, os.WriteFile(path, []byte("fake"), 0o644))
				archives = append(archives, &artifact.Artifact{
					Name:   "foo_" + id + ".zip",
					Path:   path,
					Goos:   "windows",
					Goarch: goarch,
					Type:   artifact.UploadableArchive,
					Extra: map[string]any{
						artifact.ExtraID:       id,
						artifact.ExtraFormat:   "zip",
						artifact.ExtraBinaries: []string{"foo.exe"},
					},
				})
			}

			_, err := makeInstaller(ctx, config.Winget{}, archives)
			require.ErrorIs(t, err, errMultipleArchives)
		})
	}
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Winget:      []config.Winget{{}},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	winget := ctx.Config.Winget[0]
	require.Equal(t, "v1", winget.Goamd64)
	require.NotEmpty(t, winget.CommitMessageTemplate)
	require.Equal(t, "foo", winget.Name)
}

func TestFormatBinary(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Winget: []config.Winget{{
				Name:             "foo",
				Publisher:        "goreleaser",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			}},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
		testctx.WithSemver(1, 2, 1, "rc1"),
		testctx.WithDate(time.Date(2023, 6, 12, 20, 32, 10, 12, time.Local)))

	ctx.ReleaseNotes = "the changelog for this release..."
	createFakeArtifact := func(id, goos, goarch, goamd64 string) {
		path := filepath.Join(folder, "dist/foo_"+goos+goarch+goamd64+".exe")
		art := artifact.Artifact{
			Name:    "foo_" + goos + "_" + goarch + goamd64 + ".exe",
			Path:    path,
			Goos:    goos,
			Goarch:  goarch,
			Goamd64: goamd64,
			Type:    artifact.UploadableBinary,
			Extra: map[string]any{
				artifact.ExtraID:     id,
				artifact.ExtraBinary: "somebin",
			},
		}
		ctx.Artifacts.Add(&art)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	goos := "windows"
	createFakeArtifact("foo", goos, "amd64", "v1")
	createFakeArtifact("foo", goos, "386", "")
	createFakeArtifact("foo", goos, "arm64", "")

	client := client.NewMock()
	pipe := Pipe{}

	require.NoError(t, pipe.Default(ctx))
	require.NoError(t, pipe.runAll(ctx, client))
	for _, winget := range ctx.Artifacts.Filter(artifact.ByTypes(
		artifact.WingetInstaller,
		artifact.WingetVersion,
		artifact.WingetDefaultLocale,
		artifact.WingetLocale,
	)).List() {
		bts, err := os.ReadFile(winget.Path)
		require.NoError(t, err)
		locale := artifact.ExtraOr(*winget, wingetLocaleExtra, "")
		if locale == "" {
			cfg := artifact.MustExtra[config.Winget](*winget, wingetConfigExtra)
			locale = cfg.DefaultLocale
		}
		golden.RequireEqualExtSubfolder(t, bts, extFor(winget.Type, locale))
	}
	require.NoError(t, pipe.publishAll(ctx, client))
	require.True(t, client.CreatedFile)
}
