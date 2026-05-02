---
title: "Quick Start"
weight: 30
---

In this example we will build, archive and release a sample project.

Create a GitHub repository, clone and `cd` into it, and let's get started!

{{< tabs >}}
{{< tab name="Go" icon="go" >}}

Initialize your module with:

```sh
go mod init github.com/you/your-repo
```

Then create a `main.go` file:

```go {filename="main.go"}
package main

func main() {
  println("Ba dum, tss!")
}
```

{{< /tab >}}
{{< tab name="Rust" icon="rust" >}}

Initialize your project with:

```sh
cargo init --bin
```

{{< /tab >}}
{{< tab name="Node.js" icon="node" >}}

Initialize your project with:

```sh
npm init -y
npm pkg set engines.node=">=25.5 <26"
npm install
```

Then create an `index.js` file:

```js {filename="index.js"}
console.log("Ba dum, tss!");
```

{{< /tab >}}
{{< tab name="Zig" icon="zig" >}}

Initialize your project with:

```sh
zig init
```

{{< /tab >}}
{{< tab name="Bun" icon="bun" >}}

Initialize your project with:

```sh
bun init
```

{{< /tab >}}
{{< tab name="Deno" icon="deno" >}}

Initialize your project with:

```sh
deno init
```

{{< /tab >}}
{{< tab name="UV" icon="uv" >}}

Initialize your project with:

```sh
uv init
```

{{< /tab >}}
{{< tab name="Poetry" icon="poetry" >}}

Initialize your project with:

```sh
poetry new .
```

{{< /tab >}}
{{< /tabs >}}

Run the init command to create an example `.goreleaser.yaml` file:

```sh
goreleaser init
```

Now, lets run a "local-only" release to see if it works using the release command:

```sh
goreleaser release --snapshot --clean
```

At this point, you can [customize](/customization/index/) the generated `.goreleaser.yaml` or leave it as-is, it's up to you.
It is best practice to check `.goreleaser.yaml` into the source control.

You can verify your `.goreleaser.yaml` is valid by running the check command:

```sh
goreleaser check
```

You can also use GoReleaser to build the binary only for a given target, which is useful for local development:

{{< tabs >}}
{{< tab name="Go" icon="go" >}}

```sh
GOOS="linux" \
GOARCH="arm64" \
  goreleaser build --single-target
```

It will default to your current `GOOS`/`GOARCH`.
{{< /tab >}}
{{< tab name="Rust" icon="rust" >}}

```sh
TARGET="aarch64-unknown-linux-gnu" \
  goreleaser build --single-target
```

{{< /tab >}}
{{< tab name="Node.js" icon="node" >}}

```sh
TARGET="linux-arm64" \
  goreleaser build --single-target
```

{{< /tab >}}
{{< tab name="Zig" icon="zig" >}}

```sh
TARGET="aarch64-linux" \
  goreleaser build --single-target
```

{{< /tab >}}
{{< tab name="Bun" icon="bun" >}}

```sh
TARGET="bun-linux-arm64" \
  goreleaser build --single-target
```

{{< /tab >}}
{{< tab name="Deno" icon="deno" >}}

```sh
TARGET="aarch64-unknown-linux-gnu" \
  goreleaser build --single-target
```

{{< /tab >}}
{{< tab name="UV" icon="uv" >}}

```sh
TARGET="py3-none-any" \
  goreleaser build --single-target
```

{{< /tab >}}
{{< tab name="Poetry" icon="poetry" >}}

```sh
TARGET="py3-none-any" \
  goreleaser build --single-target
```

{{< /tab >}}
{{< /tabs >}}

To release to GitHub, you'll need to export a `GITHUB_TOKEN` environment variable, which should contain a valid GitHub token with the `repo` scope.
It will be used to deploy releases to your GitHub repository.
You can create a new GitHub token [here](https://github.com/settings/tokens/new?scopes=repo,write:packages).

> [!NOTE]
> The minimum permissions the `GITHUB_TOKEN` should have to run this are `write:packages`

```sh
export GITHUB_TOKEN="YOUR_GH_TOKEN"
```

GoReleaser will use the latest [Git tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging) of your repository.

Now, create a tag and push it to GitHub:

```sh
git tag -a v0.1.0 -m "First release"
git push origin v0.1.0
```

> [!NOTE]
> Check if your tag adheres to [semantic versioning](/resources/limitations/semver/).

> [!NOTE]
> If you don't want to create a tag yet, you can also run GoReleaser without
> publishing based on the latest commit by using the `--snapshot` flag:
>
> ```sh
> goreleaser release --snapshot
> ```

Now you can run GoReleaser at the root of your repository:

```sh
goreleaser release
```

That's all it takes!

GoReleaser will build the binaries for your app for the default targets for the
build mechanism being used.
You can customize that by changing the `builds` section.
Check the [documentation](/customization/builds/) for more information.

After building the binaries, GoReleaser will create an archive for each target into a separate file.
You can customize several things by changing the `archives` section, including releasing only the binaries and not creating archives at all.
Check the [documentation](/customization/package/archives/) for more information.

Finally, it will create a release on GitHub with all the artifacts.

Check your GitHub project's releases page!

## Live examples

We have a ton of example repositories!
You can use them to learn more and see how GoReleaser works.

<br>
{{< g_button href="https://github.com/orgs/goreleaser/repositories?q=example" label="Browse example repositories" icon="github" primary="true" >}}

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

You can check the command line usage help here or with:

```sh
goreleaser --help
```
