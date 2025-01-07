# Quick Start

In this example we will build, archive and release a sample project.

Create a GitHub repository, clone and `cd` into it, and let's get started!

=== ":simple-go: Go"

    Initialize your module with:

    ```sh
    go mod init github.com/you/your-repo
    ```

    Then create a `main.go` file:

    ```go title="main.go"
    package main

    func main() {
      println("Ba dum, tss!")
    }
    ```

=== ":simple-rust: Rust"

    Initialize your project with:

    ```sh
    cargo init --bin
    ```

=== ":simple-zig: Zig"

    Initialize your project with:

    ```sh
    zig init
    ```

=== ":simple-bun: Bun"

    Initialize your project with:

    ```sh
    bun init
    ```

=== ":simple-deno: Deno"

    Initialize your project with:

    ```sh
    deno init
    ```

Run the [init](cmd/goreleaser_init.md) command to create an example `.goreleaser.yaml` file:

```sh
goreleaser init
```

Now, lets run a "local-only" release to see if it works using the [release](cmd/goreleaser_release.md) command:

```sh
goreleaser release --snapshot --clean
```

At this point, you can [customize](customization/index.md) the generated `.goreleaser.yaml` or leave it as-is, it's up to you.
It is best practice to check `.goreleaser.yaml` into the source control.

You can verify your `.goreleaser.yaml` is valid by running the [check](cmd/goreleaser_check.md) command:

```sh
goreleaser check
```

You can also use GoReleaser to [build](cmd/goreleaser_build.md) the binary only for a given target, which is useful for local development:

=== ":simple-go: Go"

    ```sh
    GOOS="linux" \
    GOARCH="arm64" \
      goreleaser build --single-target
    ```

    It will default to your current `GOOS`/`GOARCH`.

=== ":simple-rust: Rust"

    ```sh
    TARGET="aarch64-unknown-linux-gnu" \
      goreleaser build --single-target
    ```

=== ":simple-zig: Zig"

    ```sh
    TARGET="aarch64-linux" \
      goreleaser build --single-target
    ```

=== ":simple-bun: Bun"

    ```sh
    TARGET="bun-linux-arm64" \
      goreleaser build --single-target
    ```

=== ":simple-deno: Deno"

    ```sh
    TARGET="aarch64-unknown-linux-gnu" \
      goreleaser build --single-target
    ```

To release to GitHub, you'll need to export a `GITHUB_TOKEN` environment variable, which should contain a valid GitHub token with the `repo` scope.
It will be used to deploy releases to your GitHub repository.
You can create a new GitHub token [here](https://github.com/settings/tokens/new?scopes=repo,write:packages).

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

    Check if your tag adheres to [semantic versioning](limitations/semver.md).

!!! info

    If you don't want to create a tag yet, you can also run GoReleaser without
    publishing based on the latest commit by using the `--snapshot` flag:

    ```sh
    goreleaser release --snapshot
    ```

Now you can run GoReleaser at the root of your repository:

```sh
goreleaser release
```

That's all it takes!

GoReleaser will build the binaries for your app for the default targets for the
build mechanism being used.
You can customize that by changing the `builds` section.
Check the [documentation](customization/builds/index.md) for more information.

After building the binaries, GoReleaser will create an archive for each target into a separate file.
You can customize several things by changing the `archives` section, including releasing only the binaries and not creating archives at all.
Check the [documentation](customization/archive.md) for more information.

Finally, it will create a release on GitHub with all the artifacts.

Check your GitHub project's releases page!

<a href="https://github.com/goreleaser/example/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-github.png"/>
    <figcaption>Example release on GitHub.</figcaption>
  </figure>
</a>

## Live examples

We have a ton of example repositories!
You can use them to learn more and see how GoReleaser works.

[Browse example repositories](https://github.com/orgs/goreleaser/repositories?q=example){ .md-button .md-button--primary }

## Dry run

If you want to test everything before doing a release "for real", you can
use the following techniques.

### Verify dependencies

You can check if you have every tool needed for the current configuration:

```sh
goreleaser healthcheck
```

### Build-only Mode

Build command will build the project:

```sh
goreleaser build
```

This can be useful as part of CI pipelines to verify the project builds
without errors for all build targets.

### Release Flags

Use the `--skip=publish` flag to skip publishing:

```sh
goreleaser release --skip=publish
```

### More options

You can check the command line usage help [here](./cmd/goreleaser.md) or with:

```sh
goreleaser --help
```
