GoReleaser builds Go binaries for several platforms, creates a GitHub release and then
pushes a Homebrew formulae to a repository. All that wrapped in your favorite CI.

[![Release](https://img.shields.io/github/release/goreleaser/goreleaser.svg?style=flat-square)](https://github.com/goreleaser/goreleaser/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE.md)
[![Travis](https://img.shields.io/travis/goreleaser/goreleaser.svg?style=flat-square)](https://travis-ci.org/goreleaser/goreleaser)
[![Go Report Card](https://goreportcard.com/badge/github.com/goreleaser/goreleaser?style=flat-square)](https://goreportcard.com/report/github.com/goreleaser/goreleaser)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

This project adheres to the Contributor Covenant [code of conduct](https://github.com/goreleaser/goreleaser/blob/master/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.   
We appreciate your contribution. Please refer to our [contributing guidelines](/CONTRIBUTING.md).

# Documentation

- [Introduction](#intorduction)
- [Quick start](#quick-start)
- [Environment setup](#environment-setup)
- [Release customization](#release-customization)
- [Integration with CI](#integration-with-ci)

##  Introduction

GoReleaser is a release automation tool for Golang projects, the goal is to simplify the build, release and publish steps while providing variant customization options for all steps.  

GoReleaser is built for CI tools, you only need a [single command line](#integration-with-ci) in your build script. Therefore, no package is required.  
You can define your customization in a `goreleaser.yml` file. For examples, check the [goreleaser.example.yml](https://github.com/goreleaser/goreleaser/blob/master/goreleaser.example.yml) or the [goreleaser.yml](https://github.com/goreleaser/goreleaser/blob/master/goreleaser.yml) used by GoReleaser itself. More on this in [Release customization](#release-customization).
We are also working on integrating package managers, we currently support Homebrew.

The idea started with a simple shell script ([old GoReleaser](https://github.com/goreleaser/old-go-releaser)), but it quickly became more complex and I also wanted to publish binaries via Homebrew.

_So, the all-new GoReleaser was born._

##  Quick start

In this example, we will build, archive and release a Golang project.  
For simplicity, create a GitHub repository and add a single main package:
```go
// main.go
package main

func main() {
  println("Ba dum, tss!")
}
```
By default, GoReleaser will build the main.go file located in your current directory, but you can change the build package path in a GoReleaser configuration file.

Speaking of GoReleaser configuration, let's create _goreleaser.yml_ file:

```yml
# goreleaser.yml
# Build customization
build:
  binary_name: drum-roll
  goos:
    - windows
    - darwin
    - linux
  goarch:
    - amd64
```

This configuration specifies the build operating systems to Windows, Linux and MacOS using 64bit architecture, the name of the binaries is "drum-roll".  

GoReleaser will then archive the result binaries of each Os/Arch into a separate file. The default syntax is `{{.BinaryName}}_{{.Os}}_{{.Arch}}`.  
You can change the archives name, and format, you can also replace the OS and the Architecture names with your own.  
Another useful feature is to add files to archives, this is very useful for integrating assets like resource files.

```yml
# goreleaser.yml
# Build customization
build:
  main: main.go
  binary_name: drum-roll
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
    - /path/drum-roll.licence.txt
```

This configuration will generate tar archives, contains an additional file "drum-roll.licence.txt", the archives will be  located in:  
  "./dist/drum-roll_windows_64-bit.tar.gz"  
  "./dist/drum-roll_macOS_64-bit.tar.gz"  
  "./dist/drum-roll_Tux_64-bit.tar.gz"  

Next, you need to export a `GITHUB_TOKEN` environment variable with the `repo` scope selected. This will be used to deploy releases to your GitHub repository. Create yours [here](https://github.com/settings/tokens/new).  
```sh
export GITHUB_TOKEN=`YOUR_TOKEN`
```

GoReleaser uses the latest [Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository,
create a tag:  
```sh
git tag -a v0.1 -m "First release"
```

Now you can run GoReleaser at the root of your repository:  
```sh
curl -s https://raw.githubusercontent.com/goreleaser/get/master/latest | bash
```

That's it!, check your GitHub release page.  
The release on GitHub looks pretty much like this:

[![image](https://cloud.githubusercontent.com/assets/245435/21578845/09404c8a-cf78-11e6-92d7-165ddc03ca6c.png)
](https://github.com/goreleaser/goreleaser/releases)

## Environment setup

### GitHub Token
GoReleaser, requires GitHub api token with the `repo` scope checked. This will help release the artifacts to GitHub. You can create one [here](https://github.com/settings/tokens/new).  
This token should be added to the environment variables as `GITHUB_TOKEN`. Here is how to do it with Travis-ci: [Defining Variables in Repository Settings](https://docs.travis-ci.com/user/environment-variables/#Defining-Variables-in-Repository-Settings).  
You can manually export this variable in your build script:
```sh
export GITHUB_TOKEN=`YOUR_TOKEN`
```
However, exposing your tokens in this method is not recommended.

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

## Release customization

GoReleaser provides multiple customizations, we will cover them with the help of [goreleaser.example.yml](https://github.com/goreleaser/goreleaser/blob/master/goreleaser.example.yml).  
Below is the _goreleaser.yml_ file structure:
```yml
build:
  ...
archive:
  ...
release:
  ...
brew:
  ...
```
### Build
- *main* _(string)_ Path to the main Golang file. By default, the target file is main.go from the current directory.
- *binary_name* _(string)_ Name to be assigned to the binary file in each archive. Default is the name of the project directory.
- *ldflags* _(string)_ Custom Golang ldlags, used in the `go build` command. Default is "-s -w".
- *goos* _(array)_ List of the target operating systems. Default OS are darwin and linux.
- *goarch* _(array)_ List of the target architectures. Default OS are 386 and amd64.

### Archive
- **name_template** _(string)_ Archive name pattern, the following variables are are available: _BinaryName_ ,_Version_ ,_Os_ ,_Arch_. Default is `{{.BinaryName}}_{{.Os}}_{{.Arch}`.
- **format** _(string)_ Archive format, the following variables are are available: _tar.gz_ and _zip_. Default is "zip".
- **replacements** _(map)_ Replacements for GOOS and GOARCH on the archive name. The keys should be valid GOOS or GOARCH values followed by your custom replacements.
- **files** _(array)_ Additional files you want to add to the archive. Defaults are any files matching "LICENCE*", "LICENSE*", "README*" and "CHANGELOG*" (case-insensitive).

### Release
- **repo** _(string)_ Target release repository "username/repository". Default is extracted from the origin remote URL.

### Brew
- **repo** _(string)_ Tap repository "username/homebrew-tap-repository".
- **folder** _(string)_ Folder inside the repository to put the formula. Default is the root folder.
- **caveats** _(string)_ Caveats for the user of your binary. Default is empty.

By defining the _brew_ property, GoReleaser will take care of publishing the Homebrew tap.

```yml
...
brew:
  repo: user/homebrew-tap-repository
  folder: Formula
  caveats: "How to use this binary"
```

For example, the previous config will generate the next formula in the _Formula_ folder of _homebrew-tap-repository_:

```rb
class Release < Formula
  desc "How to use this binary"
  homepage "https://github.com/goreleaser/goreleaser"
  url "path-to-release-file"
  version "current-version"
  sha256 "9ee30fc358fae8d248a2d7538957089885da321dca3f09e3296fe2058e7fff74"

  def install
    bin.install "release"
  end
end
```

## Integration with CI

You may want to wire this to auto-deploy your new tags on [Travis](https://travis-ci.org), for example:

```yaml
# .travis.yml
after_success:
  test -n "$TRAVIS_TAG" && curl -s https://raw.githubusercontent.com/goreleaser/get/master/latest | bash
```

Here is how to do it with [CircleCI](https://circleci.com):
```yml
# circle.yml
deployment:
  master:
    branch: master
    commands:
      - curl -s https://raw.githubusercontent.com/goreleaser/get/master/latest | bash
```

---

Would you like to fix something in the documentation? Feel free to open an [issue](https://github.com/goreleaser/goreleaser/issues).
