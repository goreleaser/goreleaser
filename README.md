# GoReleaser

<img src="https://avatars2.githubusercontent.com/u/24697112?v=3&s=200" alt="goreleaser" align="right" />

GoReleaser builds Go binaries for several platforms, creates a GitHub release and then
pushes a Homebrew formula to a repository. All that wrapped in your favorite CI.

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.
We appreciate your contribution. Please refer to our [contributing guidelines](CONTRIBUTING.md).

[![Release](https://img.shields.io/github/release/goreleaser/goreleaser.svg?style=flat-square)](https://github.com/goreleaser/goreleaser/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE.md)
[![Travis](https://img.shields.io/travis/goreleaser/goreleaser.svg?style=flat-square)](https://travis-ci.org/goreleaser/goreleaser)
[![Coverage Status](https://img.shields.io/coveralls/goreleaser/goreleaser/master.svg?style=flat-square)](https://coveralls.io/github/goreleaser/goreleaser?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/goreleaser/goreleaser?style=flat-square)](https://goreportcard.com/report/github.com/goreleaser/goreleaser)
[![SayThanks.io](https://img.shields.io/badge/SayThanks.io-%E2%98%BC-1EAEDB.svg?style=flat-square)](https://saythanks.io/to/caarlos0)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

For questions join the [#goreleaser](https://gophers.slack.com/messages/goreleaser/) channel in the [Gophers Slack](https://invite.slack.golangbridge.org/).

# Table of contents

- [Introduction](#intorduction)
- [Quick start](#quick-start)
- [Environment setup](#environment-setup)
- [Release customization](#release-customization)
- [Integration with CI](#integration-with-ci)

##  Introduction

GoReleaser is a release automation tool for Golang projects, the goal is to simplify the build, release and publish steps while providing variant customization options for all steps.

GoReleaser is built for CI tools; you only need to [download and execute it](#integration-with-ci) in your build script.
You can [customize](#release-customization) your release process by createing a `goreleaser.yml` file.
We are also working on integrating with package managers, we currently support Homebrew.

The idea started with a [simple shell script](https://github.com/goreleaser/old-go-releaser), but it quickly became more complex and I also wanted to publish binaries via Homebrew.

_So, the all-new GoReleaser was born._

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

By default GoReleaser will build the **main.go** file located in your current directory, but you can change the build package path in the GoReleaser configuration file.

```yml
# goreleaser.yml
# Build customization
build:
  binary: drum-roll
  goos:
    - windows
    - darwin
    - linux
  goarch:
    - amd64
```

This configuration specifies the build operating systems to Windows, Linux and MacOS using 64bit architecture, the name of the binaries is `drum-roll`.

GoReleaser will then archive the result binaries of each Os/Arch into a separate file. The default format is `{{.Binary}}_{{.Os}}_{{.Arch}}`.
You can change the archives name and format. You can also replace the OS and the Architecture with your own.
Another useful feature is to add files to archives, this is very useful for integrating assets like resource files.

```yml
# goreleaser.yml
# Build customization
build:
  main: main.go
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
Create a tag:

```console
$ git tag -a v0.1.0 -m "First release"
```

**Note**: we recommend the use of [semantic versioning](http://semver.org/). We
are not enforcing it though. We do remove the `v` prefix and then enforce
that the next character is a number. So, `v0.1.0` and `0.1.0` are virtually the
same and are both accepted, while `version0.1.0` is not.

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

`version` will always be the name of the current Git tag.

## Release customization

GoReleaser provides multiple customizations. We will cover them with the help of `goreleaser.yml`:

### Build customization

```yml
# goreleaser.yml
build:
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
  # - Version
  # - Date
  # - Commit
  # The default is `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}`
  # Date format is `2006-01-02_15:04:05`
  ldflags_template: -s -w -X main.build={{.Version}}

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

  # Hooks can be used to customize the final binary, for example, to run
  # generator or whatever you want.
  # Default is both hooks empty.
  hooks:
    pre: rice embed-go
    post: ./script.sh
```

### Archive customization

```yml
# goreleaser.yml
archive:
  # You can change the name of the archive.
  # This is parsed with Golang template engine and the following variables
  # are available:
  # - Binary
  # - Version
  # - Os
  # - Arch
  # The default is `{{.Binary}}_{{.Os}}_{{.Arch}}`
  name_template: "{{.Binary}}_{{.Version}}_{{.Os}}_{{.Arch}}"

  # Archive format. Valid options are `tar.gz` and `zip`.
  # Default is `tar.gz`
  format: zip

  # Replacements for GOOS and GOARCH on the archive name.
  # The keys should be valid GOOS or GOARCH values followed by your custom
  # replacements.
  # By default, `replacements` replace GOOS and GOARCH values with valid outputs
  # of `uname -s` and `uname -m` respectively.
  replacements:
    amd64: 64-bit
    386: 32-bit
    darwin: macOS
    linux: Tux

  # Additional files you want to add to the archive.
  # Defaults are any files matching `LICENCE*`, `LICENSE*`,
  # `README*` and `CHANGELOG*` (case-insensitive)
  files:
    - LICENSE.txt
    - README.md
    - CHANGELOG.md
```

### Release customization

```yml
# goreleaser.yml
release:
  # Repo in which the release will be created.
  # Default is extracted from the origin remote URL.
  github:
    owner: user
    name: repo
```

### Homebrew tap customization

The brew section specifies how the formula should be created.
Check [the Homebrew documentation](https://github.com/Homebrew/brew/blob/master/docs/How-to-Create-and-Maintain-a-Tap.md) and the [formula cookbook](https://github.com/Homebrew/brew/blob/master/docs/Formula-Cookbook.md) for details.

```yml
# goreleaser.yml
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

  # Dependencies of your package
  dependencies:
    - git
    - zsh

  # Packages that conflict with your package
  conflicts:
    - svn
    - bash

  # Packages that run as a service
  plist:|
    <?xml version="1.0" encoding="UTF-8"?>
    ...

  # Custom install script for brew. Default: "bin.install "program"
  install:|
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

### FPM build customization

GoReleaser can be wired to [fpm]() to generate `.deb`, `.rpm` and other archives. Check it's
[wiki](https://github.com/jordansissel/fpm/wiki) for more info.

[fpm]: https://github.com/jordansissel/fpm

```yml
# goreleaser.yml
fpm:
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
```

Note that GoReleaser will not install `fpm` nor any of it's dependencies for you.

## Integration with CI

You may want to wire this to auto-deploy your new tags on [Travis](https://travis-ci.org), for example:

```yaml
# .travis.yml
after_success:
  test -n "$TRAVIS_TAG" && curl -sL https://git.io/goreleaser | bash
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

---

Would you like to fix something in the documentation? Feel free to open an [issue](https://github.com/goreleaser/goreleaser/issues).
