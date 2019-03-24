---
title: Quick Start
weight: 10
menu: true
---

In this example we will build, archive and release a sample Go project.

Create a GitHub repository and add a single main package:

```go
// main.go
package main

func main() {
  println("Ba dum, tss!")
}
```

Run `goreleaser init` to create an example `.goreleaser.yaml` file:

```sh
$ goreleaser init

   • Generating .goreleaser.yml file
   • config created; please edit accordingly to your needs file=.goreleaser.yml
```

You can [customize](/customization) the generated `.goreleaser.yml` or leave
it as-is, it's up to you.

You can test the configuration at any time by running GoReleaser with a few
extra parameters to not require a version tag, skip publishing to GitHub,
and remove any already-built files:

```sh
$ goreleaser --snapshot --skip-publish --rm-dist
```

If you are not using vgo or Go modules, then you will need to comment out the
before hooks in the generated config file or update them to match your setup
accordingly.

GoReleaser will build the binaries for your app for Windows, Linux and macOS,
both amd64 and i386 architectures. You can customize that by changing the
`builds` section. Check the [documentation](/build) for more information.

After building the binaries, GoReleaser will create an archive for each OS/Arch
pair into a separate file. You can customize several things by changing
the `archive` section, including releasing only the binaries and not creating
archives at all. Check the [documentation](/archive) for more information.

You'll need to export a `GITHUB_TOKEN` environment variable, which should
contain a valid GitHub token with the `repo` scope.
It will be used to deploy releases to your GitHub repository.
You can create a token [here](https://github.com/settings/tokens/new).

```sh
$ export GITHUB_TOKEN=`YOUR_TOKEN`
```

GoReleaser will use the latest
[Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository.
Create a tag and push it to GitHub:

```sh
$ git tag -a v0.1.0 -m "First release"
$ git push origin v0.1.0
```

> **Attention**: Check if your tag adheres to [semantic versioning](/semver).

If you don't want to create a tag yet, you can also create a release
based on the latest commit by using the `--snapshot` flag.

Now you can run GoReleaser at the root of your repository:

```sh
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

```sh
$ goreleaser release --skip-publish
```

You can check the other options by running:

```sh
$ goreleaser --help
```

and

```sh
$ goreleaser release --help
```
