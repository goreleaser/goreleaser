---
title: Quick Start
weight: 10
menu: true
---

In this example we will build, archive and release a Go project.

Create a GitHub repository and add a single main package:

```go
// main.go
package main

func main() {
  println("Ba dum, tss!")
}
```

Run `goreleaser init` to create an example `.goreleaser.yaml` file:

```console
$ goreleaser init

   • Generating .goreleaser.yml file
   • config created; please edit accordingly to your needs file=.goreleaser.yml
```

The generated config file will look like this:

```yml
# This is an example goreleaser.yaml file with some sane defaults.
# Make sure to check the documentation at http://goreleaser.com
builds:
- env:
  - CGO_ENABLED=0
archive:
  replacements:
    darwin: Darwin
    linux: Linux
    windows: Windows
    386: i386
    amd64: x86_64
checksum:
  name_template: 'checksums.txt'
snapshot:
  name_template: "{{ .Tag }}-next"
changelog:
  sort: asc
  filters:
    exclude:
    - '^docs:'
    - '^test:'
```

GoReleaser will build the binaries for your app for Windows, Linux and macOS,
both amd64 and i386 architectures. You can customize that by changing the
`builds` section. Check the [documentation](/build) for more information.

After building the binaries, GoReleaser will create an archive for each OS/Arch
pair into a separate file. You can customize several things by changing
the `archive` section. Check the [documentation](/archive) for more information.

You'll need to export a `GITHUB_TOKEN` environment variable, which should
contain a valid GitHub token with the `repo` scope.
It will be used to deploy releases to your GitHub repository.
You can create a token [here](https://github.com/settings/tokens/new).

```console
$ export GITHUB_TOKEN=`YOUR_TOKEN`
```

GoReleaser will use the latest
[Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository.
Create a tag and push it to GitHub:

```console
$ git tag -a v0.1.0 -m "First release"
$ git push origin v0.1.0
```

> **Attention**: Check if your tag adheres to [semantic versioning](/semver).

If you don't want to create a tag yet, you can also create a release
based on the latest commit by using the `--snapshot` flag.

Now you can run GoReleaser at the root of your repository:

```console
$ goreleaser
```

That's all! Check your GitHub project's release page.
The release should look like this:

<a href="https://github.com/goreleaser/goreleaser/releases">
  <img width="100%"
    src="https://cloud.githubusercontent.com/assets/245435/23342061/fbcbd506-fc31-11e6-9d2b-4c1b776dee9c.png">
</a>

## Dry run

If you want to test everything before doing a release "for real", you can
use the `--skip-publish` flag, which will only build and package things:

```console
$ goreleaser release --skip-publish
```

You can check the other options by running:

```console
$ goreleaser --help
```

and

```console
$ goreleaser release --help
```
