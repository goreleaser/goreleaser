---
title: Node.js
weight: 25
---

<!-- markdownlint-disable MD025 -->

{{< experimental >}}

You can build Node.js [Single Executable Application][sea] (SEA) binaries
with GoReleaser, in pure Go — no `postject` or any other npm tooling
required. The only external runtime requirement is `node` (>= 22) on the
machine running GoReleaser, used to generate the SEA preparation blob.

[sea]: https://nodejs.org/api/single-executable-applications.html

## How it works

For each requested target, GoReleaser:

1. Resolves the Node.js version to use (see [Version
   resolution](#version-resolution)).
2. Downloads the official Node.js host binary for that target from
   <https://nodejs.org/dist>, verifying its SHA-256 against the matching
   `SHASUMS256.txt` entry, and caches it under
   `${XDG_CACHE_HOME:-$HOME/.cache}/goreleaser/node/` so subsequent
   builds are offline.
3. Strips the existing code signature from the host binary on macOS and
   Windows (no-op on Linux).
4. Generates a minimal `sea-config.json` on the fly (pointing at
   `builds.main`, with `disableExperimentalSEAWarning: true`) and runs
   `node --experimental-sea-config` against it to produce the SEA blob.
5. Copies the prepared host binary to the build output path and injects
   the blob (ELF section / Mach-O segment / PE resource), flipping the
   `NODE_SEA_FUSE_…` sentinel so Node.js loads the embedded
   application.

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

    # Hooks can be used to customize the final binary, for example to
    # run generators or sign the produced executable.
    #
    # Templates: allowed.
    hooks:
      pre: npm run bundle
      post: ./script.sh {{ .Path }}

    # If true, skip the build.
    skip: false
```

The following standard build fields are intentionally **not** supported
by the `node` builder:

- `tool`, `command`, `flags` — the SEA pipeline invokes `node`
  directly with a known set of arguments.

## Version resolution

The Node.js version used for both blob generation and the bundled host
binary is resolved in this order:

1. The `engines.node` field in `package.json` (highest matching official
   release).
2. A `.nvmrc` file in the build directory.
3. A `.node-version` file in the build directory.

Either an exact version (`v22.10.0`, `22.10.0`) or a semver range
(`>=22 <23`, `^22`) is accepted. Ranges are resolved against the
nodejs.org release index.

## Code signing

The produced macOS and Windows binaries are **unsigned**. macOS in
particular will refuse to run an unsigned binary. Wire up the existing
[`signs`](/customization/sign/) pipe (Apple `codesign`/`notarytool`,
Microsoft `signtool`, etc.) to sign the binary after the build completes.

## Environment setup

GoReleaser will not install Node.js for you. Make sure `node` (>= 22) is
available on `PATH` before running GoReleaser. The downloaded host
binaries are cached and reused across builds.

{{< g_templates >}}
