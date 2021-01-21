# Quick Start

In this example we will build, archive and release a sample Go project.

Create a GitHub repository and add a single main package:

```go
// main.go
package main

func main() {
  println("Ba dum, tss!")
}
```

Run `goreleaser init` to create an example `.goreleaser.yml` file:

```sh
goreleaser init
```

You can [customize](/customization/) the generated `.goreleaser.yml` or leave
it as-is, it's up to you. It is best practice to check `.goreleaser.yml` into the source control.

You can test the configuration at any time by running GoReleaser with a few
extra parameters to not require a version tag, skip publishing to GitHub,
and remove any already-built files:

```sh
goreleaser --snapshot --skip-publish --rm-dist
```

If you are not using vgo or Go modules, then you will need to comment out the
before hooks in the generated config file or update them to match your setup
accordingly.

GoReleaser will build the binaries for your app for Windows, Linux and macOS,
both amd64 and i386 architectures. You can customize that by changing the
`builds` section. Check the [documentation](/customization/build) for more information.

After building the binaries, GoReleaser will create an archive for each OS/Arch
pair into a separate file. You can customize several things by changing
the `archive` section, including releasing only the binaries and not creating
archives at all. Check the [documentation](/customization/archive) for more information.

You'll need to export either a `GITHUB_TOKEN` **or** `GITLAB_TOKEN` environment variable, which should
contain a valid GitHub token with the `repo` scope or GitLab token with `api` scope.
It will be used to deploy releases to your GitHub/GitLab repository.
You can create a token [here](https://github.com/settings/tokens/new) for GitHub or [here](https://gitlab.com/profile/personal_access_tokens) for GitLab.

```sh
export GITHUB_TOKEN="YOUR_GH_TOKEN"
```

or

```sh
export GITLAB_TOKEN="YOUR_GL_TOKEN"
```

GoReleaser will use the latest
[Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository.
Create a tag and push it to GitHub:

```sh
git tag -a v0.1.0 -m "First release"
git push origin v0.1.0
```

!!! info
    Check if your tag adheres to [semantic versioning](/limitations/semver).

If you don't want to create a tag yet, you can also run GoReleaser without publishing
based on the latest commit by using the `--snapshot` flag:

```sh
goreleaser --snapshot
```

Now you can run GoReleaser at the root of your repository:

```sh
goreleaser release
```

That's all!

Check your GitHub project's releases page!

<a href="https://github.com/goreleaser/example/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-github.png"/>
    <figcaption>Example release on GitHub.</figcaption>
  </figure>
</a>

Or, if you released to GitLab, check it out too!

<a href="https://gitlab.com/goreleaser/example/-/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-gitlab.png"/>
    <figcaption>Example release on GitLab.</figcaption>
  </figure>
</a>

!!! note
    Releasing to a private-hosted GitLab CE will only work for version `v11.7+`,
    because the release feature was introduced in this
    [version](https://docs.gitlab.com/ee/user/project/releases/index.html).

## Dry run

If you want to test everything before doing a release "for real", you can
use the following techniques.

### Build-only Mode

Build command will build the project

```sh
goreleaser build
```

This can be useful as part of CI pipelines to verify the project builds
without errors for all build targets.

You can check the other options by running:

```sh
goreleaser build --help
```

### Release Flags

Use the `--skip-publish` flag to skip publishing:

```sh
goreleaser release --skip-publish
```

You can check the other options by running:

```sh
goreleaser --help
```

and

```sh
goreleaser release --help
```
