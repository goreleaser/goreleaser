# GoReleaser

<img src="https://avatars2.githubusercontent.com/u/24697112?v=3&s=200" alt="goreleaser" align="right" />

GoReleaser builds Go binaries for several platforms, creates a Github release and then
push a Homebrew formulae to a repository. All that wrapped in your favorite CI.

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md).
By participating, you are expected to uphold this code. Please report unacceptable behavior to root@carlosbecker.com.

[![Build Status](https://travis-ci.org/goreleaser/releaser.svg?branch=master)](https://travis-ci.org/goreleaser/releaser) [![Go Report Card](https://goreportcard.com/badge/github.com/goreleaser/releaser)](https://goreportcard.com/report/github.com/goreleaser/releaser) [![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

## How it works?

The idea started with a [simple shell script](https://github.com/goreleaser/old-go-releaser),
but it quickly became more complex and I also wanted to publish binaries via
Homebrew.

So, the all-new goreleaser was born.

## Usage

You may then run releaser at the root of your repository:

You need to export a `GITHUB_TOKEN` environment variable with
the `repo` scope selected. You can create one
[here](https://github.com/settings/tokens/new).

GoReleaser uses the latest [Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository,
so you need to [create a tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging#Annotated-Tags) first.

Now you can run `releaser` at the root of your repository:

```sh
curl -s https://raw.githubusercontent.com/goreleaser/get/master/latest | bash
```

This will build `main.go` as binary, for `Darwin` and `Linux`
(`amd64` and `i386`), archive the binary and common files as `.tar.gz`,
and finally, publish a new Github release in the repository with
archives uploaded.


For further customization create a `goreleaser.yml` file in the root of your repository.

### Homebrew

Add a `brew` section to push a formulae to a Homebrew tab repository:

```yaml
brew:
  repo: user/homebrew-tap
  folder: optional/subfolder/inside/the/repo
  caveats: "Optional caveats to add to the formulae"
```

See the [Homebrew docs](https://github.com/Homebrew/brew/blob/master/docs/How-to-Create-and-Maintain-a-Tap.md) for creating your own tap.

### Build customization

Just add a `build` section

```yaml
build:
  main: ./cmd/main.go
  ldflags: -s -w
  oses:
    - darwin
    - freebsd
  arches:
    - amd64
```

> - `oses` and `arches` should be in `GOOS`/`GOARCH`-compatible format.
> - `-s -w` is the default value for `ldflags`.

### Archive customization

You can customize the name and format of the archive adding an `archive`
section:

```yaml
archive:
  name_template: "{{.BinaryName}}_{{.Version}}_{{.Os}}_{{.Arch}}"
  format: zip
  replacements:
    amd64: 64-bit
    386: 32-bit
    darwin: macOS
    linux: Tux
```

> - Default `name_template` is `{{.BinaryName}}_{{.Os}}_{{.Arch}}`
> - Valid formats are `tar.gz` and `zip`, default is `tar.gz`
> - By default, `replacements` replace `GOOS` with `uname -s` values and
> `GOARCH` with `uname -m` values. They keys should always be in the `GOOS` and
> `GOARCH` form.

### Add more files

You might also want to change the files that are packaged by adding a `files`
section:

```yaml
files:
  - LICENSE.txt
  - README.md
  - CHANGELOG.md
```

> By default GoReleaser adds the binary itself, `LICENCE*`, `LICENSE*`,
`README*` and `CHANGELOG*`.

### ldflags (main.version)

GoReleaser already sets a `main.version` ldflag, so, in you `main.go` program,
you can:

```go
package main

var version = "master"

func main() {
  println(version)
}
```

And this version will always be the tag name.


### Other customizations

- By default it's assumed that the repository to release to is the same as the Git `remote origin`.
  If this is not the case for your project, you can specify a `repo`:

```yaml
repo: owner/custom-repo
```

- By default the binary name is the name of the project directory.
  You can specify a different `binary_name`:

```yaml
binary_name: my-binary
```


## Wire it with travis-ci

You may want to wire this to auto-deploy your new tags on travis, for example:

```yaml
after_success:
  test -n "$TRAVIS_TAG" && curl -s https://raw.githubusercontent.com/goreleaser/get/master/latest | bash
```

## What the end result looks like

The release on Github looks pretty much like this:

[![image](https://cloud.githubusercontent.com/assets/245435/21578845/09404c8a-cf78-11e6-92d7-165ddc03ca6c.png)
](https://github.com/goreleaser/releaser/releases)

And the [Homebrew formulae](https://github.com/goreleaser/homebrew-tap/blob/master/release.rb) would look like:

```rb
class Release < Formula
  desc "Deliver Go binaries as fast and easily as possible"
  homepage "https://github.com/goreleaser/releaser"
  url "https://github.com/goreleaser/releaser/releases/download/v0.2.8/release_Darwin_x86_64.tar.gz"
  version "v0.2.8"
  sha256 "9ee30fc358fae8d248a2d7538957089885da321dca3f09e3296fe2058e7fff74"

  def install
    bin.install "release"
  end
end
```

## How to contribute

Please refer to our [contributing guidelines](/CONTRIBUTING.md).

## Badges

Feel free to use it in your own projects:

```md
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)
```
