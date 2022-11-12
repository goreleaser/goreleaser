package chocolatey

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/stretchr/testify/require"
)

func TestNuspecBytes(t *testing.T) {
	m := &Nuspec{
		Xmlns: schema,
		Metadata: Metadata{
			ID:                       "goreleaser",
			Version:                  "1.12.3",
			PackageSourceURL:         "https://github.com/goreleaser/goreleaser",
			Owners:                   "caarlos0",
			Title:                    "GoReleaser",
			Authors:                  "caarlos0",
			ProjectURL:               "https://goreleaser.com/",
			IconURL:                  "https://raw.githubusercontent.com/goreleaser/goreleaser/main/www/docs/static/avatar.png",
			Copyright:                "2016-2022 Carlos Alexandro Becker",
			LicenseURL:               "https://github.com/goreleaser/goreleaser/blob/main/LICENSE.md",
			RequireLicenseAcceptance: true,
			ProjectSourceURL:         "https://github.com/goreleaser/goreleaser",
			DocsURL:                  "https://github.com/goreleaser/goreleaser/blob/main/README.md",
			BugTrackerURL:            "https://github.com/goreleaser/goreleaser/issues",
			Tags:                     "go docker homebrew golang package",
			Summary:                  "Deliver Go binaries as fast and easily as possible",
			Description:              "GoReleaser builds Go binaries for several platforms, creates a GitHub release and then pushes a Homebrew formula to a tap repository. All that wrapped in your favorite CI.",
			ReleaseNotes:             "This tag is only to keep version parity with the pro version, which does have a couple of bugfixes.",
			Dependencies: &Dependencies{Dependency: []Dependency{
				{ID: "nfpm", Version: "2.20.0"},
			}},
		},
		Files: Files{File: []File{
			{Source: "tools\\**", Target: "tools"},
		}},
	}

	out, err := m.Bytes()
	require.NoError(t, err)

	golden.RequireEqualExt(t, out, ".nuspec")
}
