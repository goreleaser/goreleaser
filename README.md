<p align="center">
  <img alt="GoReleaser Logo" src="https://avatars2.githubusercontent.com/u/24697112?v=3&s=200" height="140" />
  <h3 align="center">GoReleaser</h3>
  <p align="center">Deliver Go binaries as fast and easily as possible.</p>
  <p align="center">
    <a href="https://github.com/goreleaser/goreleaser/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/goreleaser/goreleaser.svg?style=flat-square"></a>
    <a href="/LICENSE.md"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
    <a href="https://travis-ci.org/goreleaser/goreleaser"><img alt="Travis" src="https://img.shields.io/travis/goreleaser/goreleaser.svg?style=flat-square"></a>
    <a href="https://codecov.io/gh/goreleaser/goreleaser"><img alt="Codecov branch" src="https://img.shields.io/codecov/c/github/goreleaser/goreleaser/master.svg?style=flat-square"></a>
    <a href="https://goreportcard.com/report/github.com/goreleaser/goreleaser"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/goreleaser/goreleaser?style=flat-square"></a>
    <a href="http://godoc.org/github.com/goreleaser/goreleaser"><img alt="Go Doc" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square"></a>
    <a href="https://saythanks.io/to/caarlos0"><img alt="SayThanks.io" src="https://img.shields.io/badge/SayThanks.io-%E2%98%BC-1EAEDB.svg?style=flat-square"></a>
    <a href="https://github.com/goreleaser"><img alt="Powered By: GoReleaser" src="https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square"></a>
  </p>
</p>

---


GoReleaser builds Go binaries for several platforms, creates a GitHub release and then
pushes a Homebrew formula to a tap repository. All that wrapped in your favorite CI.

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.
We appreciate your contribution. Please refer to our [contributing guidelines](CONTRIBUTING.md) for further information.

For questions join the [#goreleaser](https://gophers.slack.com/messages/goreleaser/) channel in the [Gophers Slack](https://invite.slack.golangbridge.org/).

# Table of contents

- [Introduction](#introduction)
- [Quick start](#quick-start)
- [Environment setup](#environment-setup)
- [Release customization](#release-customization)
- [Integration with CI](#integration-with-ci)

##  Introduction

GoReleaser is a release automation tool for Golang projects, the goal is to simplify the build, release and publish steps while providing variant customization options for all steps.

GoReleaser is built for CI tools; you only need to [download and execute it](#integration-with-ci) in your build script.
You can [customize](#release-customization) your release process by createing a `.goreleaser.yml` file.
We are also working on integrating with package managers, we currently support Home
.

The idea started with a [simple shell script](https://github.com/goreleaser/old-go-releaser), but it quickly became more complex and I also wanted to publish binaries via Homebrew taps.

##  Quick start

In this example we will build, archive and release a Golang project.
Create a GitHub repository and add a single main package:
```go
// main.go
package main

func main() {
  println("Ba dum, tss!")
}
```

By default GoReleaser will build the current directory, but you can change the build package path in the GoReleaser configuration file.

```yml
# .goreleaser.yml
# Build customization
builds:
  - binary: drum-roll
    goos:
      - windows
      - darwin
      - linux
    goarch:
      - amd64
```

PS: Invalid GOOS/GOARCH combinations will automatically be skipped.

This configuration specifies the build operating systems to Windows, Linux and MacOS using 64bit architecture, the name of the binaries is `drum-roll`.

GoReleaser will then archive the result binaries of each Os/Arch into a separate file. The default format is `{{.ProjectName}}_{{.Os}}_{{.Arch}}`.
You can change the archives name and format. You can also replace the OS and the Architecture with your own.
Another useful feature is to add files to archives, this is very useful for integrating assets like resource files.

```yml
# .goreleaser.yml
# Build customization
builds:
  - main: main.go
    binary: drum-roll
    goos:
      - windows
      - darwin
      - linux
    goarch:
      - amd64
# Archive customization
archive:
  format: tar.gz
  replacements:
    amd64: 64-bit
    darwin: macOS
    linux: Tux
  files:
    - drum-roll.licence.txt
```

This configuration will generate tar archives, contains an additional file `drum-roll.licence.txt`, the archives will be located in:

- `./dist/drum-roll_windows_64-bit.tar.gz`
- `./dist/drum-roll_macOS_64-bit.tar.gz`
- `./dist/drum-roll_Tux_64-bit.tar.gz`

Next export a `GITHUB_TOKEN` environment variable with the `repo` scope selected. This will be used to deploy releases to your GitHub repository. Create yours [here](https://github.com/settings/tokens/new).

```console
$ export GITHUB_TOKEN=`YOUR_TOKEN`
```

GoReleaser uses the latest [Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository.
Create a tag and push it to GitHub:

```console
$ git tag -a v0.1.0 -m "First release" && git push origin v0.1.0
```

**Note**: we recommend the use of [semantic versioning](http://semver.org/). We
are not enforcing it though. We do remove the `v` prefix and then enforce
that the next character is a number. So, `v0.1.0` and `0.1.0` are virtually the
same and are both accepted, while `version0.1.0` is not.

If you don't want to create a tag yet but instead simply create a package based on the latest commit, then you can also use the `--snapshot` flag.

Now you can run GoReleaser at the root of your repository:

```console
$ goreleaser
```

That's it! Check your GitHub project's release page.
The release should look like this:

[![image](https://cloud.githubusercontent.com/assets/245435/23342061/fbcbd506-fc31-11e6-9d2b-4c1b776dee9c.png)
](https://github.com/goreleaser/goreleaser/releases)

## Environment setup

### GitHub Token

GoReleaser requires a GitHub API token with the `repo` scope checked to deploy the artefacts to GitHub. You can create one [here](https://github.com/settings/tokens/new).
This token should be added to the environment variables as `GITHUB_TOKEN`. Here is how to do it with Travis CI: [Defining Variables in Repository Settings](https://docs.travis-ci.com/user/environment-variables/#Defining-Variables-in-Repository-Settings).

### A note about `main.version`

GoReleaser always sets a `main.version` ldflag. You can use it in your
`main.go` file:

```go
package main

var version = "master"

func main() {
  println(version)
}
```

`version` will be the current Git tag (with `v` prefix stripped) or the name of the snapshot if you're using the `--snapshot` flag.

## GoReleaser customization

GoReleaser provides multiple customizations via the `.goreleaser.yml` file.
You can generate it by running `goreleaser init` or start from scratch. The
defaults are sensible and fit for most projects.

We'll cover all customizations available bellow:

### Project name

```yml
# .goreleaser.yml
# The name of the project. It is used in the name of the brew formula, archives,
# etc. Defaults to the name of the git project.
project_name: myproject
```

### Build customization

```yml
# .goreleaser.yml
builds:
  # You can have multiple builds, its a common yaml list
  -
    # Path to main.go file or main package.
    # Default is `.`
    main: ./cmd/main.go

    # Name of the binary.
    # Default is the name of the project directory.
    binary: program

    # Custom build tags.
    # Default is empty
    flags: -tags dev

    # Custom ldflags template.
    # This is parsed with Golang template engine and the following variables
    # are available:
    # - Date
    # - Commit
    # - Tag
    # - Version (Tag with the `v` prefix stripped)
    # The default is `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}`
    # Date format is `2006-01-02_15:04:05`
    ldflags: -s -w -X main.build={{.Version}}

    # Custom environment variables to be set durign the builds.
    # Default is empty
    env:
    - CGO_ENABLED=0

    # GOOS list to build in.
    # For more info refer to https://golang.org/doc/install/source#environment
    # Defaults are darwin and linux
    goos:
      - freebsd
      - windows

    # GOARCH to build in.
    # For more info refer to https://golang.org/doc/install/source#environment
    # Defaults are 386 and amd64
    goarch:
      - amd64
      - arm
      - arm64

    # GOARM to build in when GOARCH is arm.
    # For more info refer to https://golang.org/doc/install/source#environment
    # Defaults are 6
    goarm:
      - 6
      - 7

    # List of combinations of GOOS + GOARCH + GOARM to ignore.
    # Default is empty.
    ignore:
      - goos: darwin
        goarch: 386
      - goos: linux
        goarch: arm
        goarm: 7

    # Hooks can be used to customize the final binary, for example, to run
    # generator or whatever you want.
    # Default is both hooks empty.
    hooks:
      pre: rice embed-go
      post: ./script.sh
```

### Archive customization

```yml
# .goreleaser.yml
archive:
  # You can change the name of the archive.
  # This is parsed with Golang template engine and the following variables
  # are available:
  # - ProjectName
  # - Tag
  # - Version (Tag with the `v` prefix stripped)
  # - Os
  # - Arch
  # - Arm (ARM version)
  # The default is `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}`
  name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

  # Archive format. Valid options are `tar.gz`, `zip` and `binary`.
  # If format is `binary` no archives are created and the binaries are instead uploaded directly.
  # In that case name_template the below specified files are ignored.
  # Default is `tar.gz`
  format: zip

  # Can be used to archive on different formats for specific GOOSs.
  # Most common use case is to archive as zip on Windows.
  # Default is empty
  format_overrides:
    - goos: windows
      format: zip

  # Replacements for GOOS and GOARCH on the archive name.
  # The keys should be valid GOOS or GOARCH values followed by your custom
  # replacements.
  replacements:
    amd64: 64-bit
    386: 32-bit
    darwin: macOS
    linux: Tux

  # Additional files/globs you want to add to the archive.
  # Defaults are any files matching `LICENCE*`, `LICENSE*`,
  # `README*` and `CHANGELOG*` (case-insensitive)
  files:
    - LICENSE.txt
    - README.md
    - CHANGELOG.md
    - docs/*
    - design/*.png
```

### Release customization

```yml
# .goreleaser.yml
release:
  # Repo in which the release will be created.
  # Default is extracted from the origin remote URL.
  github:
    owner: user
    name: repo

  # If set to true, will not auto-publish the release.
  # Default is false
  draft: true
```

You can also specify a release notes file in markdown format using the
`--release-notes` flag.

### Snapshot customization

```yml
# .goreleaser.yml
snapshot:
  # Allows you to change the name of the generated snapshot
  # releases. The following variables are available:
  # - Commit
  # - Tag
  # - Timestamp
  # Default: SNAPSHOT-{{.Commit}}
  name_template: SNAPSHOT-{{.Commit}}
```

### Checksums file customization

```yml
# .goreleaser.yml
checksum:
  # You can change the name of the checksums file.
  # This is parsed with Golang template engine and the following variables
  # are available:
  # - ProjectName
  # - Tag
  # - Version (Tag with the `v` prefix stripped)
  # The default is `{{ .ProjectName }}_{{ .Version }}_checksums.txt`
  name_template: "{{ .ProjectName }}_checksums.txt"
```

### Homebrew tap customization

The brew section specifies how the formula should be created.
Check [the Homebrew documentation](https://github.com/Homebrew/brew/blob/master/docs/How-to-Create-and-Maintain-a-Tap.md) and the [formula cookbook](https://github.com/Homebrew/brew/blob/master/docs/Formula-Cookbook.md) for details.

```yml
# .goreleaser.yml
brew:
  # Reporitory to push the tap to.
  github:
    owner: user
    name: homebrew-tap

  # Folder inside the repository to put the formula.
  # Default is the root folder.
  folder: Formula

  # Caveats for the user of your binary.
  # Default is empty.
  caveats: "How to use this binary"

  # Your app's homepage
  # Default is empty
  homepage: "https://example.com/"

  # Your app's description
  # Default is empty
  description: "Software to create fast and easy drum rolls."

  # Dependencies of your package
  dependencies:
    - git
    - zsh

  # Packages that conflict with your package
  conflicts:
    - svn
    - bash

  # Packages that run as a service. Default is empty.
  plist: |
    <?xml version="1.0" encoding="UTF-8"?>
    ...

  # So you can brew test your formula. Default is empty.
  test: |
    system "#{bin}/program --version"
    ...

  # Custom install script for brew. Default is 'bin.install "program"'
  install: |
    bin.install "program"
    ...
```

By defining the `brew` section, GoReleaser will take care of publishing the Homebrew tap.
Assuming that the current tag is `v1.2.3`, the above config will generate a `program.rb` formula in the `Formula` folder of `user/homebrew-tap` repository:

```rb
class Program < Formula
  desc "How to use this binary"
  homepage "https://github.com/user/repo"
  url "https://github.com/user/repo/releases/download/v1.2.3/program_v1.2.3_macOs_64bit.zip"
  version "v1.2.3"
  sha256 "9ee30fc358fae8d248a2d7538957089885da321dca3f09e3296fe2058e7fff74"

  depends_on "git"
  depends_on "zsh"

  def install
    bin.install "program"
  end
end
```

Note that GoReleaser does not yet generate a valid homebrew-core formula. The generated formulas
are meant to be published as [homebrew taps](https://docs.brew.sh/brew-tap.html), not in any
of the official homebrew repositories.

### FPM build customization

GoReleaser can be wired to [fpm]() to generate `.deb`, `.rpm` and other archives. Check its
[wiki](https://github.com/jordansissel/fpm/wiki) for more info.

[fpm]: https://github.com/jordansissel/fpm

```yml
# .goreleaser.yml
fpm:
  # Your app's vendor
  # Default is empty
  vendor: Drum Roll Inc.
  # Your app's homepage
  # Default is empty
  homepage: https://example.com/

  # Your app's maintainer (probably you)
  # Default is empty
  maintainer: Drummer <drum-roll@example.com>

  # Your app's description
  # Default is empty
  description: Software to create fast and easy drum rolls.

  # Your app's license
  # Default is empty
  license: Apache 2.0

  # Formats to generate as output
  formats:
    - deb
    - rpm

  # Dependencies of your package
  dependencies:
    - git
    - zsh

  # Packages that conflict with your package
  conflicts:
    - svn
    - bash

  # Files or directories to add to your package (beyond the binary)
  files:
    "scripts/etc/init.d/": "/etc/init.d"

```

Note that GoReleaser will not install `fpm` nor any of its dependencies for you.

### Snapcraft build customization

GoReleaser can generate `snap` packages. [Snaps](http://snapcraft.io/) are a new packaging format that will let you publish your project directly to the Ubuntu store. From there it will be installable in all the [supported Linux distros](https://snapcraft.io/docs/core/install), with automatic and transactional updates.

You can read more about it in the [snapcraft docs](https://snapcraft.io/docs/).

```yml
# .goreleaser.yml
snapcraft:

  # The name of the snap. This is optional and defaults to the project name.
  name: drumroll

  # Single-line elevator pitch for your amazing snap.
  # 79 char long at most.
  summary: Software to create fast and easy drum rolls.

  # This the description of your snap. You have a paragraph or two to tell the
  # most important story about your snap. Keep it under 100 words though,
  # we live in tweetspace and your description wants to look good in the snap
  # store.
  description: |
    This is the best drum roll application out there.
    Install it and awe!

  # A guardrail to prevent you from releasing a snap to all your users before
  # it is ready.
  # `devel` will let you release only to the `edge` and `beta` channels in the
  # store. `stable` will let you release also to the `candidate` and `stable`
  # channels. More info about channels here:
  # https://snapcraft.io/docs/reference/channels.
  grade: stable

  # Snaps can be setup to follow three different confinement policies:
  # `strict`, `devmode` and `classic`. A strict confinement where the snap
  # can only read and write in its own namespace is recommended. Extra
  # permissions for strict snaps can be declared as `plugs` for the app, which
  # are explained later. More info about confinement here:
  # https://snapcraft.io/docs/reference/confinement).
  confinement: strict

  # Each binary built by GoReleaser is an app inside the snap. In this section
  # you can declare extra details for those binaries. It is optional.
  apps:

    # The name of the app must be the same name of the binary built.
    drumroll:

      # If your app requires extra permissions to work outside of its default
      # confined space, delcare them here.
      # You can read the documentation about the available plugs and the
      # things they allow:
      # https://snapcraft.io/docs/reference/interfaces).
      plugs: ["home", "network"]

      # If you want your app to be autostarted and to always run in the
      # background, you can make it a simple daemon.
      daemon: simple
```

Note that GoReleaser will not install `snapcraft` nor any of its dependencies for you.

### Custom release notes

You can have a markdown file previously created with the release notes, and
pass it down to goreleaser with the `--release-notes=FILE` flag.

## Integration with CI

You may want to wire this to auto-deploy your new tags on [Travis](https://travis-ci.org), for example:

```yaml
# .travis.yml
after_success:
  - test -n "$TRAVIS_TAG" && curl -sL https://git.io/goreleaser | bash
```

Here is how to do it with [CircleCI](https://circleci.com):

```yml
# circle.yml
deployment:
  tag:
    tag: /v[0-9]+(\.[0-9]+)*(-.*)*/
    owner: user
    commands:
      - curl -sL https://git.io/goreleaser | bash
```

*Note that if you test multiple versions or multiple OSes you probably want to make sure GoReleaser is just run once*

### Stargazers over time

[![goreleaser/goreleaser stargazers over time](https://starcharts.herokuapp.com/goreleaser/goreleaser.svg)](https://starcharts.herokuapp.com/goreleaser/goreleaser)


---

Would you like to fix something in the documentation? Feel free to open an [issue](https://github.com/goreleaser/goreleaser/issues).
