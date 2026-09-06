---
title: "Build does not contain a main function"
weight: 60
---

This usually happens if you're trying to build a library or if you didn't set up
the `builds.main` section in your `.goreleaser.yaml` and your `main.go` is not
in the root directory.

Here's an example error:

```sh
   ⨯ build failed after 0.11s error=build for foo does not contain a main function

Learn more at https://goreleaser.com/errors/no-main
```

## If you are building a library

Add something like this to your config:

```yaml {filename=".goreleaser.yaml"}
builds:
  - skip: true
```

## If your `main.go` is not in the root directory

Add something like this to your config:

```yaml {filename=".goreleaser.yaml"}
builds:
  - main: ./path/to/your/main/pkg/
```

For more information, check the [builds documentation](/customization/builds/builders/go/).

## If you ran goreleaser outside the root of the project

Run goreleaser in the root of the project.

## If your `main` is an ellipsis path and the package is behind a build tag

Ellipsis discovery uses the host build context, so it can miss packages behind
build tags or OS constraints. Set `main` to the package path instead, and use one
build per package.

For details and examples, see
[Discovery uses the host build context](/customization/builds/builders/go/#discovery-uses-the-host-build-context).

## If you are building in `plugin`, `c-shared` or `c-archive` build modes

You can set `no_main_check` to `true`:

```yaml {filename=".goreleaser.yaml"}
builds:
  - main: ./path/...
    buildmode: plugin
    no_main_check: true
```

For more information, check the [builds documentation](/customization/builds/builders/go/).
