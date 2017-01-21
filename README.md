# GoReleaser

<img src="https://avatars2.githubusercontent.com/u/24697112?v=3&s=200" alt="goreleaser" align="right" />

GoReleaser builds Go binaries for several platforms, creates a GitHub release and then
pushes a Homebrew formulae to a repository. All that wrapped in your favorite CI.

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md).
By participating, you are expected to uphold this code. Please report unacceptable behavior to root@carlosbecker.com.

[![Release](https://img.shields.io/github/release/goreleaser/goreleaser.svg?style=flat-square)](https://github.com/goreleaser/goreleaser/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE.md)
[![Travis](https://img.shields.io/travis/goreleaser/goreleaser.svg?style=flat-square)](https://travis-ci.org/goreleaser/goreleaser)
[![Go Report Card](https://goreportcard.com/badge/github.com/goreleaser/goreleaser?style=flat-square)](https://goreportcard.com/report/github.com/goreleaser/goreleaser)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

## Why?

The idea started with a [simple shell script](https://github.com/goreleaser/old-go-releaser),
but it quickly became more complex and I also wanted to publish binaries via
Homebrew.

So, the all-new GoReleaser was born.

## Documentation

For Documentation, visit the [GoReleaser website](https://goreleaser.github.io/documentation/) or our GitHub [docs](/docs).

## Usage

- You need to export a `GITHUB_TOKEN` environment variable with
the `repo` scope selected. You can create one [here](https://github.com/settings/tokens/new).

- GoReleaser uses the latest [Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository,
so you need to [create a tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging#Annotated-Tags) first.

- Now you can run `goreleaser` at the root of your repository:

```sh
curl -s https://raw.githubusercontent.com/goreleaser/get/master/latest | bash
```

This will build `main.go` as binary, for `Darwin` and `Linux`
(`amd64` and `i386`), archive the binary and common files as `.tar.gz`,
and finally, publish a new GitHub release in the repository with
archives uploaded.

Of course, all this can be customized!

## Customization

For customization create a `goreleaser.yml` file in the root of your repository.

A complete and commented example can be found in the [documentation](/docs/#release-customization).

You can also check the [goreleaser.yml](/goreleaser.yml) used by GoReleaser
itself.

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

And this version will always be the name of the current tag.

## Wire it with Travis CI

You may want to wire this to auto-deploy your new tags on Travis, for example:

```yaml
after_success:
  test -n "$TRAVIS_TAG" && curl -s https://raw.githubusercontent.com/goreleaser/get/master/latest | bash
```

## What the end result looks like

The release on GitHub looks pretty much like this:

[![image](https://cloud.githubusercontent.com/assets/245435/22177948/e1d77494-e010-11e6-8dc9-c1d3a6eab40e.png)
](https://github.com/goreleaser/goreleaser/releases)

And the [Homebrew formulae](https://github.com/goreleaser/homebrew-tap/blob/master/Formula/goreleaser.rb) would look like:

```rb
class Goreleaser < Formula
  desc "Deliver Go binaries as fast and easily as possible"
  homepage "https://goreleaser.github.io/"
  url "https://github.com/goreleaser/goreleaser/releases/download/v0.5.8/goreleaser_Darwin_x86_64.tar.gz"
  version "v0.5.8"
  sha256 "c37784679840fb9e89c445f6176442eb3624d18cf232ff5a1dfe57e905c83d77"

  depends_on "git"

  def install
    bin.install "goreleaser"
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
