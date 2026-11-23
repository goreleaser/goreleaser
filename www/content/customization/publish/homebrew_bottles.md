---
title: "Homebrew Bottles"
linkTitle: Homebrew Bottles
weight: 91
---

GoReleaser can publish a regular _Homebrew Formula_ that works the way
`homebrew-core` formulas do:

- the formula builds from the release's **source archive**, so
  `brew install --build-from-source` works;
- it also carries a `bottle do` block pointing at **bottles** GoReleaser built
  from your release binaries, so a normal `brew install` pours a prebuilt
  binary and never compiles anything.

Use this for command-line tools. Casks are meant for applications; a formula
with bottles is what Homebrew itself ships for CLIs.

The bottles are regular release artifacts: they are uploaded with the release,
listed in the checksums file, and signed if you sign artifacts.

## Requirements

The source archive must be enabled, since the formula's `url` points at it:

```yaml {filename=".goreleaser.yaml"}
source:
  enabled: true
```

## Bottle tags

| GOOS/GOARCH  | Bottle tag             |
| ------------ | ---------------------- |
| darwin/arm64 | `arm64_<bottles.macos>` |
| darwin/amd64 | `<bottles.macos>`      |
| darwin/all   | both of the above      |
| linux/arm64  | `arm64_linux`          |
| linux/amd64  | `x86_64_linux`         |

Homebrew pours a macOS bottle on its tagged release and every newer one, so
`bottles.macos` should be the oldest macOS your binaries support. It defaults
to `monterey`, the oldest release current Go toolchains support.

Bottles are named `<name>-<version>.<tag>.bottle.tar.gz`, which is the name
Homebrew downloads from the formula's `root_url`.

## Building from source

By default the formula's `install` block rebuilds each bottled binary with
`go build` and Homebrew's `std_go_args`, reusing the build's `main`, `dir`,
rendered `ldflags` and `CGO_ENABLED` setting, and `go` is added as a build
dependency. Set `install` to replace that, for example for non-Go builders.

Bottles contain only what GoReleaser packs into them. Files you add with
`bottles.files` (completions, manpages) are not in the source archive unless
they are committed, so add the equivalent to `extra_install` for source
builds, e.g. `generate_completions_from_executable(bin/"foo", "completion")`.

## Configuration

```yaml {filename=".goreleaser.yaml"}
homebrew_bottles:
  - # Formula name. Templates: allowed.
    # Default: the project name.
    name: myproject

    # Tap repository; same options as the other Homebrew publishers,
    # including git URLs and pull requests.
    repository:
      owner: user
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"

    # Default: Formula.
    directory: Formula

    # Default: "Brew formula update for {{ .ProjectName }} version {{ .Tag }}".
    commit_msg_template: "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # Build IDs whose binaries go into the bottles.
    # Default: all darwin and linux binaries.
    ids: [myproject]

    # Default: v1.
    goamd64: v1

    # URL template for downloads; must end in the artifact name.
    # Default: the release download URL.
    url_template: "https://github.com/user/repo/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    description: "Software to create fast and easy drum rolls."
    homepage: "https://example.com/"
    license: "MIT"

    bottles:
      # Oldest macOS codename the bottles are tagged for.
      # Default: monterey.
      macos: monterey

      # Extra files to pack into each bottle, relative to the keg.
      files:
        - src: completions/myproject.zsh
          dst: share/zsh/site-functions/_myproject

    # Replaces the default `go build` install block (source builds only).
    install: |
      system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/myproject"

    # Appended to the install block (source builds only).
    extra_install: |
      generate_completions_from_executable(bin/"myproject", "completion")

    dependencies:
      - name: git
    conflicts: [svn]
    caveats: "How to use this binary"
    post_install: |
      etc.install "app-config.conf"
    service: |
      run [opt_bin/"myproject", "serve"]
    test: |
      system "#{bin}/myproject", "--version"
    custom_block: |
      head "https://github.com/user/repo.git"

    # "true" never publishes; "auto" skips prereleases.
    skip_upload: auto
```

{{% g_include file="includes/prs.md" %}}
