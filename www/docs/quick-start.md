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

Run `goreleaser init` to create an example `.goreleaser.yaml` file:

```console
$ goreleaser init

   • Generating .goreleaser.yml file
   • config created; please edit accordingly to your needs file=.goreleaser.yml
```

You can [customize](/customization) the generated `.goreleaser.yml` or leave
it as-is, it's up to you. It is best practice to check `.goreleaser.yml` into the source control.

You can test the configuration at any time by running GoReleaser with a few
extra parameters to not require a version tag, skip publishing to GitHub,
and remove any already-built files:

```console
$ goreleaser --snapshot --skip-publish --rm-dist
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

```console
$ export GITHUB_TOKEN="YOUR_GH_TOKEN"
# or
$ export GITLAB_TOKEN="YOUR_GL_TOKEN"
```

GoReleaser will use the latest
[Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository.
Create a tag and push it to GitHub:

```console
$ git tag -a v0.1.0 -m "First release"
$ git push origin v0.1.0
```

!!! info
    Check if your tag adheres to [semantic versioning](/limitations/semver).

If you don't want to create a tag yet, you can also run GoReleaser without publishing
based on the latest commit by using the `--snapshot` flag:

```console
$ goreleaser --snapshot
```

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

Or check your GitLab project's release page.
The release should also look like this:
<a href="https://gitlab.com/mavogel/release-testing/-/releases">
  <img width="100%"
    src="https://user-images.githubusercontent.com/8409778/59390011-55fcdf80-8d70-11e9-840f-c568ddc0e965.png">
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

```console
$ goreleaser build
```

This can be useful as part of CI pipelines to verify the project builds
without errors for all build targets.

You can check the other options by running:

```console
$ goreleaser build --help
```

### Release Flags

Use the `--skip-publish` flag to skip publishing:

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
