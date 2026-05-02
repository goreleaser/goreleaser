---
title: Node.js
weight: 25
---

{{< g_experimental "https://github.com/goreleaser/goreleaser/pull/6579" >}}

You can build Node.js [Single Executable Application][sea] (SEA) binaries
with GoReleaser, in pure Go — no `postject` or any other npm tooling
required. GoReleaser shells out to whatever `node` is on your `PATH` to
invoke `node --build-sea`, and downloads the per-target Node.js binary
that becomes the SEA executable.

The `--build-sea` subcommand exists only in Node.js ≥ v25.5.0 built with
LIEF — make sure the `node` on your `PATH` is recent enough; if it is
not, the underlying `node --build-sea` failure is surfaced.

[sea]: https://nodejs.org/api/single-executable-applications.html

## Quick start

For a brand-new project:

```sh
goreleaser init --language node
```

This drops a starter `.goreleaser.yaml` next to your `package.json`.
For an existing project, copy the snippet under
[Configuration](#configuration) into your config and adjust `binary`,
`main`, and `targets`.

## How it works

For each requested target, GoReleaser:

1. Resolves the target Node.js version from `engines.node` in
   `package.json` (see [Version resolution](#version-resolution)).
2. Downloads the official Node.js binary for that target, verifying
   its SHA-256 against the release index embedded in GoReleaser.
3. Merges your `sea-config.json` (if present) with the
   goreleaser-owned fields and runs
   `node --build-sea sea-config.json` to produce the final binary.

## Bundling your app

GoReleaser does **not** bundle your `node_modules/` for you, but it
does run `npm run build` automatically when your `package.json`
declares a `scripts.build` entry — so the file referenced by
[`main`](#configuration) is the freshly bundled output.

```json {filename="package.json"}
{
  "scripts": {
    "build": "esbuild src/index.js --bundle --platform=node --outfile=dist/bundle.js"
  }
}
```

```yaml {filename=".goreleaser.yaml"}
builds:
  - builder: node
    main: dist/bundle.js
```

That's it — no GoReleaser config needed. When `scripts.build` is
absent, the step is skipped silently and GoReleaser uses
[`main`](#configuration) as-is.

Dependency installation (`npm ci`, `pnpm install --frozen-lockfile`,
…) is **not** run for you — drive it from the global
[`before`](/customization/global_hooks/) hook so it executes once per
release rather than once per build:

```yaml {filename=".goreleaser.yaml"}
before:
  hooks:
    - npm ci

builds:
  - builder: node
    main: dist/bundle.js
```

If you need per-target bundling (e.g. different output for darwin
vs. linux), bypass the auto-step by giving your `package.json` no
`scripts.build` entry and drive the bundling from a per-target
[`hooks.pre`](/customization/build/) instead.

## Configuration

```yaml {filename=".goreleaser.yaml"}
builds:
  - id: my-build
    builder: node

    # Binary name.
    # Can be a path (e.g. `bin/app`) to wrap the binary in a directory.
    #
    # Default: Project directory name.
    binary: program

    # Entrypoint script bundled into the SEA blob.
    #
    # Default: 'index.js'.
    # Templates: allowed.
    main: index.js

    # Targets, in nodejs.org/dist format.
    # Default: all of: darwin-arm64, darwin-x64, linux-arm64, linux-x64,
    #                  win-arm64, win-x64.
    targets:
      - linux-x64
      - darwin-arm64

    # Path to the project's (sub)directory containing the code and
    # (typically) package.json.
    #
    # Default: '.'.
    dir: my-app

    # Custom environment variables to set when invoking node.
    # Invalid environment variables will be ignored.
    #
    # Default: os.Environ() ++ env config section.
    # Templates: allowed.
    env:
      - FOO=bar

    # Auto-bundle step. Runs `npm run build` in `dir` before invoking
    # `node --build-sea`, when `package.json` declares a non-empty
    # `scripts.build` entry. Silent skip otherwise. See "Bundling your
    # app" above for details. Dependency installation (`npm ci` and
    # friends) is intentionally not performed — drive it from the
    # global `before` hook.

    # Hooks can be used to customize the final binary, for example to
    # bundle the entrypoint or sign the produced executable.
    #
    # Templates: allowed.
    hooks:
      pre: npx esbuild src/index.js --bundle --platform=node --outfile=dist/bundle.js
      post: ./script.sh {{ .Path }}

    # If true, skip the build.
    skip: false
```

The following standard build fields are intentionally **not** supported
by the `node` builder:

- `tool`, `command`, `flags` — the SEA pipeline invokes `node`
  directly with a known set of arguments.

The following template variables are available in the per-target build
context: `.Os`, `.Arch`, `.Goos`, `.Goarch`, `.Target`, `.Name`,
`.Path`, `.Ext`, `.Env.*`. Use them in `main`, `env`,
and the `hooks` recipes.

## Tuning the SEA blob (`sea-config.json`)

GoReleaser does not expose `sea-config.json` knobs in `.goreleaser.yaml`.
Drop a `sea-config.json` file alongside your `package.json` (i.e. in
the build's `dir`) and GoReleaser will merge it with its own
goreleaser-owned fields before invoking `node --build-sea`:

```json {filename="sea-config.json"}
{
  "assets": {
    "icon.png": "./assets/icon.png",
    "schema.json": "./schema.json"
  },
  "execArgv": ["--max-old-space-size=4096"],
  "disableExperimentalSEAWarning": true,
  "mainFormat": "commonjs"
}
```

GoReleaser always overwrites `output`, `executable`, `main`,
`useCodeCache`, and `useSnapshot` — these point at internal cache
paths and scratch tempfiles, so any user-supplied values are ignored.
Relative paths under `assets` are anchored at the build directory so
they keep resolving after GoReleaser moves the merged config into a
scratch directory.

When no `sea-config.json` is present, GoReleaser generates the
minimum config needed to drive `node --build-sea`.

See Node's [Single Executable Applications docs][sea] for the full
list of accepted fields.

## Version resolution

The target Node.js version (the binary that becomes the SEA
executable) is resolved exclusively from the `engines.node` field of
the build directory's `package.json`. Either an exact version
(`v25.5.0`, `25.5.0`) or a semver range (`>=25.5 <26`, `^25`) is
accepted; ranges resolve to the highest matching release in the
embedded nodedist index. Pin to an exact version for reproducible
release artifacts.

Only Node ≥ v25.5.0 is supported — that is the floor where the
official Node.js builds ship `--build-sea` (LIEF-backed). Older
releases are rejected at resolve time.

{{< g_templates >}}
