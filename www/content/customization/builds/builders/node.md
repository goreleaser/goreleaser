---
title: Node.js
weight: 25
---

{{< g_experimental "https://github.com/goreleaser/goreleaser/pull/6579" >}}

You can build Node.js [Single Executable Application][sea] (SEA) binaries
with GoReleaser!

> [!WARNING]
> Only Node ≥ v25.5.0 is supported.

[sea]: https://nodejs.org/api/single-executable-applications.html

## Configuration

Simply set the `builder` to `node`, for instance:

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

    # Set a specific node binary to use when building the SEA bundle.
    # It is safe to ignore this option in most cases.
    #
    # Default: "node".
    # Templates: allowed.
    tool: "node-nightly"

    # Custom environment variables to set when invoking node.
    # Invalid environment variables will be ignored.
    #
    # Default: os.Environ() ++ env config section.
    # Templates: allowed.
    env:
      - FOO=bar

    # Hooks can be used to customize the final binary, for example to
    # bundle the entrypoint or sign the produced executable.
    #
    # Templates: allowed.
    hooks:
      post: ./script.sh {{ .Path }}

    # If true, skip the build.
    skip: false
```

### Environment setup

GoReleaser will not install Node.js, project dependencies, or run your
JavaScript build for you. Run them before GoReleaser, usually with a global
[`before` hook](/customization/general/hooks/):

```yaml {filename=".goreleaser.yaml"}
before:
  hooks:
    - npm ci
    - npm run build
```

Do not write your JavaScript build output to the `dist` directory.
GoReleaser requires that directory to be empty after
`before` hooks run. Configure your bundler to write somewhere else, for
example `build/`, and point `main` at the generated entrypoint.

## Caveats

The following standard build fields are intentionally **not** supported
by the `node` builder:

- `command`, `flags` — the SEA pipeline invokes `node`
  directly with a known set of arguments.

The following template variables are available in the per-target build
context: `.Os`, `.Arch`, `.Goos`, `.Goarch`, `.Target`, `.Name`,
`.Path`, `.Ext`, `.Env.*`.
Use them in `main`, `env`, and the `hooks` recipes.

## Tuning the SEA blob (`sea-config.json`)

Drop a `sea-config.json` file alongside your `package.json` (i.e. in
the build's `dir`) and GoReleaser will merge it with its own
GoReleaser-owned fields before invoking `node --build-sea`:

```json {filename="sea-config.json"}
{
  "assets": {
    "schema.json": "./schema.json"
  },
  "execArgv": ["--max-old-space-size=4096"],
  "disableExperimentalSEAWarning": true
}
```

GoReleaser always overwrites `output`, `executable`, `main` — these point at
internal cache paths, so any user-supplied values are ignored.
Relative paths under `assets` are anchored at the build directory so
they keep resolving after GoReleaser moves the merged configuration into a
scratch directory.

When no `sea-config.json` is present, GoReleaser generates the minimum
config needed to drive `node --build-sea`.
See Node's [Single Executable Applications docs][sea] for the full list of
accepted fields.

## Version resolution

The target Node.js version comes from `engines.node` in `package.json`.
Exact versions (`v25.5.0`, `25.5.0`) and ranges (`>=25.5 <26`, `^25`)
are accepted.
Pin an exact version for reproducible release artifacts.

{{< g_templates >}}
