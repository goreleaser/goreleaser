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

Run the [init](/cmd/goreleaser_init/) command to create an example `.goreleaser.yaml` file:

```sh
goreleaser init
```

Now, lets run a "local-only" release to see if it works using the [release](/cmd/goreleaser_release/) command:

```sh
goreleaser release --snapshot --rm-dist
```

At this point, you can [customize](/customization/) the generated `.goreleaser.yaml` or leave it as-is, it's up to you.
It is best practice to check `.goreleaser.yaml` into the source control.

You can verify your `.goreleaser.yaml` is valid by running the [check](/cmd/goreleaser_check/) command:

```sh
goreleaser check
```

You can also use GoReleaser to [build](/cmd/goreleaser_build/) the binary only for a given GOOS/GOARCH, which is useful for local development:

```sh
goreleaser build --single-target
```

In order to release to GitHub, you'll need to export a `GITHUB_TOKEN` environment variable, which should contain a valid GitHub token with the `repo` scope.
It will be used to deploy releases to your GitHub repository.
You can create a new github token [here](https://github.com/settings/tokens/new).

!!! info
    The minimum permissions the `GITHUB_TOKEN` should have to run this are `write:packages`

```sh
export GITHUB_TOKEN="YOUR_GH_TOKEN"
```

GoReleaser will use the latest [Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository.

Now, create a tag and push it to GitHub:

```sh
git tag -a v0.1.0 -m "First release"
git push origin v0.1.0
```

!!! info
    Check if your tag adheres to [semantic versioning](/limitations/semver/).

!!! info
    If you don't want to create a tag yet, you can also run GoReleaser without publishing based on the latest commit by using the `--snapshot` flag:

    ```sh
    goreleaser release --snapshot
    ```

Now you can run GoReleaser at the root of your repository:

```sh
goreleaser release
```

That's all it takes!

GoReleaser will build the binaries for your app for Windows, Linux and macOS, both amd64 and i386 architectures.
You can customize that by changing the `builds` section. Check the [documentation](/customization/build/) for more information.

After building the binaries, GoReleaser will create an archive for each OS/Arch pair into a separate file.
You can customize several things by changing the `archive` section, including releasing only the binaries and not creating archives at all.
Check the [documentation](/customization/archive/) for more information.

Finally, it will create a release on GitHub with all the artifacts.

Check your GitHub project's releases page!

<a href="https://github.com/goreleaser/example/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-github.png"/>
    <figcaption>Example release on GitHub.</figcaption>
  </figure>
</a>

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
