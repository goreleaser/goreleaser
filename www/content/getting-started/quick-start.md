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

Now, let's run a "local-only" release to see if it works, using the release
command:

```sh
goreleaser release --snapshot --clean
```

`--clean` removes files from previous runs in the distribution directory
(`dist` by default). Use it for each subsequent build or release in this guide.

At this point, you can [customize](/customization/index/) the generated
`.goreleaser.yaml` or leave it as-is — it's up to you.
Either way, check `.goreleaser.yaml` into source control.

You can verify your `.goreleaser.yaml` is valid by running the check command:

```sh
goreleaser check
```

You can also use GoReleaser to build the binary for a single target only, which
is useful for local development:

{{< tabs >}}
{{< tab name="Go" icon="go" >}}

```sh
GOOS="linux" \
GOARCH="arm64" \
  goreleaser build --single-target --clean
```

It will default to your current `GOOS`/`GOARCH`.
{{< /tab >}}
{{< tab name="Rust" icon="rust" >}}

```sh
TARGET="aarch64-unknown-linux-gnu" \
  goreleaser build --single-target --clean
```

{{< /tab >}}
{{< tab name="Node.js" icon="node" >}}

```sh
TARGET="linux-arm64" \
  goreleaser build --single-target --clean
```

{{< /tab >}}
{{< tab name="Zig" icon="zig" >}}

```sh
TARGET="aarch64-linux" \
  goreleaser build --single-target --clean
```

{{< /tab >}}
{{< tab name="Bun" icon="bun" >}}

```sh
TARGET="bun-linux-arm64" \
  goreleaser build --single-target --clean
```

{{< /tab >}}
{{< tab name="Deno" icon="deno" >}}

```sh
TARGET="aarch64-unknown-linux-gnu" \
  goreleaser build --single-target --clean
```

{{< /tab >}}
{{< tab name="UV" icon="uv" >}}

```sh
TARGET="none-any" \
  goreleaser build --single-target --clean
```

{{< /tab >}}
{{< tab name="Poetry" icon="poetry" >}}

```sh
TARGET="none-any" \
  goreleaser build --single-target --clean
```

{{< /tab >}}
{{< /tabs >}}

To release to GitHub, export a `GITHUB_TOKEN` environment variable containing a
GitHub token that can create releases in your repository.
A classic personal access token needs the `repo` scope; a fine-grained token
needs `contents: write` permission on the repository.
You can [create a new classic token](https://github.com/settings/tokens/new?scopes=repo)
with the right scope pre-selected.

> [!NOTE]
> Add the `write:packages` scope as well if you also push Docker images to the
> GitHub Container Registry.
> See the [GitHub Actions documentation](/customization/ci/actions/#token-permissions)
> for the full list of permissions each feature needs.

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
> goreleaser release --snapshot --clean
> ```

Now you can run GoReleaser at the root of your repository:

```sh
goreleaser release --clean
```

That's all it takes!

GoReleaser will build the binaries for your app for the default targets of the
builder being used.
You can customize that by changing the `builds` section.
Check the [builds documentation](/customization/builds/) for more information.

After building the binaries, GoReleaser will create a separate archive for each
target.
You can customize several things by changing the `archives` section, including
releasing only the binaries and not creating archives at all.
Check the [archives documentation](/customization/package/archives/) for more
information.

Finally, it will create a release on GitHub with all the artifacts.

Check your GitHub project's releases page!

## Live examples

We maintain many example repositories.
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

The build command builds the project without packaging or publishing it:

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

Every command documents its own flags.
Run the help for any of them:

```sh
goreleaser --help
goreleaser release --help
```
