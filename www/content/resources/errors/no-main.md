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

GoReleaser discovers `main` packages using the environment it runs in. It does
not apply the `tags` and `flags` of the build, nor the `GOOS` and `GOARCH` of
each target, so a package guarded by a build constraint is not found:

```yaml {filename=".goreleaser.yaml"}
builds:
  - main: ./... # cmd/feature/main.go has `//go:build feature`
    tags: [feature]
```

Set `main` to the package path instead, and use one build per package:

```yaml {filename=".goreleaser.yaml"}
builds:
  - id: feature
    main: ./cmd/feature
    tags: [feature]
```

The same applies to a `main` package that only builds on one operating system,
for example one with `//go:build linux` when you release from macOS.

## If you are building in `plugin`, `c-shared` or `c-archive` build modes

You can set `no_main_check` to `true`:

```yaml {filename=".goreleaser.yaml"}
builds:
  - main: ./path/...
    buildmode: plugin
    no_main_check: true
```

For more information, check the [builds documentation](/customization/builds/builders/go/).
