---
title: Quick Start
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

By default GoReleaser will build the current directory, but you can change
the package path in the GoReleaser configuration file:

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

GoReleaser skips invalid GOOS/GOARCH combinations.

With the above configuration the name of all created binaries will be `drum-roll`
and GoReleaser will build one binary in 64bit architecture for each of the operating systems Windows, Linux and MacOS.

GoReleaser will then archive the resulting binaries of each OS/Arch pair into a
separate file. The default format is `{{.ProjectName}}_{{.Os}}_{{.Arch}}`.
You can change the archive's name and format. You can also replace the OS
and the Architecture with your own.

Another useful feature is to add additional files to the created archives:

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

This configuration will generate `tar` archives, each containing an additional
file called `drum-roll.licence.txt`.
The archives will be located in the `dist` folder:

- `./dist/drum-roll_windows_64-bit.tar.gz`
- `./dist/drum-roll_macOS_64-bit.tar.gz`
- `./dist/drum-roll_Tux_64-bit.tar.gz`

Next, you need to export a `GITHUB_TOKEN` environment variable, which should contain a
GitHub token with the `repo` scope selected.
It will be used to deploy releases to your GitHub repository.
Create a token [here](https://github.com/settings/tokens/new).

```console
$ export GITHUB_TOKEN=`YOUR_TOKEN`
```

GoReleaser uses the latest
[Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository.
Create a tag and push it to GitHub:

```console
$ git tag -a v0.1.0 -m "First release"
$ git push origin v0.1.0
```

**Note**: We recommend the use of [semantic versioning](http://semver.org/). We
are not enforcing it though. We do remove the `v` prefix and then enforce
that the next character is a number. So, `v0.1.0` and `0.1.0` are virtually the
same and both are accepted, while `version0.1.0` is not.

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
